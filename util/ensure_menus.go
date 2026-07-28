package util

import (
	"fmt"
	"go-vue-admin/global"
	"go-vue-admin/models"

	"gorm.io/gorm"
)

// EnsureDefaultMenus 确保默认菜单数据完整（幂等）
// - 新装数据库：全量创建目录/菜单/按钮
// - 老库升级：按 Key（目录/菜单用 path、按钮用 perm）补齐缺失项，
//   并仅在 api_path 为空时回填菜单的 api_path/method（不覆盖用户自定义）
// - 保证 admin 角色拥有全部默认菜单（只插缺失的关联行）
// 返回新创建的菜单数量
func EnsureDefaultMenus(db *gorm.DB) (int, error) {
	idByKey := make(map[string]uint)
	var existing []models.SystemMenu
	if err := db.Find(&existing).Error; err != nil {
		return 0, fmt.Errorf("查询菜单失败: %v", err)
	}
	for _, m := range existing {
		if m.MenuType == 3 {
			if m.Perm != "" {
				idByKey[m.Perm] = m.ID
			}
		} else if m.Path != "" {
			idByKey[m.Path] = m.ID
		}
	}

	created := 0
	defaultIDs := make([]uint, 0, len(models.DefaultMenuSeeds))
	for _, seed := range models.DefaultMenuSeeds {
		if id, exists := idByKey[seed.Key]; exists {
			defaultIDs = append(defaultIDs, id)
			// 老数据回填 api_path/method（仅当为空，不覆盖用户自定义）
			if seed.Menu.ApiPath != "" {
				db.Model(&models.SystemMenu{}).
					Where("id = ? AND (api_path = '' OR api_path IS NULL)", id).
					Updates(map[string]interface{}{
						"api_path": seed.Menu.ApiPath,
						"method":   seed.Menu.Method,
					})
			}
			continue
		}

		menu := seed.Menu
		menu.ID = 0
		if seed.ParentKey != "" {
			parentID, ok := idByKey[seed.ParentKey]
			if !ok {
				return created, fmt.Errorf("菜单[%s]的父级[%s]不存在", seed.Key, seed.ParentKey)
			}
			menu.ParentID = parentID
		}
		if err := db.Create(&menu).Error; err != nil {
			return created, fmt.Errorf("创建菜单[%s]失败: %v", seed.Key, err)
		}
		idByKey[seed.Key] = menu.ID
		defaultIDs = append(defaultIDs, menu.ID)
		created++
	}

	// 给 admin 角色分配全部默认菜单（幂等，只插缺失的关联行）
	var adminRole models.SystemRole
	if err := db.Where("role_code = ?", "admin").First(&adminRole).Error; err == nil {
		var assignedIDs []uint
		if err := db.Model(&models.SystemRoleMenu{}).
			Where("role_id = ?", adminRole.ID).
			Pluck("menu_id", &assignedIDs).Error; err != nil {
			return created, fmt.Errorf("查询admin角色菜单失败: %v", err)
		}
		assignedSet := make(map[uint]bool, len(assignedIDs))
		for _, id := range assignedIDs {
			assignedSet[id] = true
		}
		var newRows []models.SystemRoleMenu
		for _, id := range defaultIDs {
			if !assignedSet[id] {
				newRows = append(newRows, models.SystemRoleMenu{RoleID: adminRole.ID, MenuID: id})
			}
		}
		if len(newRows) > 0 {
			if err := db.Create(&newRows).Error; err != nil {
				return created, fmt.Errorf("给admin角色分配菜单失败: %v", err)
			}
		}
	}

	// 修正历史数据：角色拥有按钮/子菜单但缺少其父菜单时，补挂父菜单。
	// （前端角色弹窗只保存叶子节点，导致菜单本体缺失、角色"能看到菜单却没有查看权限"）
	if err := backfillParentMenus(db); err != nil {
		return created, fmt.Errorf("补挂父菜单失败: %v", err)
	}

	return created, nil
}

// backfillParentMenus 为缺少父菜单的角色-菜单关联补挂父菜单（幂等，每次启动执行）
func backfillParentMenus(db *gorm.DB) error {
	var roleMenus []models.SystemRoleMenu
	if err := db.Find(&roleMenus).Error; err != nil {
		return err
	}
	if len(roleMenus) == 0 {
		return nil
	}

	var allMenus []models.SystemMenu
	if err := db.Select("id", "parent_id").Find(&allMenus).Error; err != nil {
		return err
	}
	parentOf := make(map[uint]uint, len(allMenus))
	for _, m := range allMenus {
		parentOf[m.ID] = m.ParentID
	}

	type rmKey struct{ roleID, menuID uint }
	have := make(map[rmKey]bool, len(roleMenus))
	for _, rm := range roleMenus {
		have[rmKey{rm.RoleID, rm.MenuID}] = true
	}

	var backfill []models.SystemRoleMenu
	for _, rm := range roleMenus {
		pid := parentOf[rm.MenuID]
		if pid != 0 && !have[rmKey{rm.RoleID, pid}] {
			backfill = append(backfill, models.SystemRoleMenu{RoleID: rm.RoleID, MenuID: pid})
			have[rmKey{rm.RoleID, pid}] = true // 同一角色同一父菜单只补一次
		}
	}
	if len(backfill) > 0 {
		if err := db.Create(&backfill).Error; err != nil {
			return err
		}
		global.Log.Infof("已为角色-菜单关联补挂 %d 条缺失的父菜单", len(backfill))
	}
	return nil
}
