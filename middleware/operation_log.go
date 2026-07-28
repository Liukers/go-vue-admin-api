package middleware

import (
	"bytes"
	"encoding/json"
	"go-vue-admin/global"
	"go-vue-admin/models"
	v1 "go-vue-admin/services/v1"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 操作日志异步队列（缓冲区满时丢弃并告警，不阻塞请求）
var operationLogChan = make(chan models.OperationLog, 500)

func init() {
	// 启动操作日志工作协程，异步写库
	for i := 0; i < 3; i++ {
		go func() {
			for log := range operationLogChan {
				if err := global.DB.Create(&log).Error; err != nil {
					global.Log.Errorf("保存操作日志失败: %v", err)
				}
			}
		}()
	}
}

// WaitOperationLogsFlushed 等待操作日志队列消费完毕（用于优雅停机）
func WaitOperationLogsFlushed(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for len(operationLogChan) > 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
}

// 敏感字段名（请求数据中出现时将其值脱敏为 ***）
var sensitiveDataKeys = map[string]bool{
	"password":        true,
	"oldPassword":     true,
	"newPassword":     true,
	"confirmPassword": true,
}

const (
	// maxLogBodySize 捕获请求体用于日志的大小上限（超过则省略，避免内存放大）
	maxLogBodySize = 64 * 1024
	// maxLogDataLen 请求/响应数据入库前的最大字符数
	maxLogDataLen = 1000
)

// truncateString 按字符数截断字符串（使用 rune 避免截断 UTF-8 字符）
func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

// 角色名称缓存（角色名极少变更，仅用于日志展示，允许短暂不一致）
var roleNameCache sync.Map // map[uint]string

// OperationLog 操作日志中间件
func OperationLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否开启操作日志
		var settingService v1.SystemSettingService
		if !settingService.IsOperationLogEnabled() {
			c.Next()
			return
		}

		startTime := time.Now()

		var requestData string

		// GET 请求记录查询参数（敏感参数脱敏），其他请求记录 Body（敏感字段脱敏）
		if c.Request.Method == "GET" {
			query := c.Request.URL.Query()
			for key := range query {
				if sensitiveDataKeys[key] {
					query.Set(key, "***")
				}
			}
			if encoded := query.Encode(); encoded != "" {
				requestData = "[Query] " + encoded
			}
		} else if c.Request.Body != nil {
			// 仅在 Content-Length 明确且不超过上限时捕获请求体用于日志，
			// 超大请求不读取（交由后续 handler 正常消费流），避免内存放大；
			// 捕获内容同样按长度截断并脱敏
			if c.Request.ContentLength > 0 && c.Request.ContentLength <= maxLogBodySize {
				bodyBytes, err := io.ReadAll(c.Request.Body)
				if err != nil {
					global.Log.Warnf("读取请求体失败: %v", err)
				}
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				requestData = truncateString(maskSensitiveJSON(string(bodyBytes)), maxLogDataLen)
			} else if c.Request.ContentLength != 0 {
				requestData = "[请求体过大，已省略]"
			}
		}

		// 创建自定义ResponseWriter来捕获响应
		blw := &bodyLogWriter{
			body:           bytes.NewBufferString(""),
			ResponseWriter: c.Writer,
			mu:             &sync.Mutex{},
		}
		c.Writer = blw

		c.Next()

		duration := time.Since(startTime).Milliseconds()

		userId, _ := c.Get("userId")
		username, _ := c.Get("username")
		roleId, _ := c.Get("roleId")

		status := 1 // 成功
		if c.Writer.Status() >= 400 {
			status = 2 // 失败
		}

		responseData := blw.body.String()

		// 对日志查询接口本身，不记录响应数据（避免套娃）
		path := c.Request.URL.Path
		if path == "/api/v1/system/operation-logs" || path == "/api/v1/system/login-logs" {
			responseData = "[日志列表数据省略]"
		} else {
			responseData = truncateString(responseData, maxLogDataLen)
		}

		log := models.OperationLog{
			UserID:        getUint(userId),
			Username:      getString(username),
			RoleName:      getRoleName(roleId),
			Method:        c.Request.Method,
			Path:          path,
			RequestData:   requestData,
			ResponseData:  responseData,
			Status:        status,
			ErrorMessage:  getErrorMessage(c),
			IP:            c.ClientIP(),
			UserAgent:     c.Request.UserAgent(),
			OperationTime: int(duration),
			CreatedAt:     models.LocalTime(time.Now()),
		}

		// 异步入库，不阻塞响应；队列满时丢弃并告警
		select {
		case operationLogChan <- log:
		default:
			global.Log.Warn("操作日志队列已满，丢弃日志记录")
		}
	}
}

// getRoleName 获取角色名称（带缓存）
func getRoleName(roleId interface{}) string {
	rid, ok := roleId.(uint)
	if !ok || rid == 0 {
		return ""
	}
	if cached, ok := roleNameCache.Load(rid); ok {
		return cached.(string)
	}
	var role models.SystemRole
	if err := global.DB.First(&role, rid).Error; err != nil {
		global.Log.Warnf("查询角色失败: %v", err)
		return ""
	}
	roleNameCache.Store(rid, role.RoleName)
	return role.RoleName
}

// maskSensitiveJSON 对JSON请求体中的敏感字段值脱敏（替换为 ***，保留其余内容）
// 非JSON数据（如表单提交）只要包含敏感字段名则整体脱敏
func maskSensitiveJSON(data string) string {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return data
	}

	var v interface{}
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil {
		// 非JSON请求体：包含敏感字段名时整体脱敏
		for key := range sensitiveDataKeys {
			if strings.Contains(data, `"`+key+`"`) || strings.Contains(data, key+"=") {
				return "[FILTERED]"
			}
		}
		return data
	}

	maskSensitiveValue(v)
	out, err := json.Marshal(v)
	if err != nil {
		return data
	}
	return string(out)
}

// maskSensitiveValue 递归脱敏map/slice中的敏感字段
func maskSensitiveValue(v interface{}) {
	switch t := v.(type) {
	case map[string]interface{}:
		for key, val := range t {
			if sensitiveDataKeys[key] {
				t[key] = "***"
			} else {
				maskSensitiveValue(val)
			}
		}
	case []interface{}:
		for _, item := range t {
			maskSensitiveValue(item)
		}
	}
}

// bodyLogWriter 自定义ResponseWriter（线程安全）
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
	mu   *sync.Mutex
}

// Write 实现Write方法（线程安全）
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	w.body.Write(b)
	w.mu.Unlock()
	return w.ResponseWriter.Write(b)
}

// getUint 安全获取uint
func getUint(v interface{}) uint {
	if v == nil {
		return 0
	}
	if id, ok := v.(uint); ok {
		return id
	}
	return 0
}

// getString 安全获取string
func getString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// getErrorMessage 获取错误信息
func getErrorMessage(c *gin.Context) string {
	if len(c.Errors) > 0 {
		return c.Errors.String()
	}
	return ""
}
