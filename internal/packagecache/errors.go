package packagecache

import "errors"

type ErrorKind uint8

const (
	ErrorConfiguration ErrorKind = iota + 1
	ErrorKey
	ErrorRecord
	ErrorLimit
	ErrorIO
)

// Error retains the underlying filesystem or sentinel error. Cache consumers
// still decide whether an unavailable/rejected record is a disposable miss.
type Error struct {
	Kind  ErrorKind
	Cause error
}

func (e *Error) Error() string { return e.Cause.Error() }
func (e *Error) Unwrap() error { return e.Cause }

func cacheError(kind ErrorKind, err error) error {
	if err == nil {
		return nil
	}
	var existing *Error
	if errors.As(err, &existing) {
		return err
	}
	if errors.Is(err, ErrRecordTooLarge) {
		kind = ErrorLimit
	}
	return &Error{Kind: kind, Cause: err}
}
