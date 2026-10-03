package cbackend

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// reachableShiftCount keeps statically invalid C shifts out of unreachable
// value/overflow expressions. The preceding MIR checked-shift terminator still
// tests the original count and traps, so a dummy zero cannot accept bad input.
// Nonconstant and valid counts retain the existing emission.
func reachableShiftCount(typ hir.Type, count mir.Operand, text string) string {
	width := uint64(32)
	switch typ {
	case hir.TypeU8:
		width = 8
	case hir.TypeI64, hir.TypeUSize:
		width = 64
	}
	switch count.Kind {
	case mir.OperandI32:
		if count.I32 < 0 || uint64(count.I32) >= width {
			return "0"
		}
	case mir.OperandI64:
		if count.I64 < 0 || uint64(count.I64) >= width {
			return "0"
		}
	case mir.OperandU8, mir.OperandU32, mir.OperandUSize:
		if count.U64 >= width {
			return "0"
		}
	}
	return text
}
