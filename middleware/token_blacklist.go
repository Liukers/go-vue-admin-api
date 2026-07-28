package middleware

import (
	"context"
	"errors"
	"go-vue-admin/global"
	"go-vue-admin/models/res"
	"go-vue-admin/util"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// blacklistTableReady 标记黑名单表是否已在启动时初始化成功
var blacklistTableReady atomic.Bool

// createBlacklistTableSQL 黑名单建表语句
const createBlacklistTableSQL = `
CREATE TABLE IF NOT EXISTS token_blacklist (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	token VARCHAR(512) NOT NULL UNIQUE COMMENT 'Token字符串',
	expires_at DATETIME NOT NULL COMMENT 'Token过期时间',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '加入黑名单时间',
	INDEX idx_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Token黑名单表';
`

// TokenBlacklist Token黑名单管理
type TokenBlacklist struct{}

// InitTokenBlacklistTable 启动时初始化token黑名单表
// 应在服务启动阶段调用且失败即退出（黑名单表不可用时继续运行会让黑名单静默失效）
func InitTokenBlacklistTable() error {
	if blacklistTableReady.Load() {
		return nil
	}
	if err := global.DB.Exec(createBlacklistTableSQL).Error; err != nil {
		return err
	}
	blacklistTableReady.Store(true)
	return nil
}

// StartTokenBlacklistCleanup 定期清理过期黑名单记录并清扫过期刷新锁（直到 ctx 取消）
func StartTokenBlacklistCleanup(ctx context.Context, interval time.Duration) {
	cleanup := func() {
		tb := &TokenBlacklist{}
		if err := tb.CleanupExpired(); err != nil {
			global.Log.Warnf("清理过期token黑名单失败: %v", err)
		}
		sweepExpiredRefreshLocks()
	}
	cleanup()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanup()
		}
	}
}

// AddToBlacklist 将token加入黑名单
func (tb *TokenBlacklist) AddToBlacklist(token string, expiresAt time.Time) error {
	if !blacklistTableReady.Load() {
		return errors.New("token黑名单表未初始化")
	}

	result := global.DB.Exec(
		"INSERT IGNORE INTO token_blacklist (token, expires_at) VALUES (?, ?)",
		token, expiresAt,
	)
	if result.Error != nil {
		global.Log.Errorf("添加token到黑名单失败: %v", result.Error)
		return result.Error
	}
	return nil
}

// IsBlacklisted 检查token是否在黑名单中
// 仅按token匹配（不带过期时间条件）：过期token在刷新宽限期内仍可用于
// refresh-token 换发，黑名单必须对其同样生效；宽限期外的旧记录由 CleanupExpired 清理
func (tb *TokenBlacklist) IsBlacklisted(token string) bool {
	if !blacklistTableReady.Load() {
		global.Log.Warn("token黑名单表未初始化，跳过黑名单检查")
		return false
	}
	
	var count int64
	result := global.DB.Raw(
		"SELECT COUNT(*) FROM token_blacklist WHERE token = ?",
		token,
	).Scan(&count)
	
	// 如果查询出错，允许访问
	if result.Error != nil {
		global.Log.Debugf("检查token黑名单时出错: %v", result.Error)
		return false
	}
	
	return count > 0
}

// CleanupExpired 清理过期时间已超过刷新宽限期的黑名单记录
// 建议定期调用（如每天一次）
func (tb *TokenBlacklist) CleanupExpired() error {
	cutoff := time.Now().Add(-util.RefreshGracePeriod)
	return global.DB.Exec("DELETE FROM token_blacklist WHERE expires_at <= ?", cutoff).Error
}

// TokenBlacklistMiddleware Token黑名单检查中间件
func TokenBlacklistMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]
		tb := &TokenBlacklist{}

		if tb.IsBlacklisted(token) {
			res.Fail(c, res.ErrorCodeTokenInvalid)
			c.Abort()
			return
		}

		c.Next()
	}
}

// LogoutHandler 登出处理
func LogoutHandler(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		res.Success(c, nil)
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		res.Success(c, nil)
		return
	}

	token := parts[1]

	// 忽略过期校验解析：过期token登出同样要进黑名单，
	// 否则其在刷新宽限期内仍可用于 refresh-token 换发新token
	j := util.NewJWT()
	claims, err := j.ParseTokenIgnoreExpiry(token)
	if err != nil {
		// Token无效，直接返回成功
		res.Success(c, nil)
		return
	}

	tb := &TokenBlacklist{}
	if claims.ExpiresAt != nil {
		if err := tb.AddToBlacklist(token, claims.ExpiresAt.Time); err != nil {
			global.Log.Errorf("添加token到黑名单失败: %v", err)
			// 继续返回成功，因为登出操作本身已完成
		}
	}

	res.Success(c, nil)
}
