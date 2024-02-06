package errors_test

import (
	"fmt"
	"strings"
	"testing"

	"code.gopub.tech/errors"
)

var (
	errFmt     = fmt.Errorf("fmtErr\nwith\nnew line")
	errLeafNew = errors.New("leafErr\nhas\nnew line")
)

func print(t *testing.T, err error) {
	t.Helper()
	t.Logf("Error(): %s", err.Error())
	t.Logf("verb= v: %v", err)
	t.Logf("verb=+v: %+v", err)
	t.Logf("verb=#v: %#v", err)
	t.Logf("verb= q: %q", err)
	t.Logf("verb= a: %a", err)
}

func TestNew(t *testing.T) {
	print(t, errLeafNew)
}

func TestErrorf(t *testing.T) {
	err := errors.Errorf("arg=%v", 1)
	print(t, err)
	err = errors.Errorf("arg=%v", errLeafNew)
	print(t, err)
	err = errors.Errorf("wrap1: %w", errLeafNew)
	print(t, err)
	err = errors.Errorf("wrap2: %w, %w", errLeafNew, errFmt)
	print(t, err)
	print(t, errors.F(newJoin(errFmt, errLeafNew)))
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
	// errLeafNew: fundamental{string, stack}
	typ := fmt.Sprintf("%T", c)
	if typ != "*errors.fundamental" {
		t.Errorf("cause failed: %v", typ)
	}

	c = errors.Unwrap(err2)
	// Errorf: withStack{withPrefix/withNewMessage, stack}
	typ = fmt.Sprintf("%T", c)
	if typ != "*errors.withPrefix" {
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
	print(t, errors.Wrap(errFmt, "prefix")) // 添加堆栈
	err := errors.New("new")
	print(t, errors.Wrap(err, "prefix")) // 重复堆栈会省略
}

func TestElideStack(t *testing.T) {
	err1 := errors.New("err1")
	t.Logf("%+v", errors.Wrapf(err1, "err1Prefix"))
	err2 := errors.New("err2")
	errJoin := errors.Join(err1, err2)
	print(t, errJoin)
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
	if err := errors.Join(); err != nil {
		t.Errorf("Join() want nil, got: %+v", err)
	}
	if err := errors.Join(nil); err != nil {
		t.Errorf("Join(nil) want nil, got: %+v", err)
	}
	print(t, errors.Join(errFmt, errLeafNew))
}

func TestPretty(t *testing.T) {
	t.Logf("%#v", errors.New("newErr"))
}
