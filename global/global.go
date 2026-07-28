package global

import (
	"go-vue-admin/conf"

	"github.com/casbin/casbin/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	Config *conf.Server
	DB     *gorm.DB
	Log    *logrus.Logger
	// Casbin 使用线程安全的 SyncedEnforcer：
	// SetRoleMenus/DeleteMenu/DeleteRole 会在请求路径上修改策略，
	// 与普通 Enforcer 的并发 Enforce 会触发 concurrent map read/write fatal
	Casbin *casbin.SyncedEnforcer
)
