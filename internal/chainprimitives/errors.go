package chainprimitives

import "errors"

var (
	ErrOverflow        = errors.New("chain primitive overflow")
	ErrUnderflow       = errors.New("chain primitive underflow")
	ErrDivisionByZero  = errors.New("chain primitive division by zero")
	ErrInvalidShift    = errors.New("invalid chain primitive shift")
	ErrInvalidEncoding = errors.New("invalid chain primitive encoding")
)
