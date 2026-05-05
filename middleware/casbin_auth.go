package middleware

import (
	"fmt"
	"go-vue-admin/global"
	"go-vue-admin/models/res"

	"github.com/gin-gonic/gin"
)

// CasbinAuth Casbin权限检查中间件
// 检查用户是否有权限访问指定资源
func CasbinAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户角色ID
		roleId, exists := c.Get("roleId")
		if !exists {
			res.Fail(c, res.ErrorCodeUnauthorized)
			c.Abort()
			return
		}

		// 获取请求路径和方法
		path := c.Request.URL.Path
		method := c.Request.Method

		// 安全的类型断言
		roleIdUint, ok := roleId.(uint)
		if !ok {
			global.Log.Errorf("roleId 类型断言失败: %T", roleId)
			res.Fail(c, res.ErrorCodeForbidden)
			c.Abort()
			return
		}

		// 构建角色key
		roleKey := fmt.Sprintf("role_%d", roleIdUint)

		// 使用Casbin检查权限
		success, err := global.Casbin.Enforce(roleKey, path, method)
		if err != nil {
			global.Log.Errorf("Casbin权限检查失败: %v", err)
			res.Fail(c, res.ErrorCodeInternalServer)
			c.Abort()
			return
		}

		if !success {
			global.Log.Warnf("用户无权限访问: role=%s, path=%s, method=%s", roleKey, path, method)
			res.Fail(c, res.ErrorCodeForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}

// CasbinAuthWithPerm 指定权限标识的Casbin权限检查
// 用于在控制器内部进行细粒度权限控制
func CheckPermission(c *gin.Context, perm string) bool {
	roleId, exists := c.Get("roleId")
	if !exists {
		return false
	}

	roleIdUint, ok := roleId.(uint)
	if !ok {
		global.Log.Errorf("roleId 类型断言失败: %T", roleId)
		return false
	}

	roleKey := fmt.Sprintf("role_%d", roleIdUint)
	success, err := global.Casbin.Enforce(roleKey, perm, "*")
	if err != nil {
		global.Log.Errorf("Casbin权限检查失败: %v", err)
		return false
	}

	return success
}
