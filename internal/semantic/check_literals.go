package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
	"strconv"
	"strings"
)

func literalType(raw string, expected hir.Type) (hir.Type, bool) {
	if raw == "true" || raw == "false" {
		return hir.TypeBool, true
	}
	if raw == "null" {
		if _, _, pointer := hir.RawPointerElement(expected); pointer {
			return expected, true
		}
		return hir.TypeInvalid, false
	}
	if strings.HasPrefix(raw, "c\"") {
		return hir.SharedBorrow(hir.TypeCStr), true
	}
	if strings.HasPrefix(raw, "b\"") {
		return hir.SharedBorrow(hir.TypeByteSlice), true
	}
	if strings.HasPrefix(raw, "\"") {
		return hir.SharedBorrow(hir.TypeStr), true
	}
	if isIntegerLiteral(raw) {
		switch expected {
		case hir.TypeU8, hir.TypeI32, hir.TypeI64, hir.TypeU32, hir.TypeUSize:
			return expected, true
		}
		return hir.TypeI32, true
	}
	return hir.TypeInvalid, false
}

func isIntegerLiteral(raw string) bool {
	if raw == "" || raw[0] < '0' || raw[0] > '9' {
		return false
	}
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") || strings.HasPrefix(raw, "0b") || strings.HasPrefix(raw, "0B") {
		return true
	}
	return !strings.ContainsAny(raw, ".eE")
}

func integerLiteralFits(raw string, selected hir.Type) bool {
	var maximum uint64
	switch selected {
	case hir.TypeI32:
		maximum = 1<<31 - 1
	case hir.TypeI64:
		maximum = 1<<63 - 1
	case hir.TypeU8:
		maximum = 1<<8 - 1
	case hir.TypeU32:
		maximum = 1<<32 - 1
	case hir.TypeUSize:
		maximum = ^uint64(0)
	default:
		return true
	}
	comparison, valid := compareIntegerLiteralMagnitude(raw, maximum)
	return valid && comparison <= 0
}

func directI32MinimumLiteral(expression *ast.Expression) bool {
	return directSignedMinimumLiteral(expression, 32)
}

func directSignedMinimumLiteral(expression *ast.Expression, width uint) bool {
	if expression == nil || expression.Kind != syntax.KindLiteral || !isIntegerLiteral(expression.Text) {
		return false
	}
	comparison, valid := compareIntegerLiteralMagnitude(expression.Text, uint64(1)<<(width-1))
	return valid && comparison == 0
}

func compareIntegerLiteralMagnitude(raw string, maximum uint64) (int, bool) {
	base, start := 10, 0
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		base, start = 16, 2
	} else if strings.HasPrefix(raw, "0b") || strings.HasPrefix(raw, "0B") {
		base, start = 2, 2
	}
	maximumDigits := strconv.FormatUint(maximum, base)
	significantDigits := 0
	seenNonZero := false
	for index := start; index < len(raw); index++ {
		if raw[index] == '_' {
			continue
		}
		digit, valid := integerDigitValue(raw[index])
		if !valid || digit >= base {
			return 0, false
		}
		if !seenNonZero && digit == 0 {
			continue
		}
		seenNonZero = true
		significantDigits++
	}
	if !seenNonZero {
		return -1, true
	}
	if significantDigits < len(maximumDigits) {
		return -1, true
	}
	if significantDigits > len(maximumDigits) {
		return 1, true
	}
	maximumIndex := 0
	seenNonZero = false
	for index := start; index < len(raw); index++ {
		if raw[index] == '_' {
			continue
		}
		digit, _ := integerDigitValue(raw[index])
		if !seenNonZero && digit == 0 {
			continue
		}
		seenNonZero = true
		maximumDigit, _ := integerDigitValue(maximumDigits[maximumIndex])
		if digit < maximumDigit {
			return -1, true
		}
		if digit > maximumDigit {
			return 1, true
		}
		maximumIndex++
	}
	return 0, true
}

func integerDigitValue(value byte) (int, bool) {
	switch {
	case value >= '0' && value <= '9':
		return int(value - '0'), true
	case value >= 'a' && value <= 'f':
		return int(value-'a') + 10, true
	case value >= 'A' && value <= 'F':
		return int(value-'A') + 10, true
	default:
		return 0, false
	}
}

func literalClass(raw string) string {
	switch {
	case strings.HasPrefix(raw, "\""):
		return "string"
	case strings.HasPrefix(raw, "b\""):
		return "byte-string"
	case strings.HasPrefix(raw, "c\""):
		return "c-string"
	case raw == "null":
		return "null"
	case strings.ContainsAny(raw, ".eE"):
		return "float"
	default:
		return "unknown"
	}
}
