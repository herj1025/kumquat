package e

import "fmt"

// Error 定义业务错误接口
type Error interface {
	error
	Code() int
	Msg() string
}

type customError struct {
	code int
	msg  string
	args []any
}

func (e *customError) Error() string {
	return fmt.Sprintf("code: %d, msg: %s", e.code, e.msg)
}

func (e *customError) Code() int {
	return e.code
}

func (e *customError) Msg() string {
	return e.msg
}

func (e *customError) Args() []any {
	return e.args
}

// New 创建一个新的业务错误
func New(code int, msg string) Error {
	return &customError{
		code: code,
		msg:  msg,
	}
}

func Newf(code int, msg string, args ...any) Error {
	return &customError{
		code: code,
		msg:  msg,
		args: args,
	}
}
