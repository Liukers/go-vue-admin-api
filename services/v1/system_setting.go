package v1

import (
	"errors"
	"sync"
	"go-vue-admin/global"
	"go-vue-admin/models"
	"time"
)

type SystemSettingService struct{}

// 系统设置内存缓存（设置项极少变更却被每个请求读取，缓存避免反复查库；
// 更新设置后主动失效，下次读取重新加载）
var (
	settingCache   *models.SystemSetting
	settingCacheMu sync.RWMutex
)

// GetSetting 获取系统设置（优先读缓存；如果不存在则创建默认设置）
func (s *SystemSettingService) GetSetting() (*models.SystemSetting, error) {
	settingCacheMu.RLock()
	if settingCache != nil {
		// 返回副本，防止调用方意外修改缓存内容
		cached := *settingCache
		settingCacheMu.RUnlock()
		return &cached, nil
	}
	settingCacheMu.RUnlock()

	var setting models.SystemSetting
	err := global.DB.First(&setting).Error
	if err != nil {
		// 如果没有记录，创建默认设置（默认关闭）
		setting = models.SystemSetting{
			EnableOperationLog: 2, // 默认关闭
			EnableLoginLog:     2, // 默认关闭
		}
		if createErr := global.DB.Create(&setting).Error; createErr != nil {
			return nil, createErr
		}
	}

	settingCacheMu.Lock()
	settingCache = &setting
	settingCacheMu.Unlock()
	return &setting, nil
}

// invalidateSettingCache 使设置缓存失效
func (s *SystemSettingService) invalidateSettingCache() {
	settingCacheMu.Lock()
	settingCache = nil
	settingCacheMu.Unlock()
}

// UpdateSetting 更新系统设置（成功后使缓存失效）
func (s *SystemSettingService) UpdateSetting(setting *models.SystemSetting) error {
	if setting == nil {
		return errors.New("设置信息不能为空")
	}

	var existing models.SystemSetting
	err := global.DB.First(&existing).Error
	if err != nil {
		setting.ID = 0 // 确保是新记录
		if err := global.DB.Create(setting).Error; err != nil {
			return err
		}
		s.invalidateSettingCache()
		return nil
	}

	setting.ID = existing.ID
	setting.CreatedAt = existing.CreatedAt
	if err := global.DB.Model(&existing).Updates(map[string]interface{}{
		"enable_operation_log": setting.EnableOperationLog,
		"enable_login_log":     setting.EnableLoginLog,
		"updated_at":           time.Now(),
	}).Error; err != nil {
		return err
	}
	s.invalidateSettingCache()
	return nil
}

// IsOperationLogEnabled 检查操作日志是否开启
func (s *SystemSettingService) IsOperationLogEnabled() bool {
	setting, err := s.GetSetting()
	if err != nil {
		return false // 默认关闭
	}
	return setting.EnableOperationLog == 1
}

// IsLoginLogEnabled 检查登录日志是否开启
func (s *SystemSettingService) IsLoginLogEnabled() bool {
	setting, err := s.GetSetting()
	if err != nil {
		return false // 默认关闭
	}
	return setting.EnableLoginLog == 1
}
