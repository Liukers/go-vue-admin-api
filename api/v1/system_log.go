package v1

import (
	"go-vue-admin/models/res"
	"go-vue-admin/util"

	"github.com/gin-gonic/gin"
)

type SystemLogApi struct{}

// GetOperationLogList
// @Tags 系统管理-日志
// @Summary 获取操作日志列表
// @Description 分页获取操作日志，支持按用户、状态、时间筛选
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页数量，默认10，最大100"
// @Param username query string false "用户名"
// @Param status query int false "状态：1成功 2失败"
// @Param startTime query string false "开始时间（格式：2006-01-02）"
// @Param endTime query string false "结束时间（格式：2006-01-02）"
// @Success 200 {object} res.Response{data=res.PageResult{list=[]models.OperationLog}} "成功"
// @Router /api/v1/system/operation-logs [get]
func (a *SystemLogApi) GetOperationLogList(c *gin.Context) {
	page := util.StringToInt(c.DefaultQuery("page", "1"))
	pageSize := util.StringToInt(c.DefaultQuery("pageSize", "10"))
	username := c.Query("username")
	status := c.Query("status")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	if pageSize > 100 {
		pageSize = 100
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if page < 1 {
		page = 1
	}

	logs, total, err := systemLogService.GetOperationLogList(page, pageSize, username, status, startTime, endTime)
	if err != nil {
		res.Error(c, err)
		return
	}

	res.PageSuccess(c, logs, total, page, pageSize)
}

// GetLoginLogList
// @Tags 系统管理-日志
// @Summary 获取登录日志列表
// @Description 分页获取登录日志，支持按用户名、状态、时间筛选
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页数量，默认10，最大100"
// @Param username query string false "用户名"
// @Param status query int false "状态：1成功 2失败"
// @Param startTime query string false "开始时间（格式：2006-01-02）"
// @Param endTime query string false "结束时间（格式：2006-01-02）"
// @Success 200 {object} res.Response{data=res.PageResult{list=[]models.LoginLog}} "成功"
// @Router /api/v1/system/login-logs [get]
func (a *SystemLogApi) GetLoginLogList(c *gin.Context) {
	page := util.StringToInt(c.DefaultQuery("page", "1"))
	pageSize := util.StringToInt(c.DefaultQuery("pageSize", "10"))
	username := c.Query("username")
	status := c.Query("status")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	if pageSize > 100 {
		pageSize = 100
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if page < 1 {
		page = 1
	}

	logs, total, err := systemLogService.GetLoginLogList(page, pageSize, username, status, startTime, endTime)
	if err != nil {
		res.Error(c, err)
		return
	}

	res.PageSuccess(c, logs, total, page, pageSize)
}

// DeleteOperationLog
// @Tags 系统管理-日志
// @Summary 删除操作日志
// @Description 根据ID删除操作日志
// @Produce json
// @Security BearerAuth
// @Param id path int true "日志ID"
// @Success 200 {object} res.Response "删除成功"
// @Router /api/v1/system/operation-logs/{id} [delete]
func (a *SystemLogApi) DeleteOperationLog(c *gin.Context) {
	id := util.StringToUint(c.Param("id"))
	if id == 0 {
		res.Fail(c, res.ErrorCodeParamInvalid)
		return
	}

	if err := systemLogService.DeleteOperationLog(id); err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, nil)
}

// DeleteLoginLog
// @Tags 系统管理-日志
// @Summary 删除登录日志
// @Description 根据ID删除登录日志
// @Produce json
// @Security BearerAuth
// @Param id path int true "日志ID"
// @Success 200 {object} res.Response "删除成功"
// @Router /api/v1/system/login-logs/{id} [delete]
func (a *SystemLogApi) DeleteLoginLog(c *gin.Context) {
	id := util.StringToUint(c.Param("id"))
	if id == 0 {
		res.Fail(c, res.ErrorCodeParamInvalid)
		return
	}

	if err := systemLogService.DeleteLoginLog(id); err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, nil)
}

// ClearOperationLog
// @Tags 系统管理-日志
// @Summary 清空操作日志
// @Description 清空所有操作日志（危险操作）
// @Produce json
// @Security BearerAuth
// @Success 200 {object} res.Response "清空成功"
// @Router /api/v1/system/operation-logs [delete]
func (a *SystemLogApi) ClearOperationLog(c *gin.Context) {
	if err := systemLogService.ClearOperationLog(); err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, nil)
}

// ClearLoginLog
// @Tags 系统管理-日志
// @Summary 清空登录日志
// @Description 清空所有登录日志（危险操作）
// @Produce json
// @Security BearerAuth
// @Success 200 {object} res.Response "清空成功"
// @Router /api/v1/system/login-logs [delete]
func (a *SystemLogApi) ClearLoginLog(c *gin.Context) {
	if err := systemLogService.ClearLoginLog(); err != nil {
		res.Error(c, err)
		return
	}

	res.Success(c, nil)
}
