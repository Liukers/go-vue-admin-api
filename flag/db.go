package flag

import (
	"fmt"
	"go-vue-admin/global"
	"go-vue-admin/models"
	"go-vue-admin/util"
	"os"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
)

// ResetAdminPassword 重置管理员密码为 admin123
func ResetAdminPassword() {
	db := global.DB
	if db == nil {
		fmt.Println("数据库连接失败")
		return
	}

	// 查找管理员账号
	var user models.SystemUser
	if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
		fmt.Println("未找到管理员账号(admin)")
		return
	}

	// 生成新密码哈希
	newPassword := "admin123"
	hash := util.BcryptHash(newPassword)

	// 更新密码并递增密码版本号，使该账号已签发的 token 全部失效
	if err := db.Model(&user).Updates(map[string]interface{}{
		"password":         hash,
		"password_version": user.PasswordVersion + 1,
	}).Error; err != nil {
		fmt.Printf("密码重置失败: %v\n", err)
		return
	}

	fmt.Println("✅ 管理员密码已重置为: admin123")
}

// ResetDB 重置数据库（删除所有表并重新创建）
func ResetDB() {
	fmt.Println("⚠️  警告: 即将删除所有数据表并重新初始化！")
	fmt.Println("开始重置数据库...")

	db := global.DB
	if db == nil {
		fmt.Println("数据库连接失败")
		return
	}

	// 删除所有表（按依赖关系倒序删除）
	tables := []interface{}{
		&models.OperationLog{},
		&models.LoginLog{},
		&models.SystemRoleMenu{},
		&models.SystemMenu{},
		&models.SystemUser{},
		&models.SystemRole{},
	}

	for _, table := range tables {
		if err := db.Migrator().DropTable(table); err != nil {
			fmt.Printf("删除表失败: %v\n", err)
		}
	}

	// 删除 Casbin 规则表
	db.Migrator().DropTable("casbin_rule")

	fmt.Println("所有数据表已删除")

	// 重新初始化
	MigrateDB()
}

// MigrateDB 数据库迁移
func MigrateDB() {
	fmt.Println("开始初始化数据库...")

	db := global.DB
	if db == nil {
		fmt.Println("数据库连接失败")
		return
	}

	// 设置数据库字符集为 utf8mb4
	db.Exec("SET NAMES utf8mb4")
	db.Exec("SET CHARACTER SET utf8mb4")
	db.Exec("SET character_set_connection=utf8mb4")

	// 自动迁移表结构
	err := db.AutoMigrate(
		&models.SystemUser{},
		&models.SystemRole{},
		&models.SystemRoleMenu{},
		&models.SystemMenu{},
		&models.SystemSetting{},
		&models.OperationLog{},
		&models.LoginLog{},
	)
	if err != nil {
		fmt.Printf("数据库迁移失败: %v\n", err)
		return
	}

	// 修复表字符集
	fixCharset()

	fmt.Println("数据库迁移完成")

	// 初始化基础数据
	initBaseData()

	fmt.Println("数据库初始化完成")
}

// fixCharset 修复所有表的字符集为 utf8mb4
func fixCharset() {
	fmt.Println("开始修复表字符集...")
	db := global.DB
	tables := []string{
		"system_user",
		"system_role",
		"system_role_menu",
		"system_menu",
		"system_setting",
		"system_operation_log",
		"system_login_log",
	}
	
	for _, table := range tables {
		sql := fmt.Sprintf("ALTER TABLE %s CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", table)
		if err := db.Exec(sql).Error; err != nil {
			fmt.Printf("  修复表 %s 字符集失败: %v\n", table, err)
		} else {
			fmt.Printf("  表 %s 字符集已设置为 utf8mb4\n", table)
		}
	}
	
	// 修复长文本列字符集
	columns := []struct {
		table  string
		column string
		ctype  string
	}{
		{"system_operation_log", "response_data", "LONGTEXT"},
		{"system_operation_log", "request_data", "LONGTEXT"},
		{"system_operation_log", "error_message", "TEXT"},
		{"system_login_log", "message", "VARCHAR(255)"},
	}
	
	for _, col := range columns {
		sql := fmt.Sprintf("ALTER TABLE %s MODIFY %s %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", 
			col.table, col.column, col.ctype)
		if err := db.Exec(sql).Error; err != nil {
			fmt.Printf("  修复列 %s.%s 字符集失败: %v\n", col.table, col.column, err)
		} else {
			fmt.Printf("  列 %s.%s 字符集已设置为 utf8mb4\n", col.table, col.column)
		}
	}
	
	fmt.Println("表字符集修复完成")
}

