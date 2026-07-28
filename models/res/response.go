package res

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"go-vue-admin/global"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Response 统一响应结构体
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: SuccessMsg,
		Data:    data,
	})
}

// SuccessWithMessage 成功响应（自定义消息）
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: message,
		Data:    data,
	})
}

// Fail 失败响应
func Fail(c *gin.Context, code int) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: GetErrorMsg(code),
		Data:    nil,
	})
}

// FailWithMessage 失败响应（自定义消息）
func FailWithMessage(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// FailWithData 失败响应（带数据）
func FailWithData(c *gin.Context, code int, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: GetErrorMsg(code),
		Data:    data,
	})
}

// Error 错误响应（内部错误，不暴露详细信息给客户端）
func Error(c *gin.Context, err error) {
	global.Log.Error("请求处理失败: ", err)
	if appErr, ok := err.(*AppError); ok {
		c.JSON(http.StatusOK, Response{
			Code:    appErr.Code,
			Message: appErr.Message,
			Data:    nil,
		})
		return
	}
	// 普通错误不暴露内部信息
	c.JSON(http.StatusOK, Response{
		Code:    ErrorCodeInternalServer,
		Message: ErrorMsgInternalServer,
		Data:    nil,
	})
}

// ErrorWithMessage 错误响应（自定义消息，用于兼容旧代码）
func ErrorWithMessage(c *gin.Context, err error) {
	global.Log.Error("请求处理失败: ", err)
	if appErr, ok := err.(*AppError); ok {
		c.JSON(http.StatusOK, Response{
			Code:    appErr.Code,
			Message: appErr.Message,
			Data:    nil,
		})
		return
	}
	// 普通错误不暴露内部信息
	c.JSON(http.StatusOK, Response{
		Code:    ErrorCodeInternalServer,
		Message: ErrorMsgInternalServer,
		Data:    nil,
	})
}

// PageResult 分页结果结构体
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// PageSuccess 分页成功响应
func PageSuccess(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: SuccessMsg,
		Data: PageResult{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// Unauthorized 未授权响应
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    ErrorCodeUnauthorized,
		Message: message,
		Data:    nil,
	})
}

// Forbidden 禁止访问响应
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    ErrorCodeForbidden,
		Message: message,
		Data:    nil,
	})
}

// ValidationError 参数验证错误响应
// 将绑定/校验错误转换为友好中文提示，不透传 validator 内部细节
// （字段名经 core/validator.go 注册的 TagNameFunc 映射为 json 标签名）
func ValidationError(c *gin.Context, err error) {
	c.JSON(http.StatusOK, Response{
		Code:    ErrorCodeParamInvalid,
		Message: friendlyValidationMessage(err),
		Data:    nil,
	})
}

// friendlyValidationMessage 将参数校验错误转换为可读的中文提示
func friendlyValidationMessage(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) && len(ve) > 0 {
		msgs := make([]string, 0, len(ve))
		for _, fe := range ve {
			msgs = append(msgs, fmt.Sprintf("%s %s", fe.Field(), friendlyTagMessage(fe.Tag())))
		}
		return strings.Join(msgs, "；")
	}
	return "请求参数格式错误"
}

// friendlyTagMessage 校验规则的中文描述
func friendlyTagMessage(tag string) string {
	switch tag {
	case "required":
		return "为必填项"
	case "min":
		return "长度或数值过小"
	case "max":
		return "长度或数值过大"
	case "len":
		return "长度不正确"
	case "email":
		return "邮箱格式不正确"
	case "oneof":
		return "取值不在允许范围内"
	case "phone":
		return "手机号格式不正确"
	case "password":
		return "需8-18位且包含数字、字母、符号中的至少两种"
	case "status":
		return "状态取值不正确"
	default:
		return "校验失败(" + tag + ")"
	}
}
