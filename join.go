package errors

// 因为 joinError 总是被其他类型包装，所以可以不必实现 Formatter
var _ error = (*joinError)(nil)

type joinError struct {
	errs []error
}

func (e *joinError) Unwrap() []error { return e.errs }

func (e *joinError) Error() string {
	var b []byte
	for i, err := range e.errs {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, err.Error()...)
	}
	return string(b)
}
