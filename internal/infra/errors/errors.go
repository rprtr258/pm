package errors

import "fmt"

func New(msg string, a ...any) error {
	return fmt.Errorf(msg, a...)
}

func Wrap(err error, msg string, a ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(msg, a...), err)
}
