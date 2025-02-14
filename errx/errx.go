package errx

import "fmt"

/**
常用通用固定错误定义
*/

// CodeError 定义了一个包含错误码和错误信息的结构体，用于统一管理错误
type CodeError struct {
	errCode uint32 `json:"code"` // 错误码，用于前端识别错误类型
	errMsg  string `json:"msg"`  // 错误信息，描述具体错误
}

// GetErrCode 返回 CodeError 中的错误码，供前端使用
func (e *CodeError) GetErrCode() uint32 {
	return e.errCode
}

// GetErrMsg 返回 CodeError 中的错误信息，供前端显示
func (e *CodeError) GetErrMsg() string {
	return e.errMsg
}

// Error 实现了 error 接口，返回格式化后的错误描述信息
func (e *CodeError) Error() string {
	return fmt.Sprintf("ErrCode:%d,ErrMsg:%s", e.errCode, e.errMsg)
}

// NewErrCodeMsg 根据传入的错误码和错误信息创建一个新的 CodeError 对象
func NewErrCodeMsg(errCode uint32, errMsg string) *CodeError {
	return &CodeError{errCode: errCode, errMsg: errMsg}
}

// NewErrCode 根据错误码创建 CodeError 对象，错误信息通过 MapErrMsg(errCode) 得到
func NewErrCode(errCode uint32) *CodeError {
	return &CodeError{errCode: errCode, errMsg: MapErrMsg(errCode)}
}

// NewDefaultError 根据传入的错误信息创建一个默认的 CodeError 对象，错误码为 ServerCommonError
func NewDefaultError(errMsg string) *CodeError {
	return &CodeError{errCode: ServerCommonError, errMsg: errMsg}
}

// NewErrorf 根据格式化字符串和参数创建一个 CodeError 对象，便于构造详细错误信息
func NewErrorf(errCode uint32, format string, args ...interface{}) *CodeError {
	return &CodeError{errCode: errCode, errMsg: fmt.Sprintf(format, args...)}
}
