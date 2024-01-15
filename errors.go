package errors

import "fmt"

// New 新建一个错误实例，带堆栈
func New(msg string) error {
	return &withStack{
		error: &leafError{msg: msg},
		stack: callers(1),
	}
}

// Errorf 按指定格式新建一个错误实例，带堆栈
func Errorf(format string, args ...any) error {
	var err error
	var errRefs []error // args 中的 error
	for _, a := range args {
		if e, ok := a.(error); ok {
			errRefs = append(errRefs, e)
		}
	}

	fmtErr := fmt.Errorf(format, args...)
	var wrapedErr error //  %w 对应的参数
	var wappedErrCount int
	if we, ok := fmtErr.(interface{ Unwrap() error }); ok {
		wrapedErr = we.Unwrap() // 只包装了一个
		wappedErrCount = 1
	}
	if we, ok := fmtErr.(interface{ Unwrap() []error }); ok {
		errs := we.Unwrap()
		wrapedErr = join(errs...) // 多个 join 起来
		wappedErrCount = len(errs)
	}

	if wappedErrCount == len(errRefs) { // 通过 %w 包装的
		err = &withNewMessage{
			cause:   wrapedErr,
			message: fmtErr.Error(),
		}
	} else {
		err = &leafError{msg: fmtErr.Error()}
		for _, e := range errRefs {
			err = WithSecondary(err, e)
		}
	}

	return &withStack{
		error: err,
		stack: callers(1),
	}
}

// Cause 获取最内层错误
func Cause(err error) error { return UnwrapAll(err) }

// Unwrap 获取内层错误
func Unwrap(err error) error { return UnwrapOnce(err) }

// WithMessage 给错误添加一个前缀注解信息
func WithMessage(err error, msg string) error {
	if err == nil {
		return nil
	}
	return &withPrefix{error: err, string: msg}
}

// WithMessagef 给错误添加一个指定格式的前缀注解信息
func WithMessagef(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return &withPrefix{
		error:  err,
		string: fmt.Sprintf(format, args...),
	}
}

// WithSecondary 给错误附加一个次要错误。
// 次要错误仅在使用 %+v 格式化动词时才会打印
func WithSecondary(err, secondary error) error {
	if err == nil || secondary == nil {
		return err
	}
	return &withSecondaryError{
		cause:          err,
		secondaryError: secondary,
	}
}

// Wrap 使用指定的前缀信息包装一个错误，带堆栈
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	if msg != "" {
		err = WithMessage(err, msg)
	}
	return &withStack{
		error: err,
		stack: callers(1),
	}
}

// Wrapf 使用指定格式的前缀信息包装一个错误，带堆栈
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	var errRefs []error
	for _, a := range args {
		if e, ok := a.(error); ok {
			errRefs = append(errRefs, e)
		}
	}
	if format != "" || len(args) > 0 {
		err = WithMessagef(err, format, args...)
	}
	for _, e := range errRefs {
		// 因为只有 fmt.Errorf 才支持传 %w
		// Wrapf 传 %w 会报错，所以直接看 args 里是否有 error
		// 有的话就当做次要错误保存一下
		err = WithSecondary(err, e)
	}
	return &withStack{
		error: err,
		stack: callers(1),
	}
}

// WithStack 为错误添加堆栈信息
func WithStack(err error) error {
	if err == nil {
		return nil
	}
	return &withStack{
		error: err,
		stack: callers(1),
	}
}

func Join(errs ...error) error {
	return &withStack{
		error: join(errs...),
		stack: callers(1),
	}
}

func join(errs ...error) error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	e := &joinError{
		errs: make([]error, 0, n),
	}
	for _, err := range errs {
		if err != nil {
			e.errs = append(e.errs, err)
		}
	}
	return e
}
