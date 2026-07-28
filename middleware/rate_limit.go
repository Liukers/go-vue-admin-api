package middleware

import (
	"fmt"
	"net"
	"strings"
	"go-vue-admin/conf"
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
		rl.visitors[key] = &visitor{
			limit:     limit,
			remaining: limit - 1,
			resetTime: now.Add(window),
			window:    window,
			lastSeen:  now,
		}
		return true
	}

	v.lastSeen = now

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

var (
	loginLimiter = NewRateLimiter()
	apiLimiter   = NewRateLimiter()
)

// getRealIP 获取真实IP（防止X-Forwarded-For伪造）
func getRealIP(c *gin.Context) string {
	ip := c.Request.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	
	// 只有显式配置信任代理时才读取 X-Forwarded-For / X-Real-Ip
	// 生产环境使用反向代理时，应在 setting.yaml 中设置 system.trust-proxy: true
	if conf.GetConfig().System.TrustProxy {
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

// CaptchaRateLimit 验证码限流中间件
// 限制每个IP每分钟最多20次，防止刷量挤占验证码存储并消耗图片生成资源
func CaptchaRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := getRealIP(c)
		key := fmt.Sprintf("captcha:%s", ip)

		if !loginLimiter.Allow(key, 20, time.Minute) {
			res.FailWithMessage(c, res.ErrorCodeTooManyRequests, "验证码获取过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
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
