package middleware

import (
	"net/http"
	"net/url"
	"go-vue-admin/conf"

	"github.com/gin-gonic/gin"
)

// Cors 跨域中间件（默认使用白名单模式，禁止反射任意Origin）
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")

		// 非浏览器请求（curl、后端直连、直接打开Swagger页面）不携带 Origin。
		// CORS 只约束浏览器行为，此类请求不应被跨域规则拦截
		if origin == "" {
			if method == "OPTIONS" {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.Next()
			return
		}

		cfg := conf.GetConfig().Cors
		var isAllowed bool

		if cfg.AllowAll {
			// 如果明确配置了允许所有，使用通配符（但不允许携带凭证）
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Credentials", "false")
			isAllowed = true
		} else if len(cfg.Whitelist) > 0 {
			for _, whitelist := range cfg.Whitelist {
				if isValidOrigin(origin, whitelist.AllowOrigin) != "" {
					c.Header("Access-Control-Allow-Origin", whitelist.AllowOrigin)
					if whitelist.AllowCredentials {
						c.Header("Access-Control-Allow-Credentials", "true")
					}
					isAllowed = true
					break
				}
			}
		}

		if !isAllowed && cfg.Mode == "strict-whitelist" && method != "OPTIONS" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// 未匹配白名单且非严格模式时放行，但不允许携带凭证
		if !isAllowed {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Credentials", "false")
		}

		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE, PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, X-Token")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type, Authorization, X-Refresh-Token")
		c.Header("Access-Control-Max-Age", "86400")

		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isValidOrigin 验证Origin是否匹配白名单
// 支持精确匹配和后缀匹配（如 https://*.example.com）
func isValidOrigin(origin, pattern string) string {
	if origin == "" {
		return ""
	}

	if origin == pattern {
		return origin
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return ""
	}

	if len(pattern) > 2 && pattern[0] == '*' && pattern[1] == '.' {
		suffix := pattern[1:] // .example.com
		if len(originURL.Host) >= len(suffix) &&
			originURL.Host[len(originURL.Host)-len(suffix):] == suffix {
			return origin
		}
	}

	return ""
}
