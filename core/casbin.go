package core

import (
	"go-vue-admin/global"
	"go-vue-admin/util"
	"os"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
)

// InitCasbin 初始化Casbin权限管理
func InitCasbin() *casbin.SyncedEnforcer {
	adapter, err := gormadapter.NewAdapterByDB(global.DB)
	if err != nil {
		global.Log.Errorf("创建Casbin适配器失败: %v", err)
		return nil
	}

	var m model.Model
	modelPath := global.Config.Casbin.ModelPath
	if modelPath != "" {
		if _, err := os.Stat(modelPath); err == nil {
			m, err = model.NewModelFromFile(modelPath)
			if err != nil {
				global.Log.Errorf("从文件加载Casbin模型失败: %v", err)
				return nil
			}
		} else {
			m, err = model.NewModelFromString(global.Config.System.CasbinConfig)
			if err != nil {
				global.Log.Errorf("加载Casbin模型失败: %v", err)
				return nil
			}
		}
	} else {
		m, err = model.NewModelFromString(global.Config.System.CasbinConfig)
		if err != nil {
			global.Log.Errorf("加载Casbin模型失败: %v", err)
			return nil
		}
	}

	// 创建Enforcer（SyncedEnforcer 线程安全，允许与请求路径上的策略变更并发）
	e, err := casbin.NewSyncedEnforcer(m, adapter)
	if err != nil {
		global.Log.Errorf("创建Casbin Enforcer失败: %v", err)
		return nil
	}

	// 从数据库加载策略
	if err := e.LoadPolicy(); err != nil {
		global.Log.Errorf("加载Casbin策略失败: %v", err)
		return nil
	}

	// 确保默认菜单/按钮数据完整（幂等，兼容老库升级时补齐按钮和新字段）
	if created, err := util.EnsureDefaultMenus(global.DB); err != nil {
		global.Log.Errorf("初始化默认菜单数据失败: %v", err)
	} else if created > 0 {
		global.Log.Infof("已补齐 %d 个默认菜单/按钮", created)
	}

	// 按角色菜单全量重建策略（以 system_role_menu + system_menu 为唯一数据源，
	// 同时完成老策略数据（如 admin 的 "*" 通配）到 keyMatch2 写法的迁移）
	if err := util.SyncAllRolesCasbinPolicies(e); err != nil {
		global.Log.Errorf("同步Casbin策略失败: %v", err)
	}

	global.Log.Info("Casbin权限管理初始化成功")
	return e
}
