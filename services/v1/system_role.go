package v1

import (
	"errors"
	"fmt"
	"strings"

	"go-vue-admin/global"
	"go-vue-admin/models"
	"go-vue-admin/models/res"
	"go-vue-admin/util"
)

type SystemRoleService struct{}

// SetRoleMenusReq 设置角色菜单权限请求
type SetRoleMenusReq struct {
	RoleID  uint   `json:"roleId" binding:"required"`
	MenuIDs []uint `json:"menuIds" binding:"required"`
}

// ==================== 角色管理 ====================

// GetRoleByID 根据ID获取角色
func (s *SystemRoleService) GetRoleByID(id uint) (*models.SystemRole, error) {
	var role models.SystemRole
	err := global.DB.First(&role, id).Error
	return &role, err
}

// GetRoleList 获取角色列表
func (s *SystemRoleService) GetRoleList(page, pageSize int, keyword string) ([]models.SystemRole, int64, error) {
	var roles []models.SystemRole
	var total int64

	db := global.DB.Model(&models.SystemRole{})

	if keyword != "" {
		db = db.Where("role_name LIKE ? OR role_code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&roles).Error

	return roles, total, err
}

// GetRoleOptions 获取角色选项列表（排除超级管理员，用于用户新增/编辑时的下拉选择）
func (s *SystemRoleService) GetRoleOptions() ([]models.SystemRole, error) {
	var roles []models.SystemRole
	err := global.DB.Model(&models.SystemRole{}).
		Where("role_code != ?", "admin").
		Where("status = ?", 1).
		Order("sort asc").
		Find(&roles).Error
	return roles, err
}

// CheckRoleCodeExist 检查角色代码是否已存在
func (s *SystemRoleService) CheckRoleCodeExist(roleCode string) bool {
	var count int64
	if err := global.DB.Model(&models.SystemRole{}).Where("role_code = ?", roleCode).Count(&count).Error; err != nil {
		global.Log.Errorf("检查角色代码是否存在失败: %v", err)
		return false
	}
	return count > 0
}

// CreateRole 创建角色
func (s *SystemRoleService) CreateRole(role *models.SystemRole) (uint, error) {
	if err := global.DB.Create(role).Error; err != nil {
		return 0, err
	}
	return role.ID, nil
}

// UpdateRole 更新角色
func (s *SystemRoleService) UpdateRole(role *models.SystemRole) error {
	return global.DB.Model(&models.SystemRole{}).Where("id = ?", role.ID).Updates(map[string]interface{}{
		"role_name":   role.RoleName,
		"description": role.Description,
		"status":      role.Status,
		"sort":        role.Sort,
	}).Error
}

// DeleteRole 删除角色
func (s *SystemRoleService) DeleteRole(id uint) error {
	// 检查是否有用户关联此角色（在事务外查询）
	var count int64
	if err := global.DB.Model(&models.SystemUser{}).Where("role_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrRoleHasUsers
	}

	tx := global.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("role_id = ?", id).Delete(&models.SystemRoleMenu{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&models.SystemRole{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// 清理该角色的 Casbin 策略，避免残留孤儿策略
	if global.Casbin != nil {
		if _, err := global.Casbin.RemoveFilteredPolicy(0, fmt.Sprintf("role_%d", id)); err != nil {
			global.Log.Errorf("清理角色[%d]的Casbin策略失败: %v", id, err)
		}
	}

	return nil
}

// GetRoleMenus 获取角色的菜单权限
func (s *SystemRoleService) GetRoleMenus(roleID uint) ([]uint, error) {
	var menuIDs []uint
	err := global.DB.Model(&models.SystemRoleMenu{}).Where("role_id = ?", roleID).Pluck("menu_id", &menuIDs).Error
	return menuIDs, err
}

// SetRoleMenus 设置角色的菜单权限
func (s *SystemRoleService) SetRoleMenus(req *SetRoleMenusReq) error {
	// 禁止操作系统保留角色的权限：
	// admin 的接口权限由通配策略保证，其侧边栏菜单由关联表驱动，改写会导致菜单丢失
	var role models.SystemRole
	if err := global.DB.First(&role, req.RoleID).Error; err != nil {
		return res.NewAppErrorByCode(res.ErrorCodeNotFound)
	}
	if role.RoleCode == "admin" {
		return res.NewAppError(res.ErrorCodeBusinessError, "系统保留角色不允许修改权限")
	}

	//有操作按钮必有查看按钮，无查看按钮必无操作按钮
	req.MenuIDs = normalizeRoleMenuIDs(req.MenuIDs)

	tx := global.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("role_id = ?", req.RoleID).Delete(&models.SystemRoleMenu{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(req.MenuIDs) > 0 {
		roleMenus := make([]models.SystemRoleMenu, 0, len(req.MenuIDs))
		for _, menuID := range req.MenuIDs {
			roleMenus = append(roleMenus, models.SystemRoleMenu{
				RoleID: req.RoleID,
				MenuID: menuID,
			})
		}

		if err := tx.CreateInBatches(roleMenus, 100).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// 同步更新 Casbin 策略（按角色菜单全量重建，唯一的策略生成入口）
	if err := util.SyncRoleCasbinPolicies(global.Casbin, req.RoleID); err != nil {
		global.Log.Errorf("同步角色[%d]的Casbin策略失败: %v", req.RoleID, err)
	}

	return nil
}

// normalizeRoleMenuIDs 兜底归一化角色菜单权限：
// 同一菜单下勾选任意操作按钮时，必须同时勾选“查看”按钮；
// 未勾选“查看”按钮时，不得保留该菜单下的操作按钮。
func normalizeRoleMenuIDs(menuIDs []uint) []uint {
	if len(menuIDs) == 0 {
		return menuIDs
	}

	var allMenus []models.SystemMenu
	if err := global.DB.Find(&allMenus).Error; err != nil {
		global.Log.Warnf("归一化角色菜单时查询所有菜单失败: %v", err)
		return menuIDs
	}

	menuMap := make(map[uint]models.SystemMenu, len(allMenus))
	childrenByParent := make(map[uint][]models.SystemMenu)
	for _, m := range allMenus {
		menuMap[m.ID] = m
		if m.ParentID > 0 {
			childrenByParent[m.ParentID] = append(childrenByParent[m.ParentID], m)
		}
	}

	idSet := make(map[uint]struct{}, len(menuIDs))
	for _, id := range menuIDs {
		idSet[id] = struct{}{}
	}

	// 第一轮：勾选操作按钮时自动补 view
	for _, id := range menuIDs {
		menu, ok := menuMap[id]
		if !ok || menu.MenuType != 3 || strings.HasSuffix(menu.Perm, ":view") {
			continue
		}
		for _, sib := range childrenByParent[menu.ParentID] {
			if sib.MenuType == 3 && strings.HasSuffix(sib.Perm, ":view") {
				idSet[sib.ID] = struct{}{}
				break
			}
		}
	}

	// 第二轮：若某菜单下没有 view，则清除该菜单下所有操作按钮
	for _, children := range childrenByParent {
		hasView := false
		for _, child := range children {
			if child.MenuType == 3 && strings.HasSuffix(child.Perm, ":view") {
				if _, ok := idSet[child.ID]; ok {
					hasView = true
					break
				}
			}
		}
		if !hasView {
			for _, child := range children {
				if child.MenuType == 3 {
					delete(idSet, child.ID)
				}
			}
		}
	}

	result := make([]uint, 0, len(idSet))
	for id := range idSet {
		result = append(result, id)
	}
	return result
}

// ==================== 菜单管理 ====================

// GetMenuByID 根据ID获取菜单
func (s *SystemRoleService) GetMenuByID(id uint) (*models.SystemMenu, error) {
	var menu models.SystemMenu
	err := global.DB.First(&menu, id).Error
	return &menu, err
}

// GetMenuList 获取菜单列表
func (s *SystemRoleService) GetMenuList() ([]models.SystemMenu, error) {
	var menus []models.SystemMenu
	err := global.DB.Order("sort asc").Find(&menus).Error
	return menus, err
}

// GetMenuTree 获取菜单树
func (s *SystemRoleService) GetMenuTree() ([]map[string]interface{}, error) {
	menus, err := s.GetMenuList()
	if err != nil {
		return nil, err
	}
	return s.buildMenuTree(menus, 0), nil
}

// buildMenuTree 构建菜单树
func (s *SystemRoleService) buildMenuTree(menus []models.SystemMenu, parentId uint) []map[string]interface{} {
	var tree []map[string]interface{}
	for _, menu := range menus {
		if menu.ParentID == parentId {
			// 手动构建 map，确保所有字段都正确（包括嵌套 Model 中的 ID）
			item := map[string]interface{}{
				"id":        menu.ID,
				"parentId":  menu.ParentID,
				"menuName":  menu.MenuName,
				"menuType":  menu.MenuType,
				"icon":      menu.Icon,
				"path":      menu.Path,
				"component": menu.Component,
				"perm":      menu.Perm,
				"apiPath":   menu.ApiPath,
				"method":    menu.Method,
				"sort":      menu.Sort,
				"status":    menu.Status,
				"visible":   menu.Visible,
				"createdAt": menu.CreatedAt,
				"updatedAt": menu.UpdatedAt,
			}
			children := s.buildMenuTree(menus, menu.ID)
			if len(children) > 0 {
				item["children"] = children
			}
			tree = append(tree, item)
		}
	}
	return tree
}

// CreateMenu 创建菜单
func (s *SystemRoleService) CreateMenu(menu *models.SystemMenu) (uint, error) {
	tx := global.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(menu).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	// 自动给超级管理员角色分配新菜单权限
	var adminRole models.SystemRole
	if err := tx.Where("role_code = ?", "admin").First(&adminRole).Error; err == nil {
		roleMenu := models.SystemRoleMenu{
			RoleID: adminRole.ID,
			MenuID: menu.ID,
		}
		if err := tx.Create(&roleMenu).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return menu.ID, nil
}

// UpdateMenu 更新菜单
func (s *SystemRoleService) UpdateMenu(menu *models.SystemMenu) error {
	if err := global.DB.Model(&models.SystemMenu{}).Where("id = ?", menu.ID).Updates(map[string]interface{}{
		"parent_id": menu.ParentID,
		"menu_name": menu.MenuName,
		"menu_type": menu.MenuType,
		"icon":      menu.Icon,
		"path":      menu.Path,
		"component": menu.Component,
		"perm":      menu.Perm,
		"api_path":  menu.ApiPath,
		"method":    menu.Method,
		"sort":      menu.Sort,
		"status":    menu.Status,
		"visible":   menu.Visible,
	}).Error; err != nil {
		return err
	}

	// 菜单的 api_path/method/status 变更会影响权限策略，同步重建关联角色的策略
	var roleIDs []uint
	if err := global.DB.Model(&models.SystemRoleMenu{}).Where("menu_id = ?", menu.ID).Distinct().Pluck("role_id", &roleIDs).Error; err != nil {
		global.Log.Errorf("查询菜单[%d]关联角色失败: %v", menu.ID, err)
		return nil
	}
	for _, roleID := range roleIDs {
		if err := util.SyncRoleCasbinPolicies(global.Casbin, roleID); err != nil {
			global.Log.Errorf("更新菜单后同步角色[%d]的Casbin策略失败: %v", roleID, err)
		}
	}
	return nil
}

// DeleteMenu 删除菜单
func (s *SystemRoleService) DeleteMenu(id uint) error {
	var count int64
	if err := global.DB.Model(&models.SystemMenu{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrMenuHasChildren
	}

	// 先记录受影响的角色，删除后需要重建它们的 Casbin 策略
	var roleIDs []uint
	if err := global.DB.Model(&models.SystemRoleMenu{}).Where("menu_id = ?", id).Distinct().Pluck("role_id", &roleIDs).Error; err != nil {
		return err
	}

	tx := global.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除角色-菜单关联，避免残留孤儿数据
	if err := tx.Where("menu_id = ?", id).Delete(&models.SystemRoleMenu{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&models.SystemMenu{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	for _, roleID := range roleIDs {
		if err := util.SyncRoleCasbinPolicies(global.Casbin, roleID); err != nil {
			global.Log.Errorf("删除菜单后同步角色[%d]的Casbin策略失败: %v", roleID, err)
		}
	}

	return nil
}

// ==================== 错误定义 ====================

var (
	ErrRoleHasUsers    = errors.New("该角色下存在用户，无法删除")
	ErrMenuHasChildren = errors.New("该菜单下存在子菜单，无法删除")
)
