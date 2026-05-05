package middleware

import (
	"fmt"
	"net"
	"strings"
	"go-vue-admin/global"
	"go-vue-admin/models/res"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 限流器
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
}

// visitor 访问者信息
type visitor struct {
	limit       int           // 限制次数
	remaining   int           // 剩余次数
	resetTime   time.Time     // 重置时间
	window      time.Duration // 时间窗口
	lastSeen    time.Time     // 最后访问时间（用于清理）
}

// NewRateLimiter 创建限流器
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
	}
	// 启动清理协程
	go rl.cleanup()
	return rl
}

// Allow 检查是否允许访问
func (rl *RateLimiter) Allow(key string, limit int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[key]

	if !exists || now.After(v.resetTime) {
		// 新访问者或已过期，创建新的限制
		rl.visitors[key] = &visitor{
			limit:     limit,
			remaining: limit - 1,
			resetTime: now.Add(window),
			window:    window,
			lastSeen:  now,
		}
		return true
	}

	// 更新最后访问时间
	v.lastSeen = now

	// 检查剩余次数
	if v.remaining > 0 {
		v.remaining--
		return true
	}

	return false
}

// GetRemaining 获取剩余次数
func (rl *RateLimiter) GetRemaining(key string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if v, exists := rl.visitors[key]; exists {
		return v.remaining
	}
	return 0
}

// cleanup 定期清理过期的访问者（防止内存泄漏）
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, v := range rl.visitors {
			// 清理已过期且超过10分钟未访问的条目
			if now.After(v.resetTime) && now.Sub(v.lastSeen) > 10*time.Minute {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

// 全局限流器实例
var (
	loginLimiter = NewRateLimiter()
	apiLimiter   = NewRateLimiter()
)

// getRealIP 获取真实IP（防止X-Forwarded-For伪造）
func getRealIP(c *gin.Context) string {
	// 默认使用直接连接的IP
	ip := c.Request.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	
	// 只有显式配置信任代理时才读取 X-Forwarded-For / X-Real-Ip
	// 生产环境使用反向代理时，应在 setting.yaml 中设置 system.trust-proxy: true
	if global.Config.System.TrustProxy {
		xff := c.Request.Header.Get("X-Forwarded-For")
		if xff != "" {
			// 取第一个IP（最原始的客户端IP）
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				trimmed := strings.TrimSpace(ips[0])
				if net.ParseIP(trimmed) != nil {
					return trimmed
				}
			}
		}
		xri := c.Request.Header.Get("X-Real-Ip")
		if xri != "" && net.ParseIP(xri) != nil {
			return xri
		}
	}
	
	return ip
}

// isPrivateIP 检查是否为内网IP
func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	// 检查私有IP段
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
	}
	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil && ipNet.Contains(parsedIP) {
			return true
		}
	}
	return false
}

// LoginRateLimit 登录限流中间件
// 限制每个IP每分钟最多5次登录尝试
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := getRealIP(c)
		key := fmt.Sprintf("login:%s", ip)

		if !loginLimiter.Allow(key, 5, time.Minute) {
			res.FailWithMessage(c, res.ErrorCodeTooManyRequests, "登录尝试次数过多，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

// APIRateLimit API限流中间件
// 限制每个用户每分钟最多100次请求
func APIRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, exists := c.Get("userId")
		if !exists {
			c.Next()
			return
		}
		
		// 安全的类型断言
		uid, ok := userId.(uint)
		if !ok {
			c.Next()
			return
		}

		key := fmt.Sprintf("api:%d", uid)

		if !apiLimiter.Allow(key, 100, time.Minute) {
			res.FailWithMessage(c, res.ErrorCodeTooManyRequests, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetLoginRemainingAttempts 获取登录剩余尝试次数
func GetLoginRemainingAttempts(ip string) int {
	key := fmt.Sprintf("login:%s", ip)
	return loginLimiter.GetRemaining(key)
}
