package errors

import "fmt"

var _ error = (*withPrefix)(nil)
var _ Formatter = (*withPrefix)(nil)
var _ fmt.Formatter = (*withPrefix)(nil)

type withPrefix struct {
	error
	string
}

func (e *withPrefix) Cause() error  { return e.error }
func (e *withPrefix) Unwrap() error { return e.error }

// Format implements fmt.Formatter.
func (e *withPrefix) Format(f fmt.State, verb rune) {
	FormatError(e, f, verb)
}

// FormatError implements Formatter.
func (e *withPrefix) FormatError(p Printer) (next error) {
	p.Print(e.string)
	return e.error
}

var _ error = (*withNewMessage)(nil)
var _ Formatter = (*withNewMessage)(nil)
var _ fmt.Formatter = (*withNewMessage)(nil)

type withNewMessage struct {
	message string
	cause   error
}

func (e *withNewMessage) Error() string { return e.message }
func (e *withNewMessage) Cause() error  { return e.cause }
func (e *withNewMessage) Unwrap() error { return e.cause }

// Format implements fmt.Formatter.
func (e *withNewMessage) Format(f fmt.State, verb rune) {
	FormatError(e, f, verb)
}

// FormatError implements Formatter.
func (e *withNewMessage) FormatError(p Printer) (next error) {
	p.Print(e.message)
	return nil
}
