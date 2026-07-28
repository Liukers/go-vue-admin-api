package models

// SystemRole 系统角色表
type SystemRole struct {
	ID          uint      `gorm:"column:id;primarykey;comment:主键ID" json:"id"`
	CreatedAt   LocalTime `gorm:"column:created_at;type:datetime;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt   LocalTime `gorm:"column:updated_at;type:datetime;not null;comment:更新时间" json:"updatedAt"`
	RoleName    string    `gorm:"column:role_name;type:varchar(128);not null;comment:角色名称" json:"roleName"`
	RoleCode    string    `gorm:"column:role_code;type:varchar(128);not null;uniqueIndex;comment:角色代码" json:"roleCode"`
	Description string    `gorm:"column:description;type:varchar(255);comment:角色描述" json:"description"`
	Status      int       `gorm:"column:status;type:tinyint;default:1;comment:状态 1启用 2禁用" json:"status"`
	Sort        int       `gorm:"column:sort;type:int;default:0;comment:排序" json:"sort"`
}

func (SystemRole) TableName() string {
	return "system_role"
}

// SystemRoleMenu 角色菜单关联表
type SystemRoleMenu struct {
	ID     uint `gorm:"column:id;primarykey" json:"id"`
	RoleID uint `gorm:"column:role_id;not null;index;comment:角色ID" json:"roleId"`
	MenuID uint `gorm:"column:menu_id;not null;index;comment:菜单ID" json:"menuId"`
}

func (SystemRoleMenu) TableName() string {
	return "system_role_menu"
}

// SystemMenu 系统菜单表
type SystemMenu struct {
	ID        uint      `gorm:"column:id;primarykey;comment:主键ID" json:"id"`
	CreatedAt LocalTime `gorm:"column:created_at;type:datetime;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt LocalTime `gorm:"column:updated_at;type:datetime;not null;comment:更新时间" json:"updatedAt"`
	ParentID  uint      `gorm:"column:parent_id;index;default:0;comment:父菜单ID" json:"parentId"`
	MenuName  string    `gorm:"column:menu_name;type:varchar(128);not null;comment:菜单名称" json:"menuName"`
	MenuType  int       `gorm:"column:menu_type;type:tinyint;default:1;comment:菜单类型 1目录 2菜单 3按钮" json:"menuType"`
	Icon      string    `gorm:"column:icon;type:varchar(128);comment:菜单图标" json:"icon"`
	Path      string    `gorm:"column:path;type:varchar(255);comment:路由路径" json:"path"`
	Component string    `gorm:"column:component;type:varchar(255);comment:组件路径" json:"component"`
	Perm      string    `gorm:"column:perm;type:varchar(255);comment:权限标识" json:"perm"`
	ApiPath   string    `gorm:"column:api_path;type:varchar(255);comment:关联API路径(菜单/按钮)" json:"apiPath"`
	Method    string    `gorm:"column:method;type:varchar(10);comment:关联API请求方法(菜单/按钮)" json:"method"`
	Sort      int       `gorm:"column:sort;type:int;default:0;comment:排序" json:"sort"`
	Status    int       `gorm:"column:status;type:tinyint;default:1;comment:状态 1启用 2禁用" json:"status"`
	Visible   int       `gorm:"column:visible;type:tinyint;default:1;comment:是否显示 1是 2否" json:"visible"`
}

func (SystemMenu) TableName() string {
	return "system_menu"
}

// CreateRoleReq 创建角色请求
 type CreateRoleReq struct {
	RoleName    string `json:"roleName" binding:"required"`
	RoleCode    string `json:"roleCode" binding:"required"`
	Description string `json:"description"`
	Status      int    `json:"status" binding:"omitempty,status"`
	Sort        int    `json:"sort"`
}

// UpdateRoleReq 更新角色请求
 type UpdateRoleReq struct {
	ID          uint   `json:"id" binding:"required"`
	RoleName    string `json:"roleName" binding:"required"`
	Description string `json:"description"`
	Status      int    `json:"status" binding:"omitempty,status"`
	Sort        int    `json:"sort"`
}

// CreateMenuReq 创建菜单请求
 type CreateMenuReq struct {
	ParentID  uint   `json:"parentId"`
	MenuName  string `json:"menuName" binding:"required"`
	MenuType  int    `json:"menuType" binding:"required,oneof=1 2 3"`
	Icon      string `json:"icon"`
	Path      string `json:"path"`
	Component string `json:"component"`
	Perm      string `json:"perm"`
	ApiPath   string `json:"apiPath"`
	Method    string `json:"method" binding:"omitempty,oneof=GET POST PUT DELETE"`
	Sort      int    `json:"sort"`
	Status    int    `json:"status" binding:"omitempty,status"`
	Visible   int    `json:"visible" binding:"omitempty,oneof=1 2"`
}

// UpdateMenuReq 更新菜单请求
 type UpdateMenuReq struct {
	ID        uint   `json:"id" binding:"required"`
	ParentID  uint   `json:"parentId"`
	MenuName  string `json:"menuName" binding:"required"`
	MenuType  int    `json:"menuType" binding:"required,oneof=1 2 3"`
	Icon      string `json:"icon"`
	Path      string `json:"path"`
	Component string `json:"component"`
	Perm      string `json:"perm"`
	ApiPath   string `json:"apiPath"`
	Method    string `json:"method" binding:"omitempty,oneof=GET POST PUT DELETE"`
	Sort      int    `json:"sort"`
	Status    int    `json:"status" binding:"omitempty,status"`
	Visible   int    `json:"visible" binding:"omitempty,oneof=1 2"`
}
