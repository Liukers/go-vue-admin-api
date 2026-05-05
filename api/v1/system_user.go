package v1

import (
	"strings"
	"time"
	"go-vue-admin/global"
	"go-vue-admin/middleware"
	"go-vue-admin/models"
	"go-vue-admin/models/res"
	"go-vue-admin/util"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type SystemUserApi struct{}

// ==================== 认证相关 ====================

// Login
// @Tags 系统管理-认证
// @Summary 用户登录
// @Description 用户登录接口，需要验证码，返回JWT token
// @Accept json
// @Produce json
// @Param data body models.SystemUserLoginReq true "登录参数（含验证码）"
// @Success 200 {object} res.Response{data=models.SystemUserLoginRes} "登录成功"
// @Failure 400 {object} res.Response "请求参数错误/验证码错误"
// @Failure 401 {object} res.Response "登录失败，用户名或密码错误"
// @Failure 423 {object} res.Response "账户已被锁定"
// @Router /api/v1/system/login [post]
func (a *SystemUserApi) Login(c *gin.Context) {
	var req models.SystemUserLoginReq
	if err := c.ShouldBindWith(&req, binding.JSON); err != nil {
		res.ValidationError(c, err.Error())
		return
	}

	// 获取客户端信息
	userAgent := c.Request.UserAgent()
	resp, err := systemUserService.Login(&req, c.ClientIP(), userAgent)
	if err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, resp)
}

// Logout
// @Tags 系统管理-认证
// @Summary 用户登出
// @Description 用户登出，token将被加入黑名单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} res.Response "登出成功"
// @Router /api/v1/system/logout [post]
func (a *SystemUserApi) Logout(c *gin.Context) {
	middleware.LogoutHandler(c)
}

// RefreshToken
// @Tags 系统管理-认证
// @Summary 刷新Token
// @Description 使用当前token刷新，返回新的token和过期时间。刷新前会校验token有效性、黑名单、用户状态及密码版本。
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} res.Response{data=models.SystemUserLoginRes} "刷新成功"
// @Failure 401 {object} res.Response "token无效或已过期/已被注销/用户已禁用"
// @Router /api/v1/system/refresh-token [post]
func (a *SystemUserApi) RefreshToken(c *gin.Context) {
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		res.Unauthorized(c, "请求未携带token")
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "token格式错误")
		return
	}

	tokenString := parts[1]
	j := util.NewJWT()

	// 1. 解析旧token
	claims, err := j.ParseToken(tokenString)
	if err != nil {
		res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "token无效: "+err.Error())
		return
	}

	// 2. 检查黑名单
	tb := &middleware.TokenBlacklist{}
	if tb.IsBlacklisted(tokenString) {
		res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "token已被注销")
		return
	}

	// 3. 查询用户并校验状态、密码版本、锁定状态
	var user models.SystemUser
	if err := global.DB.Select("password_version", "status", "locked_until", "role_id").First(&user, claims.UserID).Error; err != nil {
		res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "用户不存在")
		return
	}

	if user.PasswordVersion != claims.PasswordVersion {
		res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "密码已修改，请重新登录")
		return
	}

	if user.Status != 1 {
		res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "用户已被禁用")
		return
	}

	if user.LockedUntil != nil && time.Time(*user.LockedUntil).After(time.Now()) {
		res.FailWithMessage(c, res.ErrorCodeAccountLocked, "账户已被锁定")
		return
	}

	// 4. 刷新token
	newToken, err := j.RefreshToken(tokenString)
	if err != nil {
		res.FailWithMessage(c, res.ErrorCodeTokenInvalid, "token刷新失败: "+err.Error())
		return
	}

	// 解析新token获取过期时间
	claims, _ = j.ParseToken(newToken)
	var expiresAt int64
	if claims != nil && claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Unix()
	}

	res.Success(c, map[string]interface{}{
		"token":     newToken,
		"expiresAt": expiresAt,
	})
}

