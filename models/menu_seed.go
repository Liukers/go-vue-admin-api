package models

// DefaultMenuSeed 默认菜单种子定义
// Key 为菜单唯一标识：目录/菜单使用前端路由 Path，按钮使用 Perm
// ParentKey 引用父级的 Key，插入时解析为实际的 ParentID
type DefaultMenuSeed struct {
	Key       string
	ParentKey string
	Menu      SystemMenu
}

// DefaultMenuSeeds 系统默认菜单（目录/菜单/按钮）
// 新装数据库全量创建；老库升级时按 Key 幂等补齐缺失项并回填 api_path/method
// 菜单/按钮的 ApiPath 支持 keyMatch2 路径参数写法（如 /api/v1/system/users/:id）
var DefaultMenuSeeds = []DefaultMenuSeed{
	// ==================== 目录 ====================
	{Key: "/system", Menu: SystemMenu{MenuName: "系统管理", MenuType: 1, Icon: "ri:settings-3-line", Path: "/system", Perm: "system:view", Sort: 1, Status: 1, Visible: 1}},

	// ==================== 菜单 ====================
	{Key: "/system/user", ParentKey: "/system", Menu: SystemMenu{MenuName: "用户管理", MenuType: 2, Icon: "ri:admin-line", Path: "/system/user", Component: "system/user/index", Perm: "system:user:view", ApiPath: "/api/v1/system/users", Method: "GET", Sort: 1, Status: 1, Visible: 1}},
	{Key: "/system/role", ParentKey: "/system", Menu: SystemMenu{MenuName: "角色管理", MenuType: 2, Icon: "ri:shield-keyhole-line", Path: "/system/role", Component: "system/role/index", Perm: "system:role:view", ApiPath: "/api/v1/system/roles", Method: "GET", Sort: 2, Status: 1, Visible: 1}},
	{Key: "/system/menu", ParentKey: "/system", Menu: SystemMenu{MenuName: "菜单管理", MenuType: 2, Icon: "ep:menu", Path: "/system/menu", Component: "system/menu/index", Perm: "system:menu:view", ApiPath: "/api/v1/system/menus", Method: "GET", Sort: 3, Status: 1, Visible: 1}},
	{Key: "/system/setting", ParentKey: "/system", Menu: SystemMenu{MenuName: "系统设置", MenuType: 2, Icon: "ri:settings-4-line", Path: "/system/setting", Component: "system/setting/index", Perm: "system:setting:view", ApiPath: "/api/v1/system/settings", Method: "GET", Sort: 4, Status: 1, Visible: 1}},
	{Key: "/system/log/operation", ParentKey: "/system", Menu: SystemMenu{MenuName: "操作日志", MenuType: 2, Icon: "ri:file-list-line", Path: "/system/log/operation", Component: "system/log/operation", Perm: "system:log:operation:view", ApiPath: "/api/v1/system/operation-logs", Method: "GET", Sort: 5, Status: 1, Visible: 1}},
	{Key: "/system/log/login", ParentKey: "/system", Menu: SystemMenu{MenuName: "登录日志", MenuType: 2, Icon: "ri:login-box-line", Path: "/system/log/login", Component: "system/log/login", Perm: "system:log:login:view", ApiPath: "/api/v1/system/login-logs", Method: "GET", Sort: 6, Status: 1, Visible: 1}},

	// ==================== 按钮：用户管理 ====================
	{Key: "system:user:view", ParentKey: "/system/user", Menu: SystemMenu{MenuName: "查看用户", MenuType: 3, Perm: "system:user:view", ApiPath: "/api/v1/system/users", Method: "GET", Sort: 0, Status: 1, Visible: 1}},
	{Key: "system:user:add", ParentKey: "/system/user", Menu: SystemMenu{MenuName: "新增用户", MenuType: 3, Perm: "system:user:add", ApiPath: "/api/v1/system/users", Method: "POST", Sort: 1, Status: 1, Visible: 1}},
	{Key: "system:user:edit", ParentKey: "/system/user", Menu: SystemMenu{MenuName: "编辑用户", MenuType: 3, Perm: "system:user:edit", ApiPath: "/api/v1/system/users/:id", Method: "PUT", Sort: 2, Status: 1, Visible: 1}},
	{Key: "system:user:delete", ParentKey: "/system/user", Menu: SystemMenu{MenuName: "删除用户", MenuType: 3, Perm: "system:user:delete", ApiPath: "/api/v1/system/users/:id", Method: "DELETE", Sort: 3, Status: 1, Visible: 1}},
	{Key: "system:user:resetPwd", ParentKey: "/system/user", Menu: SystemMenu{MenuName: "重置密码", MenuType: 3, Perm: "system:user:resetPwd", ApiPath: "/api/v1/system/users/:id", Method: "PUT", Sort: 4, Status: 1, Visible: 1}},

	// ==================== 按钮：角色管理 ====================
	{Key: "system:role:view", ParentKey: "/system/role", Menu: SystemMenu{MenuName: "查看角色", MenuType: 3, Perm: "system:role:view", ApiPath: "/api/v1/system/roles", Method: "GET", Sort: 0, Status: 1, Visible: 1}},
	{Key: "system:role:add", ParentKey: "/system/role", Menu: SystemMenu{MenuName: "新增角色", MenuType: 3, Perm: "system:role:add", ApiPath: "/api/v1/system/roles", Method: "POST", Sort: 1, Status: 1, Visible: 1}},
	{Key: "system:role:edit", ParentKey: "/system/role", Menu: SystemMenu{MenuName: "编辑角色", MenuType: 3, Perm: "system:role:edit", ApiPath: "/api/v1/system/roles/:id", Method: "PUT", Sort: 2, Status: 1, Visible: 1}},
	{Key: "system:role:delete", ParentKey: "/system/role", Menu: SystemMenu{MenuName: "删除角色", MenuType: 3, Perm: "system:role:delete", ApiPath: "/api/v1/system/roles/:id", Method: "DELETE", Sort: 3, Status: 1, Visible: 1}},
	{Key: "system:role:assign", ParentKey: "/system/role", Menu: SystemMenu{MenuName: "分配权限", MenuType: 3, Perm: "system:role:assign", ApiPath: "/api/v1/system/roles/:id/menus", Method: "PUT", Sort: 4, Status: 1, Visible: 1}},

	// ==================== 按钮：菜单管理 ====================
	{Key: "system:menu:view", ParentKey: "/system/menu", Menu: SystemMenu{MenuName: "查看菜单", MenuType: 3, Perm: "system:menu:view", ApiPath: "/api/v1/system/menus", Method: "GET", Sort: 0, Status: 1, Visible: 1}},
	{Key: "system:menu:add", ParentKey: "/system/menu", Menu: SystemMenu{MenuName: "新增菜单", MenuType: 3, Perm: "system:menu:add", ApiPath: "/api/v1/system/menus", Method: "POST", Sort: 1, Status: 1, Visible: 1}},
	{Key: "system:menu:edit", ParentKey: "/system/menu", Menu: SystemMenu{MenuName: "编辑菜单", MenuType: 3, Perm: "system:menu:edit", ApiPath: "/api/v1/system/menus/:id", Method: "PUT", Sort: 2, Status: 1, Visible: 1}},
	{Key: "system:menu:delete", ParentKey: "/system/menu", Menu: SystemMenu{MenuName: "删除菜单", MenuType: 3, Perm: "system:menu:delete", ApiPath: "/api/v1/system/menus/:id", Method: "DELETE", Sort: 3, Status: 1, Visible: 1}},

	// ==================== 按钮：系统设置 ====================
	{Key: "system:setting:view", ParentKey: "/system/setting", Menu: SystemMenu{MenuName: "查看设置", MenuType: 3, Perm: "system:setting:view", ApiPath: "/api/v1/system/settings", Method: "GET", Sort: 0, Status: 1, Visible: 1}},
	{Key: "system:setting:edit", ParentKey: "/system/setting", Menu: SystemMenu{MenuName: "保存设置", MenuType: 3, Perm: "system:setting:edit", ApiPath: "/api/v1/system/settings", Method: "PUT", Sort: 1, Status: 1, Visible: 1}},

	// ==================== 按钮：操作日志 ====================
	{Key: "system:log:operation:view", ParentKey: "/system/log/operation", Menu: SystemMenu{MenuName: "查看操作日志", MenuType: 3, Perm: "system:log:operation:view", ApiPath: "/api/v1/system/operation-logs", Method: "GET", Sort: 0, Status: 1, Visible: 1}},
	{Key: "system:log:operation:delete", ParentKey: "/system/log/operation", Menu: SystemMenu{MenuName: "删除日志", MenuType: 3, Perm: "system:log:operation:delete", ApiPath: "/api/v1/system/operation-logs/:id", Method: "DELETE", Sort: 1, Status: 1, Visible: 1}},
	{Key: "system:log:operation:clear", ParentKey: "/system/log/operation", Menu: SystemMenu{MenuName: "清空日志", MenuType: 3, Perm: "system:log:operation:clear", ApiPath: "/api/v1/system/operation-logs", Method: "DELETE", Sort: 2, Status: 1, Visible: 1}},

	// ==================== 按钮：登录日志 ====================
	{Key: "system:log:login:view", ParentKey: "/system/log/login", Menu: SystemMenu{MenuName: "查看登录日志", MenuType: 3, Perm: "system:log:login:view", ApiPath: "/api/v1/system/login-logs", Method: "GET", Sort: 0, Status: 1, Visible: 1}},
	{Key: "system:log:login:delete", ParentKey: "/system/log/login", Menu: SystemMenu{MenuName: "删除日志", MenuType: 3, Perm: "system:log:login:delete", ApiPath: "/api/v1/system/login-logs/:id", Method: "DELETE", Sort: 1, Status: 1, Visible: 1}},
	{Key: "system:log:login:clear", ParentKey: "/system/log/login", Menu: SystemMenu{MenuName: "清空日志", MenuType: 3, Perm: "system:log:login:clear", ApiPath: "/api/v1/system/login-logs", Method: "DELETE", Sort: 2, Status: 1, Visible: 1}},
}
