package models

// SystemUser 系统用户表
type SystemUser struct {
	ID             uint       `gorm:"column:id;primarykey;comment:主键ID" json:"id"`
	CreatedAt      LocalTime  `gorm:"column:created_at;type:datetime;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt      LocalTime  `gorm:"column:updated_at;type:datetime;not null;comment:更新时间" json:"updatedAt"`
	Username       string     `gorm:"column:username;type:varchar(64);not null;uniqueIndex;comment:用户名" json:"username"`
	Password       string     `gorm:"column:password;type:varchar(128);not null;comment:密码" json:"-"`
	Nickname       string     `gorm:"column:nickname;type:varchar(64);comment:昵称" json:"nickname"`
	Avatar         string     `gorm:"column:avatar;type:varchar(255);comment:头像URL" json:"avatar"`
	Email          string     `gorm:"column:email;type:varchar(128);comment:邮箱" json:"email"`
	Phone          string     `gorm:"column:phone;type:varchar(20);comment:手机号" json:"phone"`
	Status         int        `gorm:"column:status;type:tinyint;default:1;comment:状态 1启用 2禁用" json:"status"`
	RoleID         uint       `gorm:"column:role_id;index;comment:角色ID" json:"roleId"`
	Role           SystemRole `gorm:"foreignKey:RoleID" json:"role"`
	LastLoginIP    string     `gorm:"column:last_login_ip;type:varchar(128);default:null;comment:最后登录IP" json:"lastLoginIp"`
	LastLoginAt    *string    `gorm:"column:last_login_at;type:datetime;default:null;comment:最后登录时间" json:"lastLoginAt"`
	LoginFailCount int        `gorm:"column:login_fail_count;type:int;default:0;comment:连续登录失败次数" json:"-"`
	LockedUntil    *LocalTime `gorm:"column:locked_until;type:datetime;default:null;comment:账户锁定截止时间" json:"-"`
	PasswordVersion int       `gorm:"column:password_version;type:int;default:0;comment:密码版本号，修改密码后递增" json:"-"`
	Roles          []string   `gorm:"-" json:"roles"` // 前端需要的角色数组格式
	Perms          []string   `gorm:"-" json:"perms"` // 前端需要的按钮/接口权限标识数组
}

func (SystemUser) TableName() string {
	return "system_user"
}

// SystemUserReq 系统用户请求参数（创建时使用）
type SystemUserReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,password"`
	Nickname string `json:"nickname"`
	Email    string `json:"email" binding:"omitempty,email"`
	Phone    string `json:"phone" binding:"omitempty,phone"`
	RoleID   uint   `json:"roleId"`
	Status   int    `json:"status" binding:"omitempty,status"`
}

// SystemUserUpdateReq 系统用户更新请求参数（更新时使用，字段都是可选的）
type SystemUserUpdateReq struct {
	ID       uint   `json:"id" binding:"required"`
	Username string `json:"username"`
	Password string `json:"password" binding:"omitempty,password"`
	Nickname string `json:"nickname"`
	Email    string `json:"email" binding:"omitempty,email"`
	Phone    string `json:"phone" binding:"omitempty,phone"`
	RoleID   uint   `json:"roleId"`
	Status   int    `json:"status" binding:"omitempty,status"`
}

// SystemUserLoginReq 登录请求参数
type SystemUserLoginReq struct {
	Username     string `json:"username" binding:"required"`
	Password     string `json:"password" binding:"required"`
	CaptchaId    string `json:"captchaId" binding:"required"`
	CaptchaCode  string `json:"captchaCode" binding:"required"`
}

// SystemUserLoginRes 登录响应参数
type SystemUserLoginRes struct {
	Token     string      `json:"token"`
	ExpiresAt int64       `json:"expiresAt"`
	UserInfo  SystemUser  `json:"userInfo"`
}

// SystemUserProfileReq 当前用户更新个人信息请求参数
type SystemUserProfileReq struct {
	Nickname *string `json:"nickname"`
	Avatar   *string `json:"avatar"`
	Email    *string `json:"email" binding:"omitempty,email"`
	Phone    *string `json:"phone" binding:"omitempty,phone"`
}

// SystemUserPasswordReq 当前用户修改密码请求参数
type SystemUserPasswordReq struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,password"`
}
