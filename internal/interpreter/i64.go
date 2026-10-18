package interpreter

import (
	"github.com/0disoft/fairyc/internal/hir"
	"math"
)

func i64Value(value int64) runtimeValue {
	return runtimeValue{kind: valueI64, scalar: i64Bits(value), typ: hir.TypeI64}
}

func unaryI64(operator string, value runtimeValue) (runtimeValue, string, error) {
	if !comparisonRuntimeMatches(value, hir.TypeI64) {
		return runtimeValue{}, "", capabilityError("invalid i64 unary operand")
	}
	switch operator {
	case "+":
		return value, "", nil
	case "-":
		if value.i64() == math.MinInt64 {
			return runtimeValue{}, "integer-overflow", nil
		}
		return i64Value(-value.i64()), "", nil
	case "~":
		return i64Value(^value.i64()), "", nil
	default:
		return runtimeValue{}, "", capabilityError("unsupported i64 unary %q", operator)
	}
}

func checkedI64Operation(operator string, left, right runtimeValue) (runtimeValue, string, error) {
	if !comparisonRuntimeMatches(left, hir.TypeI64) || !comparisonRuntimeMatches(right, hir.TypeI64) {
		return runtimeValue{}, "", capabilityError("invalid i64 arithmetic operands")
	}
	l, r := left.i64(), right.i64()
	var result int64
	overflow := false
	switch operator {
	case "+":
		overflow = r > 0 && l > math.MaxInt64-r || r < 0 && l < math.MinInt64-r
		if !overflow {
			result = l + r
		}
	case "-":
		overflow = r < 0 && l > math.MaxInt64+r || r > 0 && l < math.MinInt64+r
		if !overflow {
			result = l - r
		}
	case "*":
		result = l * r
		overflow = l == math.MinInt64 && r == -1 || r == math.MinInt64 && l == -1 || r != 0 && result/r != l
	case "/", "%":
		if r == 0 {
			return runtimeValue{}, "division-by-zero", nil
		}
		if l == math.MinInt64 && r == -1 {
			return runtimeValue{}, "signed-division-overflow", nil
		}
		if operator == "/" {
			result = l / r
		} else {
			result = l % r
		}
	case "<<", ">>":
		if r < 0 || r >= 64 {
			return runtimeValue{}, "invalid-shift", nil
		}
		if operator == ">>" {
			result = l >> uint(r)
		} else {
			overflow = l > math.MaxInt64>>uint(r) || l < math.MinInt64>>uint(r)
			if !overflow {
				result = l << uint(r)
			}
		}
	case "&":
		result = l & r
	case "^":
		result = l ^ r
	case "|":
		result = l | r
	default:
		return runtimeValue{}, "", capabilityError("unsupported i64 binary %q", operator)
	}
	if overflow {
		return runtimeValue{}, "integer-overflow", nil
	}
	return i64Value(result), "", nil
}
