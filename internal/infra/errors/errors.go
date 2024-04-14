package errors

import (
	"fmt"
	"strings"
)

func New(msg string) error {
	return fmt.Errorf("%s", msg)
}

func Newf(msg string, a ...any) error {
	return fmt.Errorf(msg, a...)
}

func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

func Wrapf(err error, msg string, a ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(msg, a...), err)
}

// Combine multiple errs into single one. If no errors are passed or all of them
// are nil, nil is returned.
func Combine(errs ...error) error {
	errList := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			errList = append(errList, err)
		}
	}

	// appendErrs - filter out nil errors
	switch len(errList) {
	case 0:
		return nil
	case 1:
		return errList[0]
	default:
		errors := make([]string, len(errList))
		for i, err := range errList {
			errors[i] = err.Error()
		}
		return New(strings.Join(errors, "; "))
	}
}
