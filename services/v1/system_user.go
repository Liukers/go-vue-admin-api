package v1

import (
	"errors"
	"go-vue-admin/global"
	"go-vue-admin/models"
	"go-vue-admin/models/res"
	"go-vue-admin/util"
	"time"

	"gorm.io/gorm"
)

type SystemUserService struct{}

// ==================== 基础 CRUD ====================

// GetUserByID 根据ID获取用户
func (s *SystemUserService) GetUserByID(id uint) (*models.SystemUser, error) {
	var user models.SystemUser
	err := global.DB.Preload("Role").First(&user, id).Error
	return &user, err
}

// GetUserByUsername 根据用户名获取用户
func (s *SystemUserService) GetUserByUsername(username string) (*models.SystemUser, error) {
	var user models.SystemUser
	err := global.DB.Preload("Role").Where("username = ?", username).First(&user).Error
	return &user, err
}

// CheckUserExist 检查用户名是否已存在
func (s *SystemUserService) CheckUserExist(username string) bool {
	var count int64
	if err := global.DB.Model(&models.SystemUser{}).Where("username = ?", username).Count(&count).Error; err != nil {
		global.Log.Errorf("检查用户名是否存在失败: %v", err)
		return false
	}
	return count > 0
}

// CheckUserExistExceptID 检查用户名是否已存在（排除指定ID）
func (s *SystemUserService) CheckUserExistExceptID(username string, excludeID uint) bool {
	var count int64
	if err := global.DB.Model(&models.SystemUser{}).Where("username = ? AND id != ?", username, excludeID).Count(&count).Error; err != nil {
		global.Log.Errorf("检查用户名是否存在失败: %v", err)
		return false
	}
	return count > 0
}

// CreateUser 创建用户
func (s *SystemUserService) CreateUser(req *models.SystemUserReq) (uint, error) {
	// 校验角色是否存在
	if req.RoleID > 0 {
		var role models.SystemRole
		if err := global.DB.First(&role, req.RoleID).Error; err != nil {
			return 0, res.NewAppErrorByCode(res.ErrorCodeNotFound)
		}
	}

	user := models.SystemUser{
		Username: req.Username,
		Password: util.BcryptHash(req.Password),
		Nickname: req.Nickname,
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   req.Status,
		RoleID:   req.RoleID,
	}
	if err := global.DB.Create(&user).Error; err != nil {
		return 0, err
	}
	return user.ID, nil
}

// UpdateUser 更新用户
func (s *SystemUserService) UpdateUser(req *models.SystemUserUpdateReq) error {
	var user models.SystemUser
	if err := global.DB.First(&user, req.ID).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{}

	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	// 使用指针类型或特殊值来判断是否更新Status
	if req.Status == 1 || req.Status == 2 {
		updates["status"] = req.Status
	}
	if req.RoleID != 0 {
		updates["role_id"] = req.RoleID
	}
	// 管理员更新密码 - 注意：这应该在单独的管理员重置密码接口中处理
	if req.Password != "" {
		updates["password"] = util.BcryptHash(req.Password)
	}

	if len(updates) == 0 {
		return nil
	}

	return global.DB.Model(&user).Updates(updates).Error
}

// DeleteUser 删除用户
func (s *SystemUserService) DeleteUser(id uint) error {
	return global.DB.Delete(&models.SystemUser{}, id).Error
}

