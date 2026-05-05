package core

import (
	"fmt"
	"os"
	"path"
	"sync"
	"time"
	"go-vue-admin/global"

	"github.com/sirupsen/logrus"
)

// rotateWriter 支持按天轮转的日志写入器
type rotateWriter struct {
	mu         sync.Mutex
	dir        string
	file       *os.File
	currentDay string
	stdout     *os.File
	enableConsole bool
}

// newRotateWriter 创建轮转写入器
func newRotateWriter(dir string, enableConsole bool) (*rotateWriter, error) {
	rw := &rotateWriter{
		dir:           dir,
		stdout:        os.Stdout,
		enableConsole: enableConsole,
	}
	if err := rw.rotate(); err != nil {
		return nil, err
	}
	return rw, nil
}

// rotate 切换日志文件（按天）
func (w *rotateWriter) rotate() error {
	day := time.Now().Format("2006-01-02")
	if w.currentDay == day && w.file != nil {
		return nil
	}

	// 关闭旧文件
	if w.file != nil {
		_ = w.file.Close()
	}

	// 创建日志目录
	if _, err := os.Stat(w.dir); os.IsNotExist(err) {
		if err := os.MkdirAll(w.dir, 0750); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}
	}

	// 打开新日志文件
	logPath := path.Join(w.dir, day+".log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	w.file = file
	w.currentDay = day
	return nil
}

// Write 实现 io.Writer 接口
func (w *rotateWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查是否需要切换文件
	if err := w.rotate(); err != nil {
		// 如果切换失败，尝试写入控制台
		if w.enableConsole && w.stdout != nil {
			return w.stdout.Write(p)
		}
		return 0, err
	}

	// 写入文件
	if w.file != nil {
		w.file.Write(p)
	}

	// 同时写入控制台
	if w.enableConsole && w.stdout != nil {
		return w.stdout.Write(p)
	}

	return len(p), nil
}

// Close 关闭写入器
func (w *rotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func InitLogrus() *logrus.Logger {
	m := global.Config.Zap

	logger := logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(m.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// 设置日志格式
	if m.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			ForceColors:     true,
		})
	}

	// 创建轮转写入器
	writer, err := newRotateWriter(m.Director, m.LogInConsole)
	if err != nil {
		logger.Errorf("创建日志写入器失败: %v", err)
		// 降级到仅控制台输出
		logger.SetOutput(os.Stdout)
	} else {
		logger.SetOutput(writer)
	}

	global.Log = logger
	return logger
}