// GetUserInfo
// @Tags 系统管理-认证
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息
// @Produce json
// @Security BearerAuth
// @Success 200 {object} res.Response{data=models.SystemUser} "成功"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/users/info [get]
func (a *SystemUserApi) GetUserInfo(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		res.Fail(c, res.ErrorCodeUnauthorized)
		return
	}
	
	// 安全的类型断言
	uid, ok := userId.(uint)
	if !ok {
		res.Fail(c, res.ErrorCodeUnauthorized)
		return
	}

	user, err := systemUserService.GetUserInfo(uid)
	if err != nil {
		res.Fail(c, res.ErrorCodeUserNotExist)
		return
	}

	res.Success(c, user)
}

// GetAsyncRoutes
// @Tags 系统管理-认证
// @Summary 获取动态路由
// @Description 获取当前登录用户的动态路由菜单
// @Produce json
// @Security BearerAuth
// @Success 200 {object} res.Response{data=[]map[string]interface{}} "成功"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/routes [get]
func (a *SystemUserApi) GetAsyncRoutes(c *gin.Context) {
	// 从JWT中获取用户ID
	userId, exists := c.Get("userId")
	if !exists {
		res.Fail(c, res.ErrorCodeUnauthorized)
		return
	}

	// 安全的类型断言
	uid, ok := userId.(uint)
	if !ok {
		res.Fail(c, res.ErrorCodeUnauthorized)
		return
	}

	routes, err := systemUserService.GetAsyncRoutes(uid)
	if err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, routes)
}

// ==================== 用户管理 ====================

// GetUserList
// @Tags 系统管理-用户
// @Summary 获取用户列表
// @Description 分页获取系统用户列表，支持关键词搜索和状态筛选
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页数量，默认10，最大100"
// @Param keyword query string false "关键词（用户名/昵称/手机号）"
// @Param status query string false "状态：1启用 2禁用"
// @Success 200 {object} res.Response{data=res.PageResult{list=[]models.SystemUser}} "成功"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/users [get]
func (a *SystemUserApi) GetUserList(c *gin.Context) {
	page := util.StringToInt(c.DefaultQuery("page", "1"))
	pageSize := util.StringToInt(c.DefaultQuery("pageSize", "10"))
	
	// 限制分页大小，防止性能问题
	const maxPageSize = 100
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if page < 1 {
		page = 1
	}
	
	keyword := c.Query("keyword")
	status := c.Query("status")

	users, total, err := systemUserService.GetUserList(page, pageSize, keyword, status)
	if err != nil {
		res.Error(c, err)
		return
	}

	res.PageSuccess(c, users, total, page, pageSize)
}

// CreateUser
// @Tags 系统管理-用户
// @Summary 创建用户
// @Description 创建新用户，用户名不能重复
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body models.SystemUserReq true "用户数据"
// @Success 200 {object} res.Response{data=uint} "创建成功，返回用户ID"
// @Failure 400 {object} res.Response "请求参数错误"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/users [post]
func (a *SystemUserApi) CreateUser(c *gin.Context) {
	var req models.SystemUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		res.ValidationError(c, err.Error())
		return
	}

	// 检查用户名是否已存在
	if systemUserService.CheckUserExist(req.Username) {
		res.Fail(c, res.ErrorCodeUserExist)
		return
	}

	id, err := systemUserService.CreateUser(&req)
	if err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, id)
}

// UpdateUser
// @Tags 系统管理-用户
// @Summary 更新用户
// @Description 更新用户信息，用户名不能与其他用户重复。管理员可通过此接口修改密码（传入 password 字段）。
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param data body models.SystemUserUpdateReq true "用户数据"
// @Success 200 {object} res.Response "更新成功"
// @Failure 400 {object} res.Response "请求参数错误"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Failure 404 {object} res.Response "用户不存在"
// @Router /api/v1/system/users/{id} [put]
func (a *SystemUserApi) UpdateUser(c *gin.Context) {
	id := util.StringToUint(c.Param("id"))
	if id == 0 {
		res.Fail(c, res.ErrorCodeParamInvalid)
		return
	}

	var req models.SystemUserUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		res.ValidationError(c, err.Error())
		return
	}

	// 将路径参数的ID设置到请求体
	req.ID = id

	// 检查用户是否存在
	if _, err := systemUserService.GetUserByID(req.ID); err != nil {
		res.Fail(c, res.ErrorCodeUserNotExist)
		return
	}

	// 如果要修改用户名，检查是否与其他用户冲突
	if req.Username != "" && systemUserService.CheckUserExistExceptID(req.Username, req.ID) {
		res.Fail(c, res.ErrorCodeUserExist)
		return
	}

	if err := systemUserService.UpdateUser(&req); err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, nil)
}

