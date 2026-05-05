package v1

import (
	"errors"
	"go-vue-admin/models"
	"go-vue-admin/models/res"
	"go-vue-admin/services/v1"
	"go-vue-admin/util"

	"github.com/gin-gonic/gin"
)

type SystemRoleApi struct{}

// SetRoleMenusReq 设置角色菜单权限请求（复用 service 层的定义）
type SetRoleMenusReq = v1.SetRoleMenusReq

// CreateRoleReq 复用 models 包中的定义
type CreateRoleReq = models.CreateRoleReq

// UpdateRoleReq 复用 models 包中的定义
type UpdateRoleReq = models.UpdateRoleReq

// CreateMenuReq 复用 models 包中的定义
type CreateMenuReq = models.CreateMenuReq

// UpdateMenuReq 复用 models 包中的定义
type UpdateMenuReq = models.UpdateMenuReq

// ==================== 角色管理 ====================

// GetRoleList
// @Tags 系统管理-角色
// @Summary 获取角色列表
// @Description 分页获取角色列表，支持关键词搜索
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页数量，默认10，最大100"
// @Param keyword query string false "关键词（角色名称/角色代码）"
// @Success 200 {object} res.Response{data=res.PageResult{list=[]models.SystemRole}} "成功"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/role/list [get]
func (a *SystemRoleApi) GetRoleList(c *gin.Context) {
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

	roles, total, err := systemRoleService.GetRoleList(page, pageSize, keyword)
	if err != nil {
		res.Error(c, err)
		return
	}

	res.PageSuccess(c, roles, total, page, pageSize)
}

// GetRoleOptions
// @Tags 系统管理-角色
// @Summary 获取角色选项列表
// @Description 获取角色选项列表（排除超级管理员，用于下拉选择）
// @Produce json
// @Security BearerAuth
// @Success 200 {object} res.Response{data=[]models.SystemRole} "成功"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/role/options [get]
func (a *SystemRoleApi) GetRoleOptions(c *gin.Context) {
	roles, err := systemRoleService.GetRoleOptions()
	if err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, roles)
}

// GetRoleDetail
// @Tags 系统管理-角色
// @Summary 获取角色详情
// @Description 根据ID获取角色详细信息
// @Produce json
// @Security BearerAuth
// @Param id path int true "角色ID"
// @Success 200 {object} res.Response{data=models.SystemRole} "成功"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Failure 404 {object} res.Response "角色不存在"
// @Router /api/v1/system/role/detail/{id} [get]
func (a *SystemRoleApi) GetRoleDetail(c *gin.Context) {
	id := util.StringToUint(c.Param("id"))
	if id == 0 {
		res.Fail(c, res.ErrorCodeParamInvalid)
		return
	}

	role, err := systemRoleService.GetRoleByID(id)
	if err != nil {
		res.Fail(c, res.ErrorCodeNotFound)
		return
	}

	res.Success(c, role)
}

// CreateRole
// @Tags 系统管理-角色
// @Summary 创建角色
// @Description 创建新角色，角色代码不能重复
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body CreateRoleReq true "角色数据"
// @Success 200 {object} res.Response{data=uint} "创建成功，返回角色ID"
// @Failure 400 {object} res.Response "请求参数错误"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/role/create [post]
func (a *SystemRoleApi) CreateRole(c *gin.Context) {
	var req CreateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		res.ValidationError(c, err.Error())
		return
	}

	// 检查角色代码是否已存在
	if systemRoleService.CheckRoleCodeExist(req.RoleCode) {
		res.FailWithMessage(c, res.ErrorCodeBusinessError, "角色代码已存在")
		return
	}

	// 转换到模型
	role := &models.SystemRole{
		RoleName:    req.RoleName,
		RoleCode:    req.RoleCode,
		Description: req.Description,
		Status:      req.Status,
		Sort:        req.Sort,
	}

	id, err := systemRoleService.CreateRole(role)
	if err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, id)
}

