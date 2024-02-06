package errors

import "fmt"

var _ error = (*fundamental)(nil)
var _ fmt.Formatter = (*fundamental)(nil)

type fundamental struct {
	string
	*stack
}

func (e *fundamental) Error() string { return e.string }

func (e *fundamental) Format(s fmt.State, verb rune) {
	FormatError(e, s, verb)
}
