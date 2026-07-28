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
			// 注册字段名映射（校验错误提示中字段显示为 json 标签名）
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

// 预编译正则（避免每次校验重复编译）
var (
	phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)
	// Go RE2 不支持 \u 转义，匹配中文用 \p{Han}
	// （原代码用 MatchString+忽略 error 的方式调用 \u4E00，正则编译一直失败，
	//   "不能包含中文"规则此前实际从未生效，此处一并修正）
	chineseRegex = regexp.MustCompile(`\p{Han}`)
	digitRegex   = regexp.MustCompile(`[0-9]`)
	lowerRegex   = regexp.MustCompile(`[a-z]`)
	upperRegex   = regexp.MustCompile(`[A-Z]`)
	symbolRegex  = regexp.MustCompile(`[^0-9a-zA-Z]`)
)

// validatePhone 手机号验证（中国大陆）
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	if phone == "" {
		return true // omitempty 时允许空
	}
	return phoneRegex.MatchString(phone)
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
	if chineseRegex.MatchString(password) {
		return false
	}
	count := 0
	if digitRegex.MatchString(password) {
		count++
	}
	if lowerRegex.MatchString(password) || upperRegex.MatchString(password) {
		count++
	}
	if symbolRegex.MatchString(password) {
		count++
	}
	return count >= 2
}

// validateStatus 状态验证（1启用 2禁用）
func validateStatus(fl validator.FieldLevel) bool {
	status := fl.Field().Int()
	return status == 1 || status == 2
}
