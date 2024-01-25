package errors

import "fmt"

var _ error = (*withPrefix)(nil)
var _ ErrorPrinter = (*withPrefix)(nil)
var _ fmt.Formatter = (*withPrefix)(nil)

type withPrefix struct {
	error
	string
}

func (e *withPrefix) Error() string {
	return fmt.Sprintf("%s: %s", e.string, e.error)
}

func (e *withPrefix) Cause() error  { return e.error }
func (e *withPrefix) Unwrap() error { return e.error }

// Format implements fmt.Formatter.
func (e *withPrefix) Format(f fmt.State, verb rune) {
	FormatError(e, f, verb)
}

// PrintError implements Formatter.
func (e *withPrefix) PrintError(p Printer) (next error) {
	p.Print(e.string)
	return e.error
}

var _ error = (*withNewMessage)(nil)
var _ ErrorPrinter = (*withNewMessage)(nil)
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

// PrintError implements Formatter.
func (e *withNewMessage) PrintError(p Printer) (next error) {
	p.Print(e.message)
	return nil
}
