package genericplan

import (
	"errors"
	"fmt"
)

// Code is one stable kernel rejection category. These codes are internal to the
// experimental planner and are not Fairy C source-diagnostic assignments.
type Code string

const (
	CodeInvalidIdentifier      Code = "G1001"
	CodeInvalidAbilitySet      Code = "G1002"
	CodeDuplicateCatalogItem   Code = "G1003"
	CodeInvalidDeclaration     Code = "G1004"
	CodeCoherenceViolation     Code = "G1005"
	CodeCatalogBudgetExceeded  Code = "G1006"
	CodeUnknownCatalogItem     Code = "G2001"
	CodeTypeArgumentArity      Code = "G2002"
	CodeAbilityUnsatisfied     Code = "G2003"
	CodeImplementationMissing  Code = "G2004"
	CodeSpecializationCycle    Code = "G2005"
	CodePlanningBudgetExceeded Code = "G2006"
)

// Error is a deterministic, machine-classifiable planner failure.
type Error struct {
	Code    Code
	Message string
}

// RequestError retains the root request responsible for a planning failure.
// It carries no source coordinates; source adapters own that mapping. The
// original kernel category remains available through ErrorCode and errors.As.
type RequestError struct {
	Request Request
	Cause   error
}

func (e *RequestError) Error() string { return e.Cause.Error() }
func (e *RequestError) Unwrap() error { return e.Cause }

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func fail(code Code, format string, arguments ...any) error {
	return &Error{Code: code, Message: fmt.Sprintf(format, arguments...)}
}

// ErrorCode extracts a kernel code without requiring callers to match prose.
func ErrorCode(err error) Code {
	var value *Error
	if !errors.As(err, &value) || value == nil {
		return ""
	}
	return value.Code
}
