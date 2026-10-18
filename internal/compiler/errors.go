package compiler

import (
	"context"
	"errors"
)

type ErrorKind uint8

const (
	ErrorInput ErrorKind = iota + 1
	ErrorInvariant
	ErrorCanceled
)

// Kind describes handling; Stage retains where the failure happened.
func (e *Error) Kind() ErrorKind {
	if errors.Is(e.Cause, context.Canceled) || errors.Is(e.Cause, context.DeadlineExceeded) {
		return ErrorCanceled
	}
	if e.Stage == StagePackageInput {
		return ErrorInput
	}
	return ErrorInvariant
}
