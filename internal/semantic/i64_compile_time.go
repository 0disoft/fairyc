package semantic

import (
	"math"
	"math/big"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

// Compile-time arithmetic uses an unbounded intermediate so a wrapped host int64 cannot pass.
func (e *compileTimeEvaluator) applyI64Binary(module *moduleScope, expression *ast.Expression, left, right int64) (compileTimeValue, bool) {
	l, r, result := big.NewInt(left), big.NewInt(right), new(big.Int)
	valid := true
	switch expression.Text {
	case "+":
		result.Add(l, r)
	case "-":
		result.Sub(l, r)
	case "*":
		result.Mul(l, r)
	case "/", "%":
		valid = right != 0 && !(left == math.MinInt64 && right == -1)
		if valid && expression.Text == "/" {
			result.Quo(l, r)
		} else if valid {
			result.Rem(l, r)
		}
	case "&":
		result.And(l, r)
	case "|":
		result.Or(l, r)
	case "^":
		result.Xor(l, r)
	case "<<", ">>":
		valid = right >= 0 && right < 64
		if valid && expression.Text == "<<" {
			result.Lsh(l, uint(right))
		} else if valid {
			result.Rsh(l, uint(right))
		}
	default:
		e.rejectExpression(module, expression)
		return compileTimeValue{}, false
	}
	if !valid || !result.IsInt64() {
		e.analyzer.addDiagnostic("E4303", span(module.file.Path, expression.Span))
		return compileTimeValue{}, false
	}
	return compileTimeValue{typ: hir.TypeI64, i: result.Int64()}, true
}
