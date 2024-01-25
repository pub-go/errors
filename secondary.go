package errors

import "fmt"

var _ error = (*withSecondaryError)(nil)
var _ ErrorPrinter = (*withSecondaryError)(nil)
var _ fmt.Formatter = (*withSecondaryError)(nil)

type withSecondaryError struct {
	cause          error
	secondaryError error
}

func (e *withSecondaryError) Error() string {
	// 直接输出时，不会输出次要错误
	return e.cause.Error()
}

func (e *withSecondaryError) Cause() error { return e.cause }

func (e *withSecondaryError) Unwrap() error {
	return e.cause
}

func (e *withSecondaryError) Format(s fmt.State, verb rune) {
	FormatError(e, s, verb)
}

func (e *withSecondaryError) PrintError(p Printer) error {
	// 详细输出时，才会输出次要错误
	p.PrintDetailf("secondary error attachment\n%+v", e.secondaryError)
	return e.cause
}
