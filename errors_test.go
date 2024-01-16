package errors_test

import (
	"fmt"
	"strings"
	"testing"

	"code.gopub.tech/errors"
)

var (
	errFmt     = fmt.Errorf("fmtErr")
	errLeafNew = errors.New("leafErr")
)

func print(t *testing.T, err error) {
	t.Helper()
	t.Logf("verb= v: %v", err)
	t.Logf("verb=+v: %+v", err)
	t.Logf("verb= q: %q", err)

}

func TestNew(t *testing.T) {
	print(t, errLeafNew)
}

func TestErrorf(t *testing.T) {
	err := errors.Errorf("arg=%v", errLeafNew)
	print(t, err)
	err = errors.Errorf("wrap1: %w", errLeafNew)
	print(t, err)
	err = errors.Errorf("wrap2: %w, %w", errLeafNew, errFmt)
	print(t, err)
	print(t, errors.Formattable(newJoin(errFmt, errLeafNew)))
}

type onlyJoin struct {
	errs []error
}

func newJoin(errs ...error) error {
	return &onlyJoin{errs: errs}
}
func (e *onlyJoin) Error() string {
	var sb strings.Builder
	for i, err := range e.errs {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(err.Error())
	}
	return sb.String()
}
func (e *onlyJoin) Unwrap() []error {
	return e.errs
}

func TestCause(t *testing.T) {
	err := errors.Errorf("wrap: %w", errLeafNew)
	err2 := errors.Errorf("wrap: %w", err)

	c := errors.Cause(err2)
	// errLeafNew: withStack{leafError, stack}
	typ := fmt.Sprintf("%T", c)
	if typ != "*errors.leafError" {
		t.Errorf("cause failed: %v", typ)
	}

	c = errors.Unwrap(err2)
	// Errorf: withStack{fmt.Errorf, stack}
	// fmt.Errorf: fmt.wrapError
	typ = fmt.Sprintf("%T", c)
	if typ != "*errors.withNewMessage" {
		t.Errorf("Unwrap failed: %v", typ)
	}
	c = errors.Unwrap(c)
	if c != err {
		t.Error("unwrap fail")
	}
	typ = fmt.Sprintf("%T", c)
	if typ != "*errors.withStack" {
		t.Errorf("Unwrap failed: %v", typ)
	}
}

func TestWithMessage(t *testing.T) {
	print(t, errors.WithMessage(errFmt, "prefix"))
	print(t, errors.WithMessage(errLeafNew, "prefix"))
	print(t, errors.WithMessagef(errLeafNew, "a=%d", 1))
	print(t, errors.WithMessage(fmt.Errorf(s("a: %w, b: %w"), errFmt, errFmt), "prefix"))
}

func s(s string) string { return s }

func TestWithSecondary(t *testing.T) {
	print(t, errors.WithSecondary(errors.Errorf("mainErr"), errFmt))
}

func TestWrap(t *testing.T) {
	print(t, errors.Wrap(errFmt, "prefix"))     // 添加堆栈
	print(t, errors.Wrap(errLeafNew, "prefix")) // 重复堆栈会省略
}

func TestWrapf(t *testing.T) {
	print(t, errors.Wrapf(errFmt, "prefix: %v", errLeafNew))
	print(t, errors.Wrapf(errFmt, s("leaf: %w"), errLeafNew))
	print(t, errors.Wrapf(errFmt, "a=%v, b=%v", errLeafNew, errFmt))
}

func TestWithStack(t *testing.T) {
	print(t, errors.WithStack(errFmt))
}

func TestJoin(t *testing.T) {
	print(t, errors.Join(errFmt, errLeafNew))
}

func TestPretty(t *testing.T) {
	t.Logf("%#v", errors.New("newErr"))
}