// GetUserList 获取用户列表
func (s *SystemUserService) GetUserList(page, pageSize int, keyword, status string) ([]models.SystemUser, int64, error) {
	var users []models.SystemUser
	var total int64

	db := global.DB.Model(&models.SystemUser{}).Preload("Role")

	if keyword != "" {
		db = db.Where("username LIKE ? OR nickname LIKE ? OR phone LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status != "" {
		db = db.Where("status = ?", util.StringToInt(status))
	}

	// 检查Count错误
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error

	// 隐藏密码
	for i := range users {
		users[i].Password = ""
	}

	return users, total, err
}

// ==================== 业务方法 ====================

// Login 用户登录
func (s *SystemUserService) Login(req *models.SystemUserLoginReq, clientIP, userAgent string) (*models.SystemUserLoginRes, error) {
	// 1. 校验验证码
	if !util.VerifyCaptcha(req.CaptchaId, req.CaptchaCode) {
		s.recordLoginLog(req.Username, clientIP, userAgent, 2, "验证码错误")
		return nil, res.NewAppErrorByCode(res.ErrorCodeCaptchaError)
	}

	// 2. 查询用户（预加载角色信息）
	var user models.SystemUser
	err := global.DB.Preload("Role").Where("username = ?", req.Username).First(&user).Error

	// 时序攻击防护：无论用户是否存在，都执行一次固定耗时的密码比较
	dummyHash := "$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqhmM6JGKpS4G3R1G2JH8YpfB0Bqy"
	_ = util.BcryptCheck(req.Password, dummyHash)

	if err != nil {
		// 记录登录失败日志
		s.recordLoginLog(req.Username, clientIP, userAgent, 2, "用户名不存在")
		return nil, res.NewAppErrorByCode(res.ErrorCodeLoginFailed)
	}

	// 3. 检查账户是否被锁定
	if user.LockedUntil != nil && time.Time(*user.LockedUntil).After(time.Now()) {
		s.recordLoginLog(req.Username, clientIP, userAgent, 2, "账户已被锁定")
		return nil, res.NewAppErrorByCode(res.ErrorCodeAccountLocked)
	}

	// 4. 验证密码
	if !util.BcryptCheck(req.Password, user.Password) {
		// 记录登录失败日志
		s.recordLoginLog(req.Username, clientIP, userAgent, 2, "密码错误")
		// 增加失败计数并检查是否锁定（使用原子操作防止并发覆盖）
		s.handleLoginFailure(user.ID, user.Username)
		return nil, res.NewAppErrorByCode(res.ErrorCodePasswordError)
	}

	// 5. 检查用户状态
	if user.Status != 1 {
		// 记录登录失败日志
		s.recordLoginLog(req.Username, clientIP, userAgent, 2, "用户已被禁用")
		return nil, res.NewAppErrorByCode(res.ErrorCodeUserDisabled)
	}

	// 6. 登录成功：重置失败计数和锁定状态
	if user.LoginFailCount > 0 || user.LockedUntil != nil {
		global.DB.Model(&user).Updates(map[string]interface{}{
			"login_fail_count": 0,
			"locked_until":     nil,
		})
	}

	// 7. 生成token
	j := util.NewJWT()
	claims := j.CreateClaims(util.CustomClaims{
		UserID:          user.ID,
		Username:        user.Username,
		RoleID:          user.RoleID,
		PasswordVersion: user.PasswordVersion,
	})
	token, err := j.CreateToken(claims)
	if err != nil {
		return nil, res.NewAppErrorWithErr(res.ErrorCodeInternalServer, res.GetErrorMsg(res.ErrorCodeInternalServer), err)
	}

	// 8. 更新登录信息
	now := time.Now().Format("2006-01-02 15:04:05")
	if err := global.DB.Model(&user).Updates(map[string]interface{}{
		"last_login_ip": clientIP,
		"last_login_at": now,
	}).Error; err != nil {
		global.Log.Errorf("更新登录信息失败: %v", err)
	}

	// 记录登录成功日志
	s.recordLoginLog(user.Username, clientIP, userAgent, 1, "登录成功")

	// 隐藏密码
	user.Password = ""
	// 设置前端需要的 roles 字段
	user.Roles = []string{user.Role.RoleCode}

	return &models.SystemUserLoginRes{
		Token:     token,
		ExpiresAt: claims.ExpiresAt.Unix(),
		UserInfo:  user,
	}, nil
}

// handleLoginFailure 处理登录失败，增加失败计数并可能锁定账户
func (s *SystemUserService) handleLoginFailure(userID uint, username string) {
	// 使用数据库原子操作增加失败计数，防止并发覆盖
	if err := global.DB.Model(&models.SystemUser{}).Where("id = ?", userID).UpdateColumn("login_fail_count", gorm.Expr("login_fail_count + ?", 1)).Error; err != nil {
		global.Log.Errorf("更新登录失败计数失败: %v", err)
		return
	}

	// 重新查询最新的失败计数
	var updatedUser models.SystemUser
	if err := global.DB.Select("login_fail_count").First(&updatedUser, userID).Error; err != nil {
		global.Log.Errorf("查询最新登录失败计数失败: %v", err)
		return
	}

	// 如果连续失败达到5次，锁定账户30分钟
	if updatedUser.LoginFailCount >= 5 {
		lockedUntil := models.LocalTime(time.Now().Add(30 * time.Minute))
		if err := global.DB.Model(&models.SystemUser{}).Where("id = ?", userID).Update("locked_until", lockedUntil).Error; err != nil {
			global.Log.Errorf("锁定账户失败: %v", err)
			return
		}
		global.Log.Warnf("用户[%s]连续登录失败%d次，账户已锁定30分钟", username, updatedUser.LoginFailCount)
	}
}

// recordLoginLog 记录登录日志
func (s *SystemUserService) recordLoginLog(username, ip, userAgent string, status int, message string) {
	// 检查是否开启登录日志
	var settingService SystemSettingService
	if !settingService.IsLoginLogEnabled() {
		return // 未开启则不记录
	}

	// 解析 User-Agent 获取浏览器和操作系统
	browser, os := util.ParseUserAgent(userAgent)
	// 获取 IP 地理位置
	location := util.GetIPLocation(ip)

	log := models.LoginLog{
		Username:  username,
		IP:        ip,
		Location:  location,
		Browser:   browser,
		OS:        os,
		Status:    status,
		Message:   message,
		CreatedAt: models.LocalTime(time.Now()),
	}
	// 使用日志通道异步记录，防止goroutine泄露
	select {
	case loginLogChan <- log:
		// 成功发送到队列
	default:
		// 队列已满，记录警告
		global.Log.Warn("登录日志队列已满，丢弃日志记录")
	}
}

// 登录日志通道
var loginLogChan = make(chan models.LoginLog, 500)

func init() {
	// 启动登录日志工作协程
	for i := 0; i < 3; i++ {
		go func() {
			for log := range loginLogChan {
				if err := global.DB.Create(&log).Error; err != nil {
					global.Log.Errorf("记录登录日志失败: %v", err)
				}
			}
		}()
	}
}

// GetUserInfo 获取当前用户信息
func (s *SystemUserService) GetUserInfo(userID uint) (*models.SystemUser, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	user.Password = ""
	user.Roles = []string{user.Role.RoleCode}
	return user, nil
}

// GetAsyncRoutes 获取当前用户的动态路由菜单
func (s *SystemUserService) GetAsyncRoutes(userID uint) ([]map[string]interface{}, error) {
	// 查询用户信息及角色
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, res.NewAppErrorByCode(res.ErrorCodeUserNotExist)
	}

	// 查询角色的菜单权限（所有角色都根据权限表查询，包括超级管理员）
	var menuIDs []uint
	if err := global.DB.Model(&models.SystemRoleMenu{}).Where("role_id = ?", user.RoleID).Pluck("menu_id", &menuIDs).Error; err != nil {
		global.Log.Errorf("查询角色菜单权限失败: %v", err)
	}

	// 查询菜单详情，同时包含所有父级目录
	menus := s.getMenusWithParents(menuIDs)

	// 转换为前端路由格式
	routes := s.buildRoutesFromMenus(menus, user.Role.RoleCode)

	return routes, nil
}

