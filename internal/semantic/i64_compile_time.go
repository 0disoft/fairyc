package semantic

import (
	"math"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

type compileTimeI64Status uint8

const (
	compileTimeI64OK compileTimeI64Status = iota
	compileTimeI64Trap
	compileTimeI64Unsupported
)

// Only this adapter changes diagnostic state; arithmetic itself is pure.
func (e *compileTimeEvaluator) applyI64Binary(module *moduleScope, expression *ast.Expression, left, right int64) (compileTimeValue, bool) {
	result, status := computeCompileTimeI64Binary(expression.Text, left, right)
	switch status {
	case compileTimeI64OK:
		return compileTimeValue{typ: hir.TypeI64, i: result}, true
	case compileTimeI64Unsupported:
		e.rejectExpression(module, expression)
	case compileTimeI64Trap:
		e.analyzer.addDiagnostic("E4303", span(module.file.Path, expression.Span))
	}
	return compileTimeValue{}, false
}

// Overflow checks preserve exact signed results without big.Int allocations.
func computeCompileTimeI64Binary(operator string, left, right int64) (int64, compileTimeI64Status) {
	switch operator {
	case "+":
		overflow := right > 0 && left > math.MaxInt64-right || right < 0 && left < math.MinInt64-right
		return checkedCompileTimeI64(left+right, overflow)
	case "-":
		overflow := right < 0 && left > math.MaxInt64+right || right > 0 && left < math.MinInt64+right
		return checkedCompileTimeI64(left-right, overflow)
	case "*":
		// MinInt64 / -1 wraps in Go too, so the quotient check alone is insufficient.
		if left == math.MinInt64 && right == -1 || right == math.MinInt64 && left == -1 {
			return 0, compileTimeI64Trap
		}
		result := left * right
		return checkedCompileTimeI64(result, right != 0 && result/right != left)
	case "/", "%":
		return computeCompileTimeI64Quotient(operator, left, right)
	case "&":
		return left & right, compileTimeI64OK
	case "|":
		return left | right, compileTimeI64OK
	case "^":
		return left ^ right, compileTimeI64OK
	case "<<", ">>":
		return computeCompileTimeI64Shift(operator, left, right)
	default:
		return 0, compileTimeI64Unsupported
	}
}

func checkedCompileTimeI64(result int64, overflow bool) (int64, compileTimeI64Status) {
	if overflow {
		return 0, compileTimeI64Trap
	}
	return result, compileTimeI64OK
}

func computeCompileTimeI64Quotient(operator string, left, right int64) (int64, compileTimeI64Status) {
	if right == 0 || left == math.MinInt64 && right == -1 {
		return 0, compileTimeI64Trap
	}
	if operator == "/" {
		return left / right, compileTimeI64OK
	}
	return left % right, compileTimeI64OK
}

func computeCompileTimeI64Shift(operator string, left, right int64) (int64, compileTimeI64Status) {
	if right < 0 || right >= int64(bitWidth(hir.TypeI64)) {
		return 0, compileTimeI64Trap
	}
	if operator == ">>" {
		return left >> uint(right), compileTimeI64OK
	}
	if left > math.MaxInt64>>uint(right) || left < math.MinInt64>>uint(right) {
		return 0, compileTimeI64Trap
	}
	return left << uint(right), compileTimeI64OK
}
