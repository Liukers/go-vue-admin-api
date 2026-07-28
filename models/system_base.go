package models

import (
	"database/sql/driver"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TimeFormat 自定义时间格式
const TimeFormat = "2006-01-02 15:04:05"

// LocalTime 自定义时间类型，用于JSON序列化
type LocalTime time.Time

// MarshalJSON 实现JSON序列化接口
func (t LocalTime) MarshalJSON() ([]byte, error) {
	timeStr := fmt.Sprintf("\"%s\"", time.Time(t).Format(TimeFormat))
	return []byte(timeStr), nil
}

// UnmarshalJSON 实现JSON反序列化接口
func (t *LocalTime) UnmarshalJSON(data []byte) error {
	str := string(data)
	if str == "null" || str == "" {
		*t = LocalTime{}
		return nil
	}
	if len(str) > 1 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	layouts := []string{
		TimeFormat,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, str); err == nil {
			*t = LocalTime(parsed)
			return nil
		}
	}
	return fmt.Errorf("无法解析时间: %s", str)
}

// Value 实现数据库驱动接口
func (t LocalTime) Value() (driver.Value, error) {
	if time.Time(t).IsZero() {
		return nil, nil
	}
	return time.Time(t), nil
}

// Scan 实现数据库扫描接口
func (t *LocalTime) Scan(v interface{}) error {
	if v == nil {
		*t = LocalTime{}
		return nil
	}
	switch value := v.(type) {
	case time.Time:
		*t = LocalTime(value)
	case []byte:
		parsed, err := time.Parse(TimeFormat, string(value))
		if err != nil {
			return err
		}
		*t = LocalTime(parsed)
	case string:
		parsed, err := time.Parse(TimeFormat, value)
		if err != nil {
			return err
		}
		*t = LocalTime(parsed)
	default:
		return fmt.Errorf("无法扫描时间类型: %T", v)
	}
	return nil
}

// String 返回格式化后的字符串
func (t LocalTime) String() string {
	return time.Time(t).Format(TimeFormat)
}

// ToTime 转换为标准time.Time
func (t LocalTime) ToTime() time.Time {
	return time.Time(t)
}

// Model 基础模型结构体（含软删除）
type Model struct {
	ID        uint      `gorm:"primarykey;comment:主键ID" json:"id"`
	CreatedAt LocalTime `gorm:"type:datetime;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt LocalTime `gorm:"type:datetime;not null;comment:更新时间" json:"updatedAt"`
	// 软删除字段，swagger中忽略
	DeletedAt gorm.DeletedAt `gorm:"index;comment:删除时间（软删除）" json:"-" swaggerignore:"true"`
}

// BaseModel 基础模型（不含软删除）
type BaseModel struct {
	ID        uint      `gorm:"primarykey;comment:主键ID" json:"id"`
	CreatedAt LocalTime `gorm:"type:datetime;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt LocalTime `gorm:"type:datetime;not null;comment:更新时间" json:"updatedAt"`
}
