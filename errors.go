package serverclientcommon

import transportstream "github.com/go-base-lib/transport-stream"

const (
	// ErrCodeUnknown 未知的异常
	ErrCodeUnknown transportstream.ErrCode = iota
	// ErrCodeReadCommand 读取命令异常
	ErrCodeReadCommand
	// ErrCodeCommandUndefined 命令未定义
	ErrCodeCommandUndefined
)

//type ErrCode string
//
//const (
//	ErrCodeNotFount  ErrCode = "404"
//	ErrServerInside  ErrCode = "500"
//	ErrDataParse     ErrCode = "501"
//	ErrValidation    ErrCode = "502"
//	ErrSystemCall    ErrCode = "503"
//	ErrSystemCallEnd ErrCode = "601"
//)
//
//func (e ErrCode) Result(msg string) *Result {
//	return ErrResult(e, msg)
//}
//
//func (e ErrCode) Resultf(msg string, args ...any) *Result {
//	return e.Result(fmt.Sprintf(msg, args...))
//}
//
//func (e ErrCode) ResultWithData(msg string, data any) *Result {
//	return ErrResultWithData((e), msg, data)
//}
//
//func (e ErrCode) ResultWithDataf(data any, msg string, args ...any) *Result {
//	return e.ResultWithData(fmt.Sprintf(msg, args...), data)
//}
