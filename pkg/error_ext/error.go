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
)
