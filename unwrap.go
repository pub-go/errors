package errors

import "errors"

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}

func UnwrapOnce(err error) error {
	switch e := err.(type) {
	case interface{ Cause() error }:
		return e.Cause()
	case interface{ Unwrap() error }:
		return e.Unwrap()
	}
	return nil
}

func UnwrapAll(err error) error {
	for {
		if cause := UnwrapOnce(err); cause != nil {
			err = cause
			continue
		}
		break
	}
	return err
}

func UnwrapMulti(err error) []error {
	if me, ok := err.(interface{ Unwrap() []error }); ok {
		return me.Unwrap()
	}
	return nil
}
