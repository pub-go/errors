package errors

import "fmt"

var _ error = (*withSecondaryError)(nil)
var _ Formatter = (*withSecondaryError)(nil)
var _ fmt.Formatter = (*withSecondaryError)(nil)

type withSecondaryError struct {
	cause          error
	secondaryError error
}

func (e *withSecondaryError) Error() string { return e.cause.Error() }

func (e *withSecondaryError) Cause() error { return e.cause }

func (e *withSecondaryError) Unwrap() error {
	return e.cause
}

func (e *withSecondaryError) Format(s fmt.State, verb rune) {
	FormatError(e, s, verb)
}

func (e *withSecondaryError) FormatError(p Printer) error {
	if p.Detail() {
		p.Printf("secondary error attachment\n%+v", e.secondaryError)
	}
	return e.cause
}
