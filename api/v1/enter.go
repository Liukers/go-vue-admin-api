package v1

import (
	"go-vue-admin/services/v1"
)

type ApiGroup struct {
	SystemUserApi
	SystemRoleApi
	SystemLogApi
	SystemSettingApi
	SystemCaptchaApi
}

var (
	systemUserService    = v1.ServiceGroupApp.SystemUserService
	systemRoleService    = v1.ServiceGroupApp.SystemRoleService
	systemLogService     = v1.ServiceGroupApp.SystemLogService
	systemSettingService = v1.ServiceGroupApp.SystemSettingService
)

var ApiGroupApp = new(ApiGroup)
