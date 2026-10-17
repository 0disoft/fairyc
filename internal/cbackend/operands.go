package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"unicode/utf8"
)

func (e *emissionAnalysis) cFunctionResultType(value hir.Type) (string, error) {
	if value == hir.TypeVoid {
		return "void", nil
	}
	return e.cValueType(value)
}

func cOperand(operand mir.Operand) (string, error) {
	switch operand.Kind {
	case mir.OperandLocal:
		if operand.Local < 0 {
			return "", fmt.Errorf("local operand has invalid id %d", operand.Local)
		}
		return fmt.Sprintf("fy_l_%d", operand.Local), nil
	case mir.OperandI32:
		return cI32(operand.I32), nil
	case mir.OperandI64:
		return cI64(operand.I64), nil
	case mir.OperandU8:
		return fmt.Sprintf("UINT8_C(%d)", operand.U64), nil
	case mir.OperandU32:
		return fmt.Sprintf("UINT32_C(%d)", operand.U64), nil
	case mir.OperandUSize:
		return fmt.Sprintf("((uintptr_t)UINT64_C(%d))", operand.U64), nil
	case mir.OperandNull:
		return "((void *)0)", nil
	case mir.OperandByteString:
		if operand.Type != hir.SharedBorrow(hir.TypeByteSlice) {
			return "", fmt.Errorf("byte-string operand has invalid static storage")
		}
		return fmt.Sprintf("(fy_byte_slice){ .data = %s, .length = ((uintptr_t)UINT64_C(%d)) }", byteStringStorageSymbol(operand.StaticData), len(operand.StaticData)), nil
	case mir.OperandString:
		if operand.Type != hir.SharedBorrow(hir.TypeStr) || !utf8.ValidString(operand.StaticData) {
			return "", fmt.Errorf("string operand has invalid static storage")
		}
		return fmt.Sprintf("(fy_str){ .data = %s, .length = ((uintptr_t)UINT64_C(%d)) }", stringStorageSymbol(operand.StaticData), len(operand.StaticData)), nil
	case mir.OperandCString:
		if operand.Type != hir.SharedBorrow(hir.TypeCStr) || !validCStringStorage(operand.StaticData) {
			return "", fmt.Errorf("C string operand has invalid static storage")
		}
		return cStringStorageSymbol(operand.StaticData), nil
	default:
		return "", fmt.Errorf("unsupported scalar operand kind %q", operand.Kind)
	}
}

func isRawPointer(value hir.Type) bool {
	_, _, pointer := hir.RawPointerElement(value)
	return pointer
}

func cI32(value int32) string {
	if value == -2147483648 {
		return "INT32_MIN"
	}
	if value < 0 {
		return fmt.Sprintf("(-INT32_C(%d))", -int64(value))
	}
	return fmt.Sprintf("INT32_C(%d)", value)
}
