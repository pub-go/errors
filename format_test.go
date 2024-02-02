package errors_test

import (
	"fmt"
	"testing"

	"code.gopub.tech/errors"
)

func TestFormat(t *testing.T) {
	err1 := errors.New("err1\nnew\nline")
	err2 := errors.New("err2\nnew\nline")
	err12 := errors.Join(err1, err2)
	err3 := errors.New("err3\nnew\nline")
	err4 := errors.New("err4\nnew\nline")
	err34 := errors.Join(err3, err4)
	err1234 := errors.Join(err12, err34)
	t.Logf("%+v", err1234)
}

func TestDetail(t *testing.T) {
	if s := errors.Detail(nil); s != "<nil>" {
		t.Errorf("Detail(nil) got %s", s)
	}
	t.Logf("%s", errors.Detail(errors.New("error")))
	e0 := fmt.Errorf("fmtErr")
	err := errors.F(e0)
	if e, ok := err.(interface{ Cause() error }); ok {
		if e.Cause() != e0 {
			t.Errorf("Cause() failed")
		}
	}
	if e, ok := err.(interface{ Unwrap() error }); ok {
		if e.Unwrap() != e0 {
			t.Errorf("Unwrap() failed")
		}
	}
}
