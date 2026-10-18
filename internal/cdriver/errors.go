package cdriver

import (
	"context"
	"errors"
)

type ErrorKind uint8

const (
	ErrorToolchain ErrorKind = iota + 1
	ErrorBuild
	ErrorExecution
	ErrorCanceled
	ErrorTimeout
)

// Error classifies a driver boundary without making its display text an API.
type Error struct {
	Kind      ErrorKind
	Operation string
	Cause     error
}

func (e *Error) Error() string { return e.Cause.Error() }
func (e *Error) Unwrap() error { return e.Cause }

func driverError(kind ErrorKind, operation string, err error) error {
	if err == nil {
		return nil
	}
	var existing *Error
	if errors.As(err, &existing) {
		return err
	}
	if errors.Is(err, context.Canceled) {
		kind = ErrorCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		kind = ErrorTimeout
	}
	return &Error{Kind: kind, Operation: operation, Cause: err}
}
