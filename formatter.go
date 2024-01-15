package errors

import "fmt"

type Formatter interface {
	error
	// FormatError 输出错误信息
	FormatError(p Printer) (next error)
}

type Printer interface {
	Print(args ...any)
	Printf(format string, args ...any)
	Detail() bool
}

var _ FormattableError = (*errorFormatter)(nil)

type FormattableError interface {
	error
	fmt.Formatter
}

type errorFormatter struct {
	err error
}

func (e *errorFormatter) Error() string {
	return e.err.Error()
}

func (e *errorFormatter) Format(s fmt.State, verb rune) {
	FormatError(e.err, s, verb) // 直接传包装的 error 不打印自身
}

func (e *errorFormatter) Cause() error {
	return e.err
}

func (e *errorFormatter) Unwrap() error {
	return e.err
}
