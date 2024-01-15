package errors

import "fmt"

var _ error = (*joinError)(nil)
var _ Formatter = (*joinError)(nil)
var _ fmt.Formatter = (*joinError)(nil)

type joinError struct {
	errs []error
}

func (e *joinError) Unwrap() []error { return e.errs }

func (e *joinError) Error() string {
	return fmt.Sprint(e)
}

func (e *joinError) Format(s fmt.State, verb rune) {
	FormatError(e, s, verb)
}

func (e *joinError) FormatError(p Printer) error {
	for i, err := range e.errs {
		if i > 0 {
			p.Print("\n")
		}
		p.Print(err)
	}
	return nil
}
