package res

import "fmt"

// AppError 应用层错误，包含错误码和错误消息
type AppError struct {
	Code    int
	Message string
	Err     error // 内部原始错误，不暴露给客户端
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 支持 errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError 创建应用错误
func NewAppError(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// NewAppErrorWithErr 创建应用错误（携带原始错误）
func NewAppErrorWithErr(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewAppErrorByCode 根据错误码创建应用错误
func NewAppErrorByCode(code int) *AppError {
	return &AppError{
		Code:    code,
		Message: GetErrorMsg(code),
	}
}