// DeleteUser
// @Tags 系统管理-用户
// @Summary 删除用户
// @Description 根据ID删除用户（软删除）
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Success 200 {object} res.Response "删除成功"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Failure 404 {object} res.Response "用户不存在"
// @Router /api/v1/system/users/{id} [delete]
func (a *SystemUserApi) DeleteUser(c *gin.Context) {
	id := util.StringToUint(c.Param("id"))
	if id == 0 {
		res.Fail(c, res.ErrorCodeParamInvalid)
		return
	}

	// 禁止删除当前登录用户自身
	currentUserId, exists := c.Get("userId")
	if exists {
		if uid, ok := currentUserId.(uint); ok && uid == id {
			res.FailWithMessage(c, res.ErrorCodeBusinessError, "不能删除当前登录用户")
			return
		}
	}

	// 禁止删除最后一个超级管理员
	var targetUser models.SystemUser
	if err := global.DB.Preload("Role").First(&targetUser, id).Error; err != nil {
		res.Fail(c, res.ErrorCodeUserNotExist)
		return
	}
	if targetUser.Role.RoleCode == "admin" {
		var adminRole models.SystemRole
		if err := global.DB.Where("role_code = ?", "admin").First(&adminRole).Error; err == nil {
			var adminCount int64
			global.DB.Model(&models.SystemUser{}).Where("role_id = ?", adminRole.ID).Count(&adminCount)
			if adminCount <= 1 {
				res.FailWithMessage(c, res.ErrorCodeBusinessError, "不能删除最后一个超级管理员")
				return
			}
		}
	}

	if err := systemUserService.DeleteUser(id); err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, nil)
}

// UpdateCurrentUser
// @Tags 系统管理-用户
// @Summary 更新当前用户信息
// @Description 当前登录用户更新自己的个人信息（昵称、头像、手机号、邮箱）
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body models.SystemUserProfileReq true "个人信息"
// @Success 200 {object} res.Response "更新成功"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/user/profile [put]
func (a *SystemUserApi) UpdateCurrentUser(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		res.Fail(c, res.ErrorCodeUnauthorized)
		return
	}
	
	// 安全的类型断言
	uid, ok := userId.(uint)
	if !ok {
		res.Fail(c, res.ErrorCodeUnauthorized)
		return
	}

	var req models.SystemUserProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		res.ValidationError(c, err.Error())
		return
	}

	if err := systemUserService.UpdateCurrentUser(uid, &req); err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, nil)
}

// UpdateCurrentUserPassword
// @Tags 系统管理-用户
// @Summary 修改当前用户密码
// @Description 当前登录用户修改自己的密码，需要验证原密码
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body models.SystemUserPasswordReq true "密码数据"
// @Success 200 {object} res.Response "修改成功"
// @Failure 400 {object} res.Response "请求参数错误"
// @Failure 401 {object} res.Response "未登录或token过期/原密码错误"
// @Router /api/v1/system/user/password [put]
func (a *SystemUserApi) UpdateCurrentUserPassword(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		res.Fail(c, res.ErrorCodeUnauthorized)
		return
	}
	
	// 安全的类型断言
	uid, ok := userId.(uint)
	if !ok {
		res.Fail(c, res.ErrorCodeUnauthorized)
		return
	}

	var req models.SystemUserPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		res.ValidationError(c, err.Error())
		return
	}

	if err := systemUserService.UpdateCurrentUserPassword(uid, req.OldPassword, req.NewPassword); err != nil {
		res.FailWithMessage(c, res.ErrorCodeBusinessError, err.Error())
		return
	}

	res.Success(c, nil)
}
