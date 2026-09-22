package cbackend

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// Preserve constant-divisor strength reduction without exposing a zero divisor
// or signed MIN/-1 expression to the C compiler. MIR trap checks are unchanged.
func directUnsignedDivisor(operation mir.Operation) bool {
	if len(operation.Operands) != 2 {
		return false
	}
	right := operation.Operands[1]
	if right.U64 == 0 {
		return false
	}
	switch operation.OwnerType {
	case hir.TypeU8:
		return right.Kind == mir.OperandU8
	case hir.TypeU32:
		return right.Kind == mir.OperandU32
	case hir.TypeUSize:
		return right.Kind == mir.OperandUSize
	default:
		return false
	}
}
