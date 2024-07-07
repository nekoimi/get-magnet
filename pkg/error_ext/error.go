package error_ext

import "net/http"

type ExtError struct {
	Code       int
	HttpStatus int
	Msg        string
}

type CodeError interface {
	error
	GetCode() int
	GetHttpStatus() int
}

func (e *ExtError) GetCode() int {
	return e.Code
}

func (e *ExtError) GetHttpStatus() int {
	return e.HttpStatus
}

func (e *ExtError) Error() string {
	return e.Msg
}

var (
	RequestBodyNotSupportedError = &ExtError{
		Code:       10400,
		HttpStatus: http.StatusBadRequest,
		Msg:        "request body not supported",
	}
	ValidateError = &ExtError{
		Code:       10400,
		HttpStatus: http.StatusBadRequest,
		Msg:        "参数验证错误",
	}
	AccountNotFoundError = &ExtError{
		Code:       20404,
		HttpStatus: http.StatusBadRequest,
		Msg:        "账号或密码错误",
	}
	PasswordError = &ExtError{
		Code:       20400,
		HttpStatus: http.StatusBadRequest,
		Msg:        "账号或密码错误",
	}
	AuthenticationError = &ExtError{
		Code:       20401,
		HttpStatus: http.StatusUnauthorized,
		Msg:        "认证信息异常",
	}
	AuthenticationExpirseError = &ExtError{
		Code:       20401,
		HttpStatus: http.StatusUnauthorized,
		Msg:        "认证信息已过期",
	}
)
