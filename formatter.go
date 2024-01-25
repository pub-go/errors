package errors

import "fmt"

type ErrorPrinter interface {
	error
	// PrintError 输出错误信息.
	// 使用 Print, Printf 输出的错误信息总是可见;
	// 使用 PrintDetail, PrintDetailf 输出错误信息仅在 %+v 格式下可见.
	//
	// 返回的 next error 如果为 nil 代表在拼接简单消息时，仅输出本 error 而忽略 cause.
	//
	// 如一个 withPrefix 结构的错误实例，应在 Print 自身消息后，再返回 cause;
	// 而一个 withNewMessage 结构的错误实例，在 Print 自身消息后，会返回 nil.
	//
	// 在 %+v 详细格式下，不管返回的 next error 是否为 nil，都会输出其 cause.
	PrintError(p Printer) (next error)
}

type Printer interface {
	Print(args ...any)
	Printf(format string, args ...any)
	PrintDetail(args ...any)
	PrintDetailf(format string, args ...any)
}

type FormattableError interface {
	error
	fmt.Formatter
}

var _ FormattableError = (*errorFormatter)(nil)

type errorFormatter struct {
	error
}

func (e *errorFormatter) Format(s fmt.State, verb rune) {
	FormatError(e.error, s, verb) // 直接传包装的 error 不打印自身
}

func (e *errorFormatter) Cause() error {
	return e.error
}

func (e *errorFormatter) Unwrap() error {
	return e.error
}
