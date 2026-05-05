package v1

import (
	v1 "go-vue-admin/api/v1"
	"go-vue-admin/middleware"

	"github.com/gin-gonic/gin"
)

var systemApi = v1.ApiGroupApp.SystemUserApi
var systemRoleApi = v1.ApiGroupApp.SystemRoleApi
var systemLogApi = v1.ApiGroupApp.SystemLogApi
var systemSettingApi = v1.ApiGroupApp.SystemSettingApi
var systemCaptchaApi = v1.ApiGroupApp.SystemCaptchaApi

// InitSystemRouter 初始化系统管理路由
func InitSystemRouter(rg *gin.RouterGroup) {
	router := rg.Group("/system")
	{
		// 公开路由（不需要认证）
		// 登录接口添加限流保护，防止暴力破解
		router.GET("/captcha", systemCaptchaApi.GetCaptcha)
		router.POST("/login", middleware.LoginRateLimit(), systemApi.Login)
		router.POST("/logout", systemApi.Logout)
		router.POST("/refresh-token", middleware.LoginRateLimit(), systemApi.RefreshToken)

		// 需要认证的路由
		authRouter := router.Use(
			middleware.JWTAuth(),
			middleware.TokenBlacklistMiddleware(),
			middleware.CasbinAuth(),
			middleware.OperationLog(),
			middleware.APIRateLimit(),
		)
		{
			// 动态路由菜单
			authRouter.GET("/routes", systemApi.GetAsyncRoutes)

			// 用户管理
			authRouter.GET("/users", systemApi.GetUserList)
			authRouter.GET("/users/info", systemApi.GetUserInfo)
			authRouter.POST("/users", systemApi.CreateUser)
			authRouter.PUT("/users/:id", systemApi.UpdateUser)
			authRouter.DELETE("/users/:id", systemApi.DeleteUser)
			// 当前用户相关（个人中心）
			authRouter.PUT("/users/profile", systemApi.UpdateCurrentUser)
			authRouter.PUT("/users/password", systemApi.UpdateCurrentUserPassword)

			// 角色管理
			authRouter.GET("/roles", systemRoleApi.GetRoleList)
			authRouter.GET("/roles/options", systemRoleApi.GetRoleOptions)
			authRouter.GET("/roles/:id", systemRoleApi.GetRoleDetail)
			authRouter.POST("/roles", systemRoleApi.CreateRole)
			authRouter.PUT("/roles/:id", systemRoleApi.UpdateRole)
			authRouter.DELETE("/roles/:id", systemRoleApi.DeleteRole)
			authRouter.GET("/roles/:id/menus", systemRoleApi.GetRoleMenus)
			authRouter.PUT("/roles/:id/menus", systemRoleApi.SetRoleMenus)

			// 菜单管理
			authRouter.GET("/menus", systemRoleApi.GetMenuList)
			authRouter.GET("/menus/tree", systemRoleApi.GetMenuTree)
			authRouter.POST("/menus", systemRoleApi.CreateMenu)
			authRouter.PUT("/menus/:id", systemRoleApi.UpdateMenu)
			authRouter.DELETE("/menus/:id", systemRoleApi.DeleteMenu)

			// 系统设置
			authRouter.GET("/settings", systemSettingApi.GetSystemSetting)
			authRouter.PUT("/settings", systemSettingApi.UpdateSystemSetting)

			// 日志管理
			// 操作日志
			authRouter.GET("/operation-logs", systemLogApi.GetOperationLogList)
			authRouter.DELETE("/operation-logs/:id", systemLogApi.DeleteOperationLog)
			authRouter.DELETE("/operation-logs", systemLogApi.ClearOperationLog)
			// 登录日志
			authRouter.GET("/login-logs", systemLogApi.GetLoginLogList)
			authRouter.DELETE("/login-logs/:id", systemLogApi.DeleteLoginLog)
			authRouter.DELETE("/login-logs", systemLogApi.ClearLoginLog)
		}
	}
}
