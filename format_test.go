package errors_test

import (
	goErr "errors"
	"fmt"
	"strings"
	"testing"

	"code.gopub.tech/errors"
)

type wrapMini struct {
	msg   string
	cause error
}

func (e *wrapMini) Error() string { return e.msg }
func (e *wrapMini) Unwrap() error { return e.cause }

type wrapElideCauses struct {
	override string
	causes   []error
}

func NewWrapElideCauses(override string, errors ...error) error {
	return &wrapElideCauses{
		override: override,
		causes:   errors,
	}
}

func (e *wrapElideCauses) Error() string {
	var b strings.Builder
	b.WriteString(e.override)
	b.WriteString(": ")
	for i, ee := range e.causes {
		b.WriteString(ee.Error())
		if i < len(e.causes)-1 {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

func (e *wrapElideCauses) FormatError(p errors.Printer) (next error) {
	p.Print(e.override)
	return nil
}

func (e *wrapElideCauses) Unwrap() []error {
	return e.causes
}

type wrapNoElideCauses struct {
	prefix string
	causes []error
}

func NewWrapNoElideCauses(prefix string, errors ...error) error {
	return &wrapNoElideCauses{
		prefix: prefix,
		causes: errors,
	}
}

func (e *wrapNoElideCauses) Error() string {
	var b strings.Builder
	b.WriteString(e.prefix)
	b.WriteString(": ")
	for i, ee := range e.causes {
		b.WriteString(ee.Error())
		if i < len(e.causes)-1 {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

func (e *wrapNoElideCauses) Unwrap() []error { return e.causes }

func (e *wrapNoElideCauses) FormatError(p errors.Printer) (next error) {
	p.Print(e.prefix)
	return e.causes[0]
}

type stdJoinError struct {
	errs []error
}

func (e *stdJoinError) Error() string {
	var b []byte
	for i, err := range e.errs {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, err.Error()...)
	}
	return string(b)
}

func (e *stdJoinError) Unwrap() []error {
	return e.errs
}

func goErrJoin(errs ...error) error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	e := &stdJoinError{
		errs: make([]error, 0, n),
	}
	for _, err := range errs {
		if err != nil {
			e.errs = append(e.errs, err)
		}
	}
	return e
}

func TestFormattable(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		simple string
		detail string
	}{
		{
			name:   "single wrapper",
			err:    fmt.Errorf("%w", fmt.Errorf("a%w", goErr.New("b"))),
			simple: "ab",
			detail: `ab
(1)
Wraps: (2) ab
Wraps: (3) b
Error types: (1) *fmt.wrapError (2) *fmt.wrapError (3) *errors.errorString`,
		},
		{
			name:   "simple multi-wrapper",
			err:    goErrJoin(goErr.New("a"), goErr.New("b")),
			simple: "a\nb",
			detail: `a
(1) ab
Wraps: (2) b
Wraps: (3) a
Error types: (1) *errors_test.stdJoinError (2) *errors.errorString (3) *errors.errorString`,
		},
		{
			name: "multi-wrapper with custom formatter and partial elide",
			err: NewWrapNoElideCauses("A",
				NewWrapNoElideCauses("C", goErr.New("3"), goErr.New("4")),
				NewWrapElideCauses("B", goErr.New("1"), goErr.New("2")),
			),
			simple: "A: B: C: 4: 3", // 1 和 2 被省略
			detail: `A: B: C: 4: 3
(1) A
Wraps: (2) B
└─ Wraps: (3) 2
└─ Wraps: (4) 1
Wraps: (5) C
└─ Wraps: (6) 4
└─ Wraps: (7) 3
Error types: (1) *errors_test.wrapNoElideCauses (2) *errors_test.wrapElideCauses (3) *errors.errorString (4) *errors.errorString (5) *errors_test.wrapNoElideCauses (6) *errors.errorString (7) *errors.errorString`,
		},
		{
			name: "multi-wrapper with custom formatter and no elide",
			err: NewWrapNoElideCauses("A",
				NewWrapNoElideCauses("B", goErr.New("1"), goErr.New("2")),
				NewWrapNoElideCauses("C", goErr.New("3"), goErr.New("4")),
			),
			simple: "A: C: 4: 3: B: 2: 1",
			detail: `A: C: 4: 3: B: 2: 1
(1) A
Wraps: (2) C
└─ Wraps: (3) 4
└─ Wraps: (4) 3
Wraps: (5) B
└─ Wraps: (6) 2
└─ Wraps: (7) 1
Error types: (1) *errors_test.wrapNoElideCauses (2) *errors_test.wrapNoElideCauses (3) *errors.errorString (4) *errors.errorString (5) *errors_test.wrapNoElideCauses (6) *errors.errorString (7) *errors.errorString`,
		},
		{
			name:   "simple multi-line error",
			err:    goErr.New("a\nb\nc\nd"),
			simple: "a\nb\nc\nd",
			// TODO 3 个换行符号都应该处理
			detail: `a
(1) ab
  |
  | c
  | d
Error types: (1) *errors.errorString`,
		},
		{
			name: "two-level multi-wrapper",
			err: goErrJoin(
				goErrJoin(goErr.New("a"), goErr.New("b")),
				goErrJoin(goErr.New("c"), goErr.New("d")),
			),
			simple: "a\nb\nc\nd",
			// TODO 换行符号没处理好: (1)(2)(5)
			detail: `a
(1) ab
  |
  | c
  | d
Wraps: (2) cd
└─ Wraps: (3) d
└─ Wraps: (4) c
Wraps: (5) ab
└─ Wraps: (6) b
└─ Wraps: (7) a
Error types: (1) *errors_test.stdJoinError (2) *errors_test.stdJoinError (3) *errors.errorString (4) *errors.errorString (5) *errors_test.stdJoinError (6) *errors.errorString (7) *errors.errorString`,
		},
		{
			name: "simple multi-wrapper with single-cause chains inside",
			err: goErrJoin(
				fmt.Errorf("a%w", goErr.New("b")),
				fmt.Errorf("c%w", goErr.New("d")),
			),
			simple: "ab\ncd",
			detail: `ab
(1) ab
  | cd
Wraps: (2) cd
└─ Wraps: (3) d
Wraps: (4) ab
└─ Wraps: (5) b
Error types: (1) *errors_test.stdJoinError (2) *fmt.wrapError (3) *errors.errorString (4) *fmt.wrapError (5) *errors.errorString`,
		},
		{
			name: "multi-cause wrapper with single-cause cains inside",
			err: goErrJoin(
				fmt.Errorf("a%w", fmt.Errorf("b%w", fmt.Errorf("c%w", goErr.New("d")))),
				fmt.Errorf("e%w", fmt.Errorf("f%w", fmt.Errorf("g%w", goErr.New("h")))),
			),
			simple: "abcd\nefgh",
			detail: `abcd
(1) abcd
  | efgh
Wraps: (2) efgh
└─ Wraps: (3) fgh
  └─ Wraps: (4) gh
    └─ Wraps: (5) h
Wraps: (6) abcd
└─ Wraps: (7) bcd
  └─ Wraps: (8) cd
    └─ Wraps: (9) d
Error types: (1) *errors_test.stdJoinError (2) *fmt.wrapError (3) *fmt.wrapError (4) *fmt.wrapError (5) *errors.errorString (6) *fmt.wrapError (7) *fmt.wrapError (8) *fmt.wrapError (9) *errors.errorString`,
		},
		{
			name: "single cause chain with multi-cause wrapper inside with single-cause chains inside",
			err: fmt.Errorf(
				"prefix1: %w",
				fmt.Errorf("prefix2: %w", goErrJoin(
					fmt.Errorf("a%w", fmt.Errorf("b%w", fmt.Errorf("c%w", goErr.New("d")))),
					fmt.Errorf("e%w", fmt.Errorf("f%w", fmt.Errorf("g%w", goErr.New("h")))),
				))),
			simple: "prefix1: prefix2: abcd\nefgh",
			detail: `prefix1: prefix2: abcd
(1) prefix1
Wraps: (2) prefix2
Wraps: (3) abcd
  | efgh
  └─ Wraps: (4) efgh
    └─ Wraps: (5) fgh
      └─ Wraps: (6) gh
        └─ Wraps: (7) h
  └─ Wraps: (8) abcd
    └─ Wraps: (9) bcd
      └─ Wraps: (10) cd
        └─ Wraps: (11) d
Error types: (1) *fmt.wrapError (2) *fmt.wrapError (3) *errors_test.stdJoinError (4) *fmt.wrapError (5) *fmt.wrapError (6) *fmt.wrapError (7) *errors.errorString (8) *fmt.wrapError (9) *fmt.wrapError (10) *fmt.wrapError (11) *errors.errorString`,
		},
		{
			name:   "wrapMini",
			err:    &wrapMini{msg: "prefix: inner", cause: goErr.New("inner")},
			simple: "prefix: inner",
			detail: `prefix: inner
(1) prefix
Wraps: (2) inner
Error types: (1) *errors_test.wrapMini (2) *errors.errorString`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fe := errors.Formattable(tt.err)
			s := fmt.Sprintf("%s", fe)
			if s != tt.simple {
				t.Errorf("\nexpected: \n%s\nbut got: \n%s\n", tt.simple, s)
			}
			s = fmt.Sprintf("%+v", fe)
			if s != tt.detail {
				t.Errorf("\nexpected: \n%s\nbut got: \n%s\n", tt.detail, s)
			}
		})
	}
}