// getMenusWithParents 获取菜单及其所有父级目录
func (s *SystemUserService) getMenusWithParents(menuIDs []uint) []models.SystemMenu {
	if len(menuIDs) == 0 {
		return nil
	}

	// 获取系统设置，检查日志菜单是否应该显示
	var settingService SystemSettingService
	setting, _ := settingService.GetSetting()
	showOperationLog := setting.EnableOperationLog == 1
	showLoginLog := setting.EnableLoginLog == 1
	
	// 查询应该隐藏的菜单ID（操作日志和登录日志菜单）
	var hiddenMenuIDs []uint
	if !showOperationLog || !showLoginLog {
		var logMenus []models.SystemMenu
		pathPatterns := []string{}
		if !showOperationLog {
			pathPatterns = append(pathPatterns, "/system/log/operation")
		}
		if !showLoginLog {
			pathPatterns = append(pathPatterns, "/system/log/login")
		}
		if len(pathPatterns) > 0 {
			global.DB.Where("path IN ?", pathPatterns).Find(&logMenus)
			for _, menu := range logMenus {
				hiddenMenuIDs = append(hiddenMenuIDs, menu.ID)
			}
		}
	}
	
	// 将隐藏的菜单ID从查询列表中移除
	filteredMenuIDs := make([]uint, 0, len(menuIDs))
	for _, id := range menuIDs {
		shouldHide := false
		for _, hiddenID := range hiddenMenuIDs {
			if id == hiddenID {
				shouldHide = true
				break
			}
		}
		if !shouldHide {
			filteredMenuIDs = append(filteredMenuIDs, id)
		}
	}
	
	// 使用map去重
	menuMap := make(map[uint]models.SystemMenu)
	
	// 当前层级的菜单ID
	currentIDs := filteredMenuIDs
	
	// 最多循环10层，防止无限循环
	for i := 0; i < 10 && len(currentIDs) > 0; i++ {
		var menus []models.SystemMenu
		if err := global.DB.Where("id IN ?", currentIDs).Find(&menus).Error; err != nil {
			global.Log.Errorf("查询菜单失败: %v", err)
			break
		}
		
		// 下一层需要查询的父级ID
		nextIDs := []uint{}
		
		for _, menu := range menus {
			// 如果已经存在，跳过
			if _, exists := menuMap[menu.ID]; exists {
				continue
			}
			
			menuMap[menu.ID] = menu
			
			// 如果有父级且父级未被查询过，加入下一层查询
			if menu.ParentID > 0 {
				if _, exists := menuMap[menu.ParentID]; !exists {
					nextIDs = append(nextIDs, menu.ParentID)
				}
			}
		}
		
		currentIDs = nextIDs
	}
	
	// 将map转换为slice，并按sort字段排序（保证顺序稳定）
	result := make([]models.SystemMenu, 0, len(menuMap))
	for _, menu := range menuMap {
		result = append(result, menu)
	}
	
	// 按sort字段升序排序
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Sort > result[j].Sort {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	
	return result
}

// buildRoutesFromMenus 将菜单列表转换为前端路由格式
func (s *SystemUserService) buildRoutesFromMenus(menus []models.SystemMenu, roleCode string) []map[string]interface{} {
	var routes []map[string]interface{}

	// 构建菜单树
	menuTree := s.buildMenuTreeForRoutes(menus, 0)

	// 转换为前端路由格式
	for _, menu := range menuTree {
		route := s.menuToRoute(menu, roleCode)
		if route != nil {
			routes = append(routes, route)
		}
	}

	return routes
}

// buildMenuTreeForRoutes 构建菜单树（用于路由）
func (s *SystemUserService) buildMenuTreeForRoutes(menus []models.SystemMenu, parentId uint) []map[string]interface{} {
	var tree []map[string]interface{}
	for _, menu := range menus {
		if menu.ParentID == parentId {
			// 只添加启用的菜单
			if menu.Status != 1 {
				continue
			}
			item := map[string]interface{}{
				"id":        menu.ID,
				"parentId":  menu.ParentID,
				"menuName":  menu.MenuName,
				"menuType":  menu.MenuType,
				"icon":      menu.Icon,
				"path":      menu.Path,
				"component": menu.Component,
				"perm":      menu.Perm,
				"sort":      menu.Sort,
				"status":    menu.Status,
				"visible":   menu.Visible,
			}
			children := s.buildMenuTreeForRoutes(menus, menu.ID)
			if len(children) > 0 {
				item["children"] = children
			}
			tree = append(tree, item)
		}
	}
	return tree
}

// menuToRoute 将菜单转换为前端路由格式
func (s *SystemUserService) menuToRoute(menu map[string]interface{}, roleCode string) map[string]interface{} {
	// 处理menuType的类型（可能是int或float64，因为从JSON转换）
	var menuType int
	switch v := menu["menuType"].(type) {
	case int:
		menuType = v
	case int8:
		menuType = int(v)
	case int16:
		menuType = int(v)
	case int32:
		menuType = int(v)
	case int64:
		menuType = int(v)
	case float64:
		menuType = int(v)
	default:
		menuType = 1 // 默认目录
	}

	// 按钮不生成路由
	if menuType == 3 {
		return nil
	}

	path, _ := menu["path"].(string)
	menuName, _ := menu["menuName"].(string)
	icon, _ := menu["icon"].(string)
	component, _ := menu["component"].(string)

	// 构建meta
	meta := map[string]interface{}{
		"title": menuName,
		"icon":  icon,
	}

	// 目录和菜单都显示在侧边栏
	meta["showLink"] = true

	// 添加角色权限（当前用户的角色）
	meta["roles"] = []string{roleCode}

	// 构建路由
	route := map[string]interface{}{
		"path": path,
		"meta": meta,
	}

	// 设置name（目录加Parent后缀，避免和子菜单冲突）
	if menuType == 1 {
		route["name"] = menuName + "Parent"
	} else {
		route["name"] = menuName
	}

	// 设置组件
	if component != "" {
		route["component"] = component
	}

	// 处理子路由
	if children, ok := menu["children"].([]map[string]interface{}); ok && len(children) > 0 {
		var childRoutes []map[string]interface{}
		for _, child := range children {
			childRoute := s.menuToRoute(child, roleCode)
			if childRoute != nil {
				childRoutes = append(childRoutes, childRoute)
			}
		}
		if len(childRoutes) > 0 {
			route["children"] = childRoutes
			// 目录添加redirect指向第一个子菜单
			if menuType == 1 {
				firstChildPath, _ := childRoutes[0]["path"].(string)
				route["redirect"] = firstChildPath
			}
		}
	}

	return route
}

// ==================== 当前用户相关 ====================

// UpdateCurrentUser 更新当前用户信息（只允许修改昵称、头像、手机号、邮箱）
func (s *SystemUserService) UpdateCurrentUser(userID uint, req *models.SystemUserProfileReq) error {
	var user models.SystemUser
	if err := global.DB.First(&user, userID).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{}

	// 仅更新客户端明确传入的字段（指针非 nil 表示传入）
	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}

	if len(updates) == 0 {
		return nil
	}

	return global.DB.Model(&user).Updates(updates).Error
}

// UpdateCurrentUserPassword 更新当前用户密码
func (s *SystemUserService) UpdateCurrentUserPassword(userID uint, oldPassword, newPassword string) error {
	var user models.SystemUser
	if err := global.DB.First(&user, userID).Error; err != nil {
		return err
	}

	// 验证原密码
	if !util.BcryptCheck(oldPassword, user.Password) {
		return errors.New("原密码不正确")
	}

	// 更新密码并递增密码版本号（使旧 Token 失效）
	return global.DB.Model(&user).Updates(map[string]interface{}{
		"password":         util.BcryptHash(newPassword),
		"password_version": gorm.Expr("password_version + ?", 1),
	}).Error
}

// ==================== 内部辅助方法 ====================

// 为了兼容现有错误处理
var ErrUserNotExist = errors.New("用户不存在")
var ErrInvalidPassword = errors.New("密码错误")
var ErrUserDisabled = errors.New("用户已被禁用")
