# errors
errors with stack, mini version of [cockroachdb/errors](github.com/cockroachdb/errors)


```go
import (
	"code.gopub.tech/errors"
)

err := errors.New("error message")

// use %v, %s, %q, %x, %X: print simple error message
fmt.Printf("%v\n", err)
fmt.Printf("%q\n", err)
// use %+v: print detailed error message(with stack)
fmt.Printf("%+v\n", err)

err := errors.Errorf("prefix: %s", "error message")
// support %w to wrap an error
// support multi %w since go 1.20
err := errors.Errorf("prefix: %w", err)

err := errors.WithMessage(err, "prefix")
err := errors.WithMessagef(err, "prefix: arg=%d", 1)

err := errors.WithSecondary(err, otherError)

err := errors.Wrap(err, "prefix")
err := errors.Wrapf(err, "prefix: arg=%v", 1)

err := errors.WithStack(err)

err := errors.Join(err, err2, err3)
```

err1: = errors.New("error cause 1") // Line a
err2: = errors.New("error cause 2") // Line b
errJoin := errors.Join(err1, err2)  // Line c
errPrefix := errors.WithMessage(errJoin, "prefix may also contains\nnew lines")
err := errors.WithStack(errPrefix)  // Line d

entry {
  head
  detail
  outer *entry
  wraps []*entry
}

prefix may also contains
new lines: error cause 1
error cause 2
(1) attached stack trace
│  -- stack trace: depth=0 【`│  `】
│  code.gopub.tech/errors_test.go:20: test 【Line d】
│  	/path/to/$GOPATH/src/code.gopub.tech/errors_test.go:21: test
│  [...repeated from below...]
├─ Wraps: (2) prefix may also contains 【`├─ `】
│  new lines
├─ Wraps: (3) attached stack trace【`├─ `】
│  -- stack trace:
│  code.gopub.tech/errors_test.go:20: test 【Line c】
└─ Wraps: (4) error cause 1 【`└─ `】 multi-cause
    │  error cause 2 【`    │  `】
    ├─ Wraps: (5) attached stack trace 【`    ├─ `】
    │   │  -- stack trace: 【`    │   │  `】
    │   │  code.gopub.tech/errors_test.go:20: test 【Line a】
    │   │  	/path/to/$GOPATH/src/code.gopub.tech/errors_test.go:21: test
    │   │  code.gopub.tech/errors_test.go:20: test
    │   │  	/path/to/$GOPATH/src/code.gopub.tech/errors_test.go:21: test
    │   │  code.gopub.tech/errors_test.go:20: test
    │   │  	/path/to/$GOPATH/src/code.gopub.tech/errors_test.go:21: test
    │   └─ Wraps: (6) error cause 1 【`    │   └─ `】 no-cause
    └─ Wraps: (7) attached stack trace 【`    └─ `】 no-cause
        │  -- stack trace: 【`        │  `】
        │  code.gopub.tech/errors_test.go:20: test 【Line b】
        │  	/path/to/$GOPATH/src/code.gopub.tech/errors_test.go:21: test
        │  code.gopub.tech/errors_test.go:20: test
        │  	/path/to/$GOPATH/src/code.gopub.tech/errors_test.go:21: test
        │  code.gopub.tech/errors_test.go:20: test
        │  	/path/to/$GOPATH/src/code.gopub.tech/errors_test.go:21: test
        └─ Wraps: (8) error cause 2 【`        └─ `】 no-cause
Error Types: (1) *errors.withStack (2) *errors.withPrefix (3) *errors.withStack (4) *errors.joinError (5) *errors.withStack (6) *errors.leafError (7) *errors.withStack (8) *errors.leafError
