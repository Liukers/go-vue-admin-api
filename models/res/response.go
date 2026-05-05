package res

import (
	"net/http"
	"go-vue-admin/global"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构体
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: SuccessMsg,
		Data:    data,
	})
}

// SuccessWithMessage 成功响应（自定义消息）
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: message,
		Data:    data,
	})
}

// Fail 失败响应
func Fail(c *gin.Context, code int) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: GetErrorMsg(code),
		Data:    nil,
	})
}

// FailWithMessage 失败响应（自定义消息）
func FailWithMessage(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// FailWithData 失败响应（带数据）
func FailWithData(c *gin.Context, code int, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: GetErrorMsg(code),
		Data:    data,
	})
}

// Error 错误响应（内部错误，不暴露详细信息给客户端）
func Error(c *gin.Context, err error) {
	global.Log.Error("请求处理失败: ", err)
	// 检查是否为 AppError
	if appErr, ok := err.(*AppError); ok {
		c.JSON(http.StatusOK, Response{
			Code:    appErr.Code,
			Message: appErr.Message,
			Data:    nil,
		})
		return
	}
	// 普通错误不暴露内部信息
	c.JSON(http.StatusOK, Response{
		Code:    ErrorCodeInternalServer,
		Message: ErrorMsgInternalServer,
		Data:    nil,
	})
}

// ErrorWithMessage 错误响应（自定义消息，用于兼容旧代码）
func ErrorWithMessage(c *gin.Context, err error) {
	global.Log.Error("请求处理失败: ", err)
	// 检查是否为 AppError
	if appErr, ok := err.(*AppError); ok {
		c.JSON(http.StatusOK, Response{
			Code:    appErr.Code,
			Message: appErr.Message,
			Data:    nil,
		})
		return
	}
	// 普通错误不暴露内部信息
	c.JSON(http.StatusOK, Response{
		Code:    ErrorCodeInternalServer,
		Message: ErrorMsgInternalServer,
		Data:    nil,
	})
}

// PageResult 分页结果结构体
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// PageSuccess 分页成功响应
func PageSuccess(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: SuccessMsg,
		Data: PageResult{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// Unauthorized 未授权响应
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    ErrorCodeUnauthorized,
		Message: message,
		Data:    nil,
	})
}

// Forbidden 禁止访问响应
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    ErrorCodeForbidden,
		Message: message,
		Data:    nil,
	})
}

// ValidationError 参数验证错误响应
func ValidationError(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    ErrorCodeParamInvalid,
		Message: message,
		Data:    nil,
	})
}
