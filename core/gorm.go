package core

import (
	"log"
	"os"
	"time"
	"go-vue-admin/global"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func InitGorm() *gorm.DB {
	if global.Config.System.DbType == "mysql" {
		return InitGormMysql()
	}
	return nil
}

func InitGormMysql() *gorm.DB {
	m := global.Config.Mysql

	if m.DbName == "" {
		return nil
	}

	dsn := m.Dsn()

	// 日志级别统一由 log-mode 决定（silent/error/warn/info）
	var gormLogLevel logger.LogLevel
	switch m.LogMode {
	case "info":
		gormLogLevel = logger.Info
	case "warn":
		gormLogLevel = logger.Warn
	case "error":
		gormLogLevel = logger.Error
	default:
		gormLogLevel = logger.Silent
	}

	var logMode logger.Interface
	if m.LogZap {
		// 自定义输出：级别同样尊重 log-mode（不再无条件 Info）；
		// ParameterizedQueries 开启参数脱敏（SQL 以占位符输出，不打印真实参数值）
		logMode = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  gormLogLevel,
				IgnoreRecordNotFoundError: true,
				ParameterizedQueries:      true,
				Colorful:                  true,
			},
		)
	} else {
		logMode = logger.Default.LogMode(gormLogLevel)
	}

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,
		DefaultStringSize:         256,
		DisableDatetimePrecision:  true,
		DontSupportRenameIndex:    true,
		DontSupportRenameColumn:   true,
		SkipInitializeWithVersion: false,
	}), &gorm.Config{
		Logger:         logMode,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})

	if err != nil {
		global.Log.Errorf("连接mysql数据库失败: %v", err)
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		global.Log.Errorf("获取sqlDB失败: %v", err)
		return nil
	}
	
	sqlDB.SetMaxIdleConns(m.MaxIdleConns)
	sqlDB.SetMaxOpenConns(m.MaxOpenConns)
	// 添加连接最大生命周期配置，防止连接泄漏和超时问题
	sqlDB.SetConnMaxLifetime(time.Hour)
	// 添加连接最大空闲时间
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	return db
}