// UpdateRole
// @Tags 系统管理-角色
// @Summary 更新角色
// @Description 更新角色信息（角色代码不能修改）
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body UpdateRoleReq true "角色数据"
// @Success 200 {object} res.Response "更新成功"
// @Failure 400 {object} res.Response "请求参数错误"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/role/update [put]
func (a *SystemRoleApi) UpdateRole(c *gin.Context) {
	var req UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		res.ValidationError(c, err.Error())
		return
	}

	if req.ID == 0 {
		res.Fail(c, res.ErrorCodeParamInvalid)
		return
	}

	// 检查是否是系统保留角色
	existingRole, err := systemRoleService.GetRoleByID(req.ID)
	if err != nil {
		res.Fail(c, res.ErrorCodeNotFound)
		return
	}
	
	// 系统保留角色（admin）只允许修改部分字段
	if existingRole.RoleCode == "admin" {
		// 可以添加更多限制逻辑
	}

	// 转换到模型
	role := &models.SystemRole{
		ID:          req.ID,
		RoleName:    req.RoleName,
		Description: req.Description,
		Status:      req.Status,
		Sort:        req.Sort,
	}

	if err := systemRoleService.UpdateRole(role); err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, nil)
}

// DeleteRole
// @Tags 系统管理-角色
// @Summary 删除角色
// @Description 根据ID删除角色（有关联用户时无法删除，系统保留角色无法删除）
// @Produce json
// @Security BearerAuth
// @Param id path int true "角色ID"
// @Success 200 {object} res.Response "删除成功"
// @Failure 400 {object} res.Response "该角色下存在用户，无法删除/系统保留角色无法删除"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/role/delete/{id} [delete]
func (a *SystemRoleApi) DeleteRole(c *gin.Context) {
	id := util.StringToUint(c.Param("id"))
	if id == 0 {
		res.Fail(c, res.ErrorCodeParamInvalid)
		return
	}

	// 检查是否是系统保留角色
	role, err := systemRoleService.GetRoleByID(id)
	if err != nil {
		res.Fail(c, res.ErrorCodeNotFound)
		return
	}
	
	// 系统保留角色无法删除
	if role.RoleCode == "admin" {
		res.FailWithMessage(c, res.ErrorCodeBusinessError, "系统保留角色无法删除")
		return
	}

	if err := systemRoleService.DeleteRole(id); err != nil {
		if errors.Is(err, v1.ErrRoleHasUsers) {
			res.FailWithMessage(c, res.ErrorCodeBusinessError, err.Error())
			return
		}
		res.Error(c, err)
		return
	}

	res.Success(c, nil)
}

// GetRoleMenus
// @Tags 系统管理-角色
// @Summary 获取角色的菜单权限
// @Description 获取指定角色关联的菜单ID列表
// @Produce json
// @Security BearerAuth
// @Param id path int true "角色ID"
// @Success 200 {object} res.Response{data=[]uint} "成功"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/role/menus/{id} [get]
func (a *SystemRoleApi) GetRoleMenus(c *gin.Context) {
	id := util.StringToUint(c.Param("id"))
	if id == 0 {
		res.Fail(c, res.ErrorCodeParamInvalid)
		return
	}

	menuIDs, err := systemRoleService.GetRoleMenus(id)
	if err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, menuIDs)
}

// SetRoleMenus
// @Tags 系统管理-角色
// @Summary 设置角色的菜单权限
// @Description 为角色分配菜单权限
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body v1.SetRoleMenusReq true "权限数据"
// @Success 200 {object} res.Response "设置成功"
// @Failure 400 {object} res.Response "请求参数错误"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/role/menus [put]
func (a *SystemRoleApi) SetRoleMenus(c *gin.Context) {
	var req SetRoleMenusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		res.ValidationError(c, err.Error())
		return
	}

	if err := systemRoleService.SetRoleMenus(&req); err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, nil)
}

// ==================== 菜单管理 ====================

