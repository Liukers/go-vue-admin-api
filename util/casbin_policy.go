package util

import (
	"fmt"
	"go-vue-admin/global"
	"go-vue-admin/models"
	"go-vue-admin/models/constants"

	"github.com/casbin/casbin/v2"
)

// 所有非admin角色都拥有的基础接口权限（动态路由、个人中心）
var baseRolePolicies = [][2]string{
	{"/api/v1/system/routes", "GET"},
	{"/api/v1/system/users/info", "GET"},
	{"/api/v1/system/users/profile", "PUT"},
	{"/api/v1/system/users/password", "PUT"},
	// 只读支撑端点：用户表单的角色下拉、角色分配弹窗、菜单管理页
	// （keyMatch2 为锚定匹配，/roles 不覆盖这些子路径，需显式声明）
	{"/api/v1/system/roles/options", "GET"},
	{"/api/v1/system/roles/:id/menus", "GET"},
	{"/api/v1/system/menus/tree", "GET"},
}

// SyncRoleCasbinPolicies 按角色菜单重建该角色的 Casbin 策略（先清后建）
// 这是唯一的策略生成入口：admin 角色授予通配权限，
// 其余角色 = 基础权限 + 其菜单/按钮上声明的 api_path + method
func SyncRoleCasbinPolicies(e *casbin.SyncedEnforcer, roleID uint) error {
	if e == nil {
		return fmt.Errorf("casbin enforcer 未初始化")
	}
	roleKey := fmt.Sprintf("role_%d", roleID)

	if _, err := e.RemoveFilteredPolicy(0, roleKey); err != nil {
		return fmt.Errorf("清除角色 %s 旧策略失败: %v", roleKey, err)
	}

	var role models.SystemRole
	if err := global.DB.First(&role, roleID).Error; err != nil {
		return fmt.Errorf("查询角色 %d 失败: %v", roleID, err)
	}

	// 超级管理员授予全部接口权限
	// 注意：matcher 为 keyMatch2，"/*" 会被转换为 "/.*" 匹配所有路径
	if role.RoleCode == "admin" {
		if _, err := e.AddPolicy(roleKey, "/*", "*"); err != nil {
			return fmt.Errorf("添加管理员通配策略失败: %v", err)
		}
		return nil
	}

	// 基础权限
	for _, p := range baseRolePolicies {
		if _, err := e.AddPolicy(roleKey, p[0], p[1]); err != nil {
			return fmt.Errorf("添加基础策略失败: %v", err)
		}
	}

	// 角色菜单/按钮声明的接口权限（仅启用状态的菜单产生策略，与前端 perms 判定一致）
	var menus []models.SystemMenu
	if err := global.DB.Model(&models.SystemMenu{}).
		Joins("JOIN system_role_menu ON system_role_menu.menu_id = system_menu.id").
		Where("system_role_menu.role_id = ? AND system_menu.status = ? AND system_menu.api_path != '' AND system_menu.method != ''", roleID, constants.MenuStatusEnabled).
		Find(&menus).Error; err != nil {
		return fmt.Errorf("查询角色 %d 菜单失败: %v", roleID, err)
	}
	for _, menu := range menus {
		if _, err := e.AddPolicy(roleKey, menu.ApiPath, menu.Method); err != nil {
			global.Log.Errorf("添加策略失败 [%s %s %s]: %v", roleKey, menu.ApiPath, menu.Method, err)
		}
	}
	return nil
}

// SyncAllRolesCasbinPolicies 重建所有角色的 Casbin 策略并保存
func SyncAllRolesCasbinPolicies(e *casbin.SyncedEnforcer) error {
	if e == nil {
		return fmt.Errorf("casbin enforcer 未初始化")
	}

	var roles []models.SystemRole
	if err := global.DB.Find(&roles).Error; err != nil {
		return fmt.Errorf("查询角色列表失败: %v", err)
	}
	for _, role := range roles {
		if err := SyncRoleCasbinPolicies(e, role.ID); err != nil {
			global.Log.Errorf("同步角色 %s(role_%d) 策略失败: %v", role.RoleName, role.ID, err)
		}
	}

	if err := e.SavePolicy(); err != nil {
		return fmt.Errorf("保存策略失败: %v", err)
	}
	global.Log.Infof("已重建 %d 个角色的Casbin策略", len(roles))
	return nil
}
