package middleware

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"go-vue-admin/conf"
	"go-vue-admin/global"
	"go-vue-admin/models"
	"go-vue-admin/models/constants"
	"go-vue-admin/models/res"
	"go-vue-admin/util"

	"github.com/gin-gonic/gin"
)

var tokenRefreshLocks sync.Map

// tokenRefreshMu 使刷新锁的检查-设置原子化，
// 防止同一用户并发刷新各自签发新 token
var tokenRefreshMu sync.Mutex

// tryLockRefresh 尝试获取刷新锁，防止同一用户并发刷新产生多个新 token
func tryLockRefresh(userID uint) bool {
	key := fmt.Sprintf("refresh:%d", userID)
	now := time.Now()
	tokenRefreshMu.Lock()
	defer tokenRefreshMu.Unlock()
	if val, ok := tokenRefreshLocks.Load(key); ok {
		if lockUntil := val.(time.Time); now.Before(lockUntil) {
			return false
		}
	}
	tokenRefreshLocks.Store(key, now.Add(10*time.Second))
	return true
}

// sweepExpiredRefreshLocks 清理已过期的刷新锁条目（避免 map 只增不减）
func sweepExpiredRefreshLocks() {
	now := time.Now()
	tokenRefreshLocks.Range(func(key, val any) bool {
		if lockUntil, ok := val.(time.Time); ok && now.After(lockUntil) {
			tokenRefreshLocks.Delete(key)
		}
		return true
	})
}

// JWTAuth JWT认证中间件
// 支持Token自动刷新，刷新后的旧Token会被加入黑名单
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			res.Unauthorized(c, "请求未携带token，无法访问")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "token格式错误")
			c.Abort()
			return
		}

		tokenString := parts[1]
		j := util.NewJWT()
		claims, err := j.ParseToken(tokenString)
		
		if err != nil {
			if err == util.TokenExpired {
				res.FailWithMessage(c, res.ErrorCodeTokenExpired, "token已过期，请重新登录")
				c.Abort()
				return
			}
			res.FailWithMessage(c, res.ErrorCodeTokenInvalid, err.Error())
			c.Abort()
			return
		}

		// 校验用户状态（禁用后立即失效）和密码版本号（修改密码后旧 token 失效）
		// 同时取出最新角色ID：鉴权以数据库为准，角色调整对活跃会话即时生效
		var currentUser models.SystemUser
		if err := global.DB.Select("password_version", "status", "role_id").First(&currentUser, claims.UserID).Error; err != nil {
			res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "用户不存在")
			c.Abort()
			return
		}
		if currentUser.Status != constants.UserStatusEnabled {
			res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "用户已被禁用，请联系管理员")
			c.Abort()
			return
		}
		if currentUser.PasswordVersion != claims.PasswordVersion {
			res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "密码已修改，请重新登录")
			c.Abort()
			return
		}

		// 校验角色状态（角色禁用后，其所有用户的会话立即失效）
		var currentRole models.SystemRole
		if err := global.DB.Select("status").First(&currentRole, currentUser.RoleID).Error; err != nil {
			res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "角色不存在，请联系管理员")
			c.Abort()
			return
		}
		if currentRole.Status != constants.RoleStatusEnabled {
			res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "角色已被禁用，请联系管理员")
			c.Abort()
			return
		}

		// 检查是否需要刷新token（在配置的缓冲期内，默认过期前1小时）
		// 注意：旧token必须等 c.Next() 之后才能加入黑名单。
		// 本中间件之后还有 TokenBlacklistMiddleware，若在响应完成前就拉黑
		// 本次请求正在使用的token，黑名单中间件会立即命中它，
		// 导致临期token的每一个请求都被拒绝
		var oldToken string
		var oldTokenExpiresAt time.Time
		if claims.ExpiresAt != nil {
			bufferTime := time.Duration(conf.GetConfig().JWT.BufferTime) * time.Hour
			if time.Until(claims.ExpiresAt.Time) < bufferTime {
				// 使用锁防止并发刷新产生多个新 token
				if tryLockRefresh(claims.UserID) {
					newToken, err := j.RefreshToken(tokenString)
					if err == nil {
						c.Header("X-Refresh-Token", newToken)
						global.Log.Infof("用户[%s]的token已自动刷新", claims.Username)
						oldToken = tokenString
						oldTokenExpiresAt = claims.ExpiresAt.Time
					}
				}
			}
		}

		// roleId 以数据库最新值为准，而非 token 中可能过期的 claims
		c.Set("userId", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roleId", currentUser.RoleID)
		c.Set("claims", claims)

		global.Log.Debugf("用户[%s]访问: %s %s", claims.Username, c.Request.Method, c.Request.URL.Path)

		c.Next()

		// 响应完成后再将刷新前的旧token加入黑名单，防止重用攻击
		if oldToken != "" {
			tb := &TokenBlacklist{}
			if err := tb.AddToBlacklist(oldToken, oldTokenExpiresAt); err != nil {
				global.Log.Errorf("将旧token加入黑名单失败: %v", err)
			}
		}
	}
}
