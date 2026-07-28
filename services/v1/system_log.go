package v1

import (
	"go-vue-admin/global"
	"go-vue-admin/models"
)

type SystemLogService struct{}

// GetOperationLogList 获取操作日志列表
func (s *SystemLogService) GetOperationLogList(page, pageSize int, username, status, startTime, endTime string) ([]models.OperationLog, int64, error) {
	var logs []models.OperationLog
	var total int64

	db := global.DB.Model(&models.OperationLog{})

	if username != "" {
		db = db.Where("username LIKE ?", "%"+username+"%")
	}

	if status != "" {
		db = db.Where("status = ?", status)
	}

	if startTime != "" {
		db = db.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		db = db.Where("created_at <= ?", endTime+" 23:59:59")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error

	return logs, total, err
}

// GetLoginLogList 获取登录日志列表
func (s *SystemLogService) GetLoginLogList(page, pageSize int, username, status, startTime, endTime string) ([]models.LoginLog, int64, error) {
	var logs []models.LoginLog
	var total int64

	db := global.DB.Model(&models.LoginLog{})

	if username != "" {
		db = db.Where("username LIKE ?", "%"+username+"%")
	}

	if status != "" {
		db = db.Where("status = ?", status)
	}

	if startTime != "" {
		db = db.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		db = db.Where("created_at <= ?", endTime+" 23:59:59")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error

	return logs, total, err
}

// DeleteOperationLog 删除操作日志
func (s *SystemLogService) DeleteOperationLog(id uint) error {
	return global.DB.Delete(&models.OperationLog{}, id).Error
}

// DeleteLoginLog 删除登录日志
func (s *SystemLogService) DeleteLoginLog(id uint) error {
	return global.DB.Delete(&models.LoginLog{}, id).Error
}

// ClearOperationLog 清空操作日志
func (s *SystemLogService) ClearOperationLog() error {
	return global.DB.Exec("TRUNCATE TABLE system_operation_log").Error
}

// ClearLoginLog 清空登录日志
func (s *SystemLogService) ClearLoginLog() error {
	return global.DB.Exec("TRUNCATE TABLE system_login_log").Error
}