// GetMenuList
// @Tags 系统管理-菜单
// @Summary 获取菜单列表
// @Description 获取所有菜单列表（扁平结构）
// @Produce json
// @Security BearerAuth
// @Success 200 {object} res.Response{data=[]models.SystemMenu} "成功"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/menu/list [get]
func (a *SystemRoleApi) GetMenuList(c *gin.Context) {
	menus, err := systemRoleService.GetMenuList()
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, menus)
}

// GetMenuTree
// @Tags 系统管理-菜单
// @Summary 获取菜单树
// @Description 获取菜单树形结构（层级结构）
// @Produce json
// @Security BearerAuth
// @Success 200 {object} res.Response{data=[]map[string]interface{}} "成功"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/menu/tree [get]
func (a *SystemRoleApi) GetMenuTree(c *gin.Context) {
	menuTree, err := systemRoleService.GetMenuTree()
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, menuTree)
}

// CreateMenu
// @Tags 系统管理-菜单
// @Summary 创建菜单
// @Description 创建新菜单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body CreateMenuReq true "菜单数据"
// @Success 200 {object} res.Response{data=uint} "创建成功，返回菜单ID"
// @Failure 400 {object} res.Response "请求参数错误"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/menu/create [post]
func (a *SystemRoleApi) CreateMenu(c *gin.Context) {
	var req CreateMenuReq
	if err := c.ShouldBindJSON(&req); err != nil {
		res.ValidationError(c, err.Error())
		return
	}

	// 转换到模型
	menu := &models.SystemMenu{
		ParentID:  req.ParentID,
		MenuName:  req.MenuName,
		MenuType:  req.MenuType,
		Icon:      req.Icon,
		Path:      req.Path,
		Component: req.Component,
		Perm:      req.Perm,
		Sort:      req.Sort,
		Status:    req.Status,
		Visible:   req.Visible,
	}

	id, err := systemRoleService.CreateMenu(menu)
	if err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, id)
}

// UpdateMenu
// @Tags 系统管理-菜单
// @Summary 更新菜单
// @Description 更新菜单信息
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body UpdateMenuReq true "菜单数据"
// @Success 200 {object} res.Response "更新成功"
// @Failure 400 {object} res.Response "请求参数错误"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/menu/update [put]
func (a *SystemRoleApi) UpdateMenu(c *gin.Context) {
	var req UpdateMenuReq
	if err := c.ShouldBindJSON(&req); err != nil {
		res.ValidationError(c, err.Error())
		return
	}

	if req.ID == 0 {
		res.Fail(c, res.ErrorCodeParamInvalid)
		return
	}

	// 转换到模型
	menu := &models.SystemMenu{
		ID:        req.ID,
		ParentID:  req.ParentID,
		MenuName:  req.MenuName,
		MenuType:  req.MenuType,
		Icon:      req.Icon,
		Path:      req.Path,
		Component: req.Component,
		Perm:      req.Perm,
		Sort:      req.Sort,
		Status:    req.Status,
		Visible:   req.Visible,
	}

	if err := systemRoleService.UpdateMenu(menu); err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, nil)
}

// DeleteMenu
// @Tags 系统管理-菜单
// @Summary 删除菜单
// @Description 根据ID删除菜单（有子菜单时无法删除）
// @Produce json
// @Security BearerAuth
// @Param id path int true "菜单ID"
// @Success 200 {object} res.Response "删除成功"
// @Failure 400 {object} res.Response "该菜单下存在子菜单，无法删除"
// @Failure 401 {object} res.Response "未登录或token过期"
// @Router /api/v1/system/menu/delete/{id} [delete]
func (a *SystemRoleApi) DeleteMenu(c *gin.Context) {
	id := util.StringToUint(c.Param("id"))
	if id == 0 {
		res.Fail(c, res.ErrorCodeParamInvalid)
		return
	}

	if err := systemRoleService.DeleteMenu(id); err != nil {
		if errors.Is(err, v1.ErrMenuHasChildren) {
			res.FailWithMessage(c, res.ErrorCodeBusinessError, err.Error())
			return
		}
		res.Error(c, err)
		return
	}

	res.Success(c, nil)
}
