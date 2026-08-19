package chainprimitives

import "errors"

var (
	ErrOverflow             = errors.New("chain primitive overflow")
	ErrUnderflow            = errors.New("chain primitive underflow")
	ErrDivisionByZero       = errors.New("chain primitive division by zero")
	ErrInvalidShift         = errors.New("invalid chain primitive shift")
	ErrInvalidEncoding      = errors.New("invalid chain primitive encoding")
	ErrInvalidAddressLength = errors.New("invalid address length")
	ErrInvalidSignature     = errors.New("invalid signature")
	ErrInvalidCostSchedule  = errors.New("invalid intrinsic cost schedule")
	ErrUnknownIntrinsic     = errors.New("unknown chain intrinsic")
	ErrUnsupportedReference = errors.New("reference intrinsic is not implemented")
)
