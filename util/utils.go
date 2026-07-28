package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"go-vue-admin/global"
)

// GenerateUUID 生成UUID
func GenerateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		global.Log.Errorf("生成UUID失败: %v", err)
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			global.Log.Errorf("生成随机字符串失败: %v", err)
			result[i] = charset[0]
			continue
		}
		result[i] = charset[num.Int64()]
	}
	return string(result)
}

// GenerateRandomNumber 生成随机数字
func GenerateRandomNumber(length int) string {
	const charset = "0123456789"
	result := make([]byte, length)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			global.Log.Errorf("生成随机数字失败: %v", err)
			result[i] = charset[0]
			continue
		}
		result[i] = charset[num.Int64()]
	}
	return string(result)
}

// IsEmail 验证邮箱格式
func IsEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, err := regexp.MatchString(pattern, email)
	if err != nil {
		global.Log.Errorf("邮箱验证失败: %v", err)
		return false
	}
	return matched
}

// IsMobile 验证手机号格式（中国大陆）
func IsMobile(mobile string) bool {
	pattern := `^1[3-9]\d{9}$`
	matched, err := regexp.MatchString(pattern, mobile)
	if err != nil {
		global.Log.Errorf("手机号验证失败: %v", err)
		return false
	}
	return matched
}

// IsPhone 验证电话号码格式
func IsPhone(phone string) bool {
	pattern := `^\d{3,4}-?\d{7,8}$`
	matched, err := regexp.MatchString(pattern, phone)
	if err != nil {
		global.Log.Errorf("电话验证失败: %v", err)
		return false
	}
	return matched
}

// Trim 去除字符串首尾空格
func Trim(str string) string {
	return strings.TrimSpace(str)
}

// Contains 判断字符串是否包含子串
func Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ToLower 字符串转小写
func ToLower(str string) string {
	return strings.ToLower(str)
}

// ToUpper 字符串转大写
func ToUpper(str string) string {
	return strings.ToUpper(str)
}

// SubString 截取字符串
func SubString(str string, start, length int) string {
	runes := []rune(str)
	if start < 0 {
		start = 0
	}
	if start >= len(runes) {
		return ""
	}
	end := start + length
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[start:end])
}

// FormatTime 格式化时间
func FormatTime(t time.Time, format string) string {
	return t.Format(format)
}

// ParseTime 解析时间字符串
func ParseTime(timeStr string, format string) (time.Time, error) {
	return time.Parse(format, timeStr)
}

// GetCurrentTime 获取当前时间
func GetCurrentTime() time.Time {
	return time.Now()
}

// GetTimestamp 获取当前时间戳（秒）
func GetTimestamp() int64 {
	return time.Now().Unix()
}

// GetTimestampMs 获取当前时间戳（毫秒）
func GetTimestampMs() int64 {
	return time.Now().UnixNano() / 1e6
}
