package core

import (
	"regexp"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var initValidatorOnce sync.Once

// InitValidator 注册自定义验证器
func InitValidator() {
	initValidatorOnce.Do(func() {
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			// 注册手机号验证器
			_ = v.RegisterValidation("phone", validatePhone)
			// 注册密码强度验证器
			_ = v.RegisterValidation("password", validatePassword)
			// 注册状态验证器（1启用 2禁用）
			_ = v.RegisterValidation("status", validateStatus)
			// 注册自定义错误消息翻译
			v.RegisterTagNameFunc(func(fld reflect.StructField) string {
				name := strings.SplitN(fld.Tag.Get("json"), ",", 1)[0]
				if name == "-" {
					return ""
				}
				return name
			})
		}
	})
}

// validatePhone 手机号验证（中国大陆）
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	if phone == "" {
		return true // omitempty 时允许空
	}
	matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, phone)
	return matched
}

// validatePassword 密码强度验证（8-18位，必须包含数字、字母、符号中的至少两种）
func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if password == "" {
		return true // omitempty 时允许空
	}
	// 长度 8-18
	if len(password) < 8 || len(password) > 18 {
		return false
	}
	// 不能包含中文
	matched, _ := regexp.MatchString(`[\u4E00-\u9FA5]`, password)
	if matched {
		return false
	}
	// 统计字符类型
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasSymbol := regexp.MustCompile(`[^0-9a-zA-Z]`).MatchString(password)

	count := 0
	if hasDigit {
		count++
	}
	if hasLower || hasUpper {
		count++
	}
	if hasSymbol {
		count++
	}
	return count >= 2
}

// validateStatus 状态验证（1启用 2禁用）
func validateStatus(fl validator.FieldLevel) bool {
	status := fl.Field().Int()
	return status == 1 || status == 2
}
