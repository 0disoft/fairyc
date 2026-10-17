package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func unsignedCMaximum(value hir.Type) (string, error) {
	switch value {
	case hir.TypeU8:
		return "UINT8_MAX", nil
	case hir.TypeU32:
		return "UINT32_MAX", nil
	case hir.TypeUSize:
		return "UINTPTR_MAX", nil
	default:
		return "", fmt.Errorf("type %q has no unsigned C maximum", value)
	}
}

func shiftCWidth(value hir.Type) (string, error) {
	switch value {
	case hir.TypeI64:
		return "INT64_C(64)", nil
	case hir.TypeI32:
		return "INT32_C(32)", nil
	case hir.TypeU8:
		return "UINT8_C(8)", nil
	case hir.TypeU32:
		return "UINT32_C(32)", nil
	case hir.TypeUSize:
		return "((uintptr_t)UINT64_C(64))", nil
	default:
		return "", fmt.Errorf("type %q has no checked right-shift width", value)
	}
}

func checkedDivisorName(value hir.Type) (string, error) {
	switch value {
	case hir.TypeI64:
		return "fy_checked_divisor_i64", nil
	case hir.TypeI32:
		return "fy_checked_divisor_i32", nil
	case hir.TypeU8:
		return "fy_checked_divisor_u8", nil
	case hir.TypeU32:
		return "fy_checked_divisor_u32", nil
	case hir.TypeUSize:
		return "fy_checked_divisor_usize", nil
	default:
		return "", fmt.Errorf("type %q has no checked divisor slot", value)
	}
}
