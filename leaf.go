package errors

import "fmt"

var _ error = (*leafError)(nil)
var _ Formatter = (*leafError)(nil)
var _ fmt.Formatter = (*leafError)(nil)

func NewLeafError(msg string) error {
	return &leafError{msg: msg}
}

type leafError struct {
	msg string
}

func (e *leafError) Error() string {
	return e.msg
}

func (e *leafError) Format(s fmt.State, verb rune) {
	FormatError(e, s, verb)
}

func (e *leafError) FormatError(p Printer) (next error) {
	p.Print(e.msg)
	return nil
}
