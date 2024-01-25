package errors

import (
	"errors"
	"fmt"
	"strings"
)

// New 新建一个错误实例，带堆栈
func New(msg string) error {
	return &withStack{
		error: errors.New(msg),
		stack: callers(),
	}
}

// Errorf 按指定格式新建一个错误实例，带堆栈
func Errorf(format string, args ...any) error {
	// 先使用内置 fmt.Errorf 创建错误
	fmtErr := fmt.Errorf(formatPlusW(format), args...)

	var wrapedErr error // 如果当前 Go 版本支持 %w 记录一下
	if we, ok := fmtErr.(interface{ Unwrap() error }); ok {
		wrapedErr = we.Unwrap() // 只包装了一个
	}
	if we, ok := fmtErr.(interface{ Unwrap() []error }); ok {
		errs := we.Unwrap()       // 1.20 起支持包装多个
		wrapedErr = join(errs...) // 多个 join 起来
	}

	var errRefs []error // args 中没被 wrap 的 error
	var hasWrap = wrapedErr != nil
	for _, a := range args {
		if e, ok := a.(error); ok {
			if hasWrap { // wrap 了
				if !Is(wrapedErr, e) { // 但没全部 wrap
					errRefs = append(errRefs, e)
				}
			} else { // 没被 wrap
				errRefs = append(errRefs, e)
			}
		}
	}

	var err error
	errMsg := fmtErr.Error()
	if hasWrap { // 通过 %w 包装了
		causeMsg := wrapedErr.Error()
		if strings.HasSuffix(errMsg, causeMsg) {
			// 如果仅仅是添加了前缀 就使用 withPrefix
			// 这样在 %+v 格式输出时，就不会重复输出 causeMsg
			// prefix: %w
			// prefix: %w\n%w
			prefix := errMsg[:len(errMsg)-len(causeMsg)]
			prefix = strings.TrimSuffix(prefix, ": ")
			err = &withPrefix{
				error:  wrapedErr,
				string: prefix,
			}
		} else {
			// 如果 err 和 cause 不是添加前缀的关系
			// 在输出详细模式时，两者都会输出
			// prefix: %w, %w
			err = &withNewMessage{
				cause:   wrapedErr,      // %w\n%w
				message: fmtErr.Error(), // prefix: %w, %w
			}
		}
	} else { // 没有包装任何错误
		err = errors.New(errMsg)
	}

	if len(errRefs) > 0 { // 没被 wrap 的错误当做次要错误记录一下
		err = WithSecondary(err, join(errRefs...))
	}

	return &withStack{
		error: err,
		stack: callers(),
	}
}

// formatPlusW 1.20 之前不能使用多个 %w 参数 会编译失败
// 所以这里用函数转一下，规避编译失败问题，运行时会打印 %!w(%T=%v)
//
// call has more than one error-wrapping directive %w
func formatPlusW(s string) string { return s }

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
		stack: callers(),
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
		format = strings.ReplaceAll(format, "%w", "%v")
		err = WithMessagef(err, format, args...)
	}

	// 因为只有 fmt.Errorf 才支持传 %w
	// Wrapf 传 %w 会报错，所以直接看 args 里是否有 error
	// 有的话就当做次要错误保存一下
	if len(errRefs) > 0 {
		err = WithSecondary(err, join(errRefs...))
	}
	return &withStack{
		error: err,
		stack: callers(),
	}
}

// WithStack 为错误添加堆栈信息
func WithStack(err error) error {
	if err == nil {
		return nil
	}
	return &withStack{
		error: err,
		stack: callers(),
	}
}

func Join(errs ...error) error {
	return &withStack{
		error: join(errs...),
		stack: callers(),
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
			if n == 1 {
				return err // 只有一个直接返回
			}
			e.errs = append(e.errs, err)
		}
	}
	return e
}
