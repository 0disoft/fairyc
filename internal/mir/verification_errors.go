package mir

import (
	"cmp"
	"fmt"
	"slices"
)

// Verification codes classify invariant failures independently of explanatory Detail.
const (
	CodeControlFlow        = "V1001"
	CodeInitialization     = "V1002"
	CodeAdmission          = "V1003"
	CodeAggregateIdentity  = "V1010"
	CodeAggregateFields    = "V1011"
	CodeDropGlue           = "V1012"
	CodeForeignDeclaration = "V1020"
	CodeFunctionIdentity   = "V1021"
	CodeCallSignature      = "V1022"
	CodeReturnShape        = "V1023"
	CodeOperationKind      = "V1030"
	CodeOperationShape     = "V1031"
	CodeOperationType      = "V1032"
	CodeAggregateUse       = "V1033"
	CodeOperandShape       = "V1034"
	CodeEffectAuthority    = "V1035"
	CodeStorageAdmission   = "V1040"
	CodeStorageShape       = "V1041"
	CodeStorageOperands    = "V1042"
	CodeStorageProvenance  = "V1043"
	CodeStorageReturn      = "V1044"
)

// VerificationError identifies one internal MIR invariant failure.
type VerificationError struct {
	Code   string
	Entity string
	Detail string
}

func (e VerificationError) Error() string {
	return fmt.Sprintf("%s at %s: %s", e.Code, e.Entity, e.Detail)
}

func canonicalVerificationErrors(failures []VerificationError) []VerificationError {
	slices.SortFunc(failures, func(left, right VerificationError) int {
		if value := cmp.Compare(left.Code, right.Code); value != 0 {
			return value
		}
		if value := cmp.Compare(left.Entity, right.Entity); value != 0 {
			return value
		}
		return cmp.Compare(left.Detail, right.Detail)
	})
	return slices.Compact(failures)
}