// initBaseData 初始化基础数据
func initBaseData() {
	db := global.DB

	// 检查是否已存在角色数据
	var count int64
	db.Model(&models.SystemRole{}).Count(&count)
	if count > 0 {
		fmt.Println("基础数据已存在，跳过初始化")
		// 但菜单数据可能需要初始化（兼容旧数据）
		initMenuData()
		fmt.Println("正在同步所有角色的权限策略...")
		initCasbinPolicy()
		return
	}

	// 创建超级管理员角色
	adminRole := models.SystemRole{
		RoleName:    "超级管理员",
		RoleCode:    "admin",
		Description: "系统超级管理员，拥有所有权限",
		Status:      1,
		Sort:        0,
	}
	if err := db.Create(&adminRole).Error; err != nil {
		fmt.Printf("创建管理员角色失败: %v\n", err)
		return
	}

	// 创建普通用户角色
	userRole := models.SystemRole{
		RoleName:    "普通用户",
		RoleCode:    "user",
		Description: "普通用户角色",
		Status:      1,
		Sort:        1,
	}
	if err := db.Create(&userRole).Error; err != nil {
		fmt.Printf("创建用户角色失败: %v\n", err)
		return
	}

	// 创建默认管理员账号
	adminUser := models.SystemUser{
		Username:    "admin",
		Password:    util.BcryptHash("admin123"),
		Nickname:    "管理员",
		Email:       "admin@go-vue-admin.com",
		Phone:       "13800138000",
		Status:      1,
		RoleID:      adminRole.ID,
		LastLoginIP: "",
		LastLoginAt: nil,
	}
	if err := db.Create(&adminUser).Error; err != nil {
		fmt.Printf("创建管理员账号失败: %v\n", err)
		return
	}

	fmt.Printf("基础数据初始化完成\n")
	fmt.Printf("管理员账号: admin, 密码: admin123\n")

	// 初始化菜单数据及权限
	initMenuData()

	// 初始化系统设置（默认关闭日志）
	initSystemSetting()

	// 初始化Casbin权限
	initCasbinPolicy()
}

// initMenuData 初始化菜单数据（目录/菜单/按钮，幂等，兼容老库升级补齐）
func initMenuData() {
	created, err := util.EnsureDefaultMenus(global.DB)
	if err != nil {
		fmt.Printf("初始化菜单数据失败: %v\n", err)
		return
	}
	if created > 0 {
		fmt.Printf("已创建 %d 个默认菜单/按钮\n", created)
	} else {
		fmt.Println("菜单数据已存在，无需创建")
	}
	fmt.Println("菜单数据初始化完成")
}

// initSystemSetting 初始化系统设置
func initSystemSetting() {
	db := global.DB

	// 检查是否已存在设置
	var count int64
	db.Model(&models.SystemSetting{}).Count(&count)
	if count > 0 {
		fmt.Println("系统设置已存在，跳过初始化")
		return
	}

	setting := models.SystemSetting{
		EnableOperationLog: 2, // 默认关闭
		EnableLoginLog:     2, // 默认关闭
	}
	if err := db.Create(&setting).Error; err != nil {
		fmt.Printf("创建系统设置失败: %v\n", err)
		return
	}

	fmt.Println("系统设置初始化完成（默认关闭操作日志和登录日志）")
}

// initCasbinPolicy 初始化Casbin策略（按角色菜单为所有角色全量重建）
func initCasbinPolicy() {
	e, err := newCasbinEnforcer()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	if err := util.SyncAllRolesCasbinPolicies(e); err != nil {
		fmt.Printf("同步Casbin策略失败: %v\n", err)
		return
	}
	fmt.Println("Casbin权限初始化完成")
}

// newCasbinEnforcer 创建Casbin Enforcer（模型文件存在则从文件加载，否则使用内联配置）
func newCasbinEnforcer() (*casbin.SyncedEnforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(global.DB)
	if err != nil {
		return nil, fmt.Errorf("创建Casbin适配器失败: %v", err)
	}

	var m model.Model
	modelPath := global.Config.Casbin.ModelPath
	if modelPath != "" {
		if _, statErr := os.Stat(modelPath); statErr == nil {
			m, err = model.NewModelFromFile(modelPath)
		} else {
			m, err = model.NewModelFromString(global.Config.System.CasbinConfig)
		}
	} else {
		m, err = model.NewModelFromString(global.Config.System.CasbinConfig)
	}
	if err != nil {
		return nil, fmt.Errorf("加载Casbin模型失败: %v", err)
	}

	e, err := casbin.NewSyncedEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("创建Casbin Enforcer失败: %v", err)
	}
	return e, nil
}
