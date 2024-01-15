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
