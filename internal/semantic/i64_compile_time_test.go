package semantic

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
)

// Arbitrary precision is the independent oracle for checked fixed-width results.
func exactCompileTimeI64(operator string, left, right int64) (int64, string) {
	l, r, result := big.NewInt(left), big.NewInt(right), new(big.Int)
	switch operator {
	case "+":
		result.Add(l, r)
	case "-":
		result.Sub(l, r)
	case "*":
		result.Mul(l, r)
	case "/", "%":
		if right == 0 || left == math.MinInt64 && right == -1 {
			return 0, "E4303"
		}
		if operator == "/" {
			result.Quo(l, r)
		} else {
			result.Rem(l, r)
		}
	case "&":
		result.And(l, r)
	case "|":
		result.Or(l, r)
	case "^":
		result.Xor(l, r)
	case "<<", ">>":
		if right < 0 || right >= 64 {
			return 0, "E4303"
		}
		if operator == "<<" {
			result.Lsh(l, uint(right))
		} else {
			result.Rsh(l, uint(right))
		}
	default:
		return 0, "E4302"
	}
	if !result.IsInt64() {
		return 0, "E4303"
	}
	return result.Int64(), ""
}

func TestCompileTimeI64BinaryMatchesExactArithmetic(t *testing.T) {
	values := []int64{
		math.MinInt64, math.MinInt64 + 1, math.MinInt64/2 - 1, math.MinInt64 / 2,
		-3037000500, -3037000499, -65, -64, -1, 0, 1, 62, 63, 64, 65,
		3037000499, 3037000500, math.MaxInt64 / 2, math.MaxInt64 - 1, math.MaxInt64,
	}
	operators := []string{"+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>", "unsupported"}
	evaluator := compileTimeEvaluator{analyzer: &analyzer{}}
	module := &moduleScope{file: File{Path: "i64.fy"}}
	expression := &ast.Expression{Span: diagnostic.Span{StartByte: 7, EndByte: 9}}
	check := func(left, right int64) {
		t.Helper()
		for _, operator := range operators {
			expression.Text = operator
			evaluator.analyzer.diagnostics = nil
			evaluator.analyzer.diagnosticBudget = newDiagnosticBudget(1)
			want, code := exactCompileTimeI64(operator, left, right)
			got, ok := evaluator.applyI64Binary(module, expression, left, right)
			if ok != (code == "") || ok && got != (compileTimeValue{typ: hir.TypeI64, i: want}) || !ok && got != (compileTimeValue{}) {
				t.Fatalf("%d %s %d = %+v, %t; want %d, %q", left, operator, right, got, ok, want, code)
			}
			diagnostics := evaluator.analyzer.diagnostics
			if code == "" {
				if len(diagnostics) != 0 {
					t.Fatalf("successful operation emitted diagnostics: %+v", diagnostics)
				}
			} else if len(diagnostics) != 1 || diagnostics[0].Code != code || diagnostics[0].Primary != (hir.Span{Source: "i64.fy", StartByte: 7, EndByte: 9}) {
				t.Fatalf("%d %s %d diagnostics = %+v, want %s at original span", left, operator, right, diagnostics, code)
			}
			if expression.Text != operator || expression.Span != (diagnostic.Span{StartByte: 7, EndByte: 9}) || module.file.Path != "i64.fy" {
				t.Fatal("evaluation changed the input expression or module")
			}
		}
	}
	for _, left := range values {
		for _, right := range values {
			check(left, right)
		}
	}
	random := rand.New(rand.NewPCG(77, 17))
	for range 4096 {
		left, right := int64(random.Uint64()), int64(random.Uint64())
		check(left, right)
		check(left, int64(random.IntN(67))-1) // Exercise valid and adjacent invalid shifts.
	}
}

func TestCompileTimeI64ArithmeticDoesNotAllocate(t *testing.T) {
	allocations := testing.AllocsPerRun(10, func() {
		for _, operator := range []string{"+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>"} {
			if _, status := computeCompileTimeI64Binary(operator, 77, 1); status != compileTimeI64OK {
				t.Fatalf("unexpected trap for %s", operator)
			}
		}
	})
	if allocations != 0 {
		t.Fatalf("scalar arithmetic allocated %v objects", allocations)
	}
}

func BenchmarkCompileTimeI64Binary(b *testing.B) {
	for _, test := range []struct {
		name, operator string
		left, right    int64
	}{
		{"addition", "+", 3000000000, 77},
		{"multiply", "*", 3000000000, 3000000000},
		{"division", "/", math.MaxInt64, 77},
		{"negative_bitwise", "&", math.MinInt64, -77},
		{"left_shift", "<<", -1, 63},
	} {
		b.Run(test.name, func(b *testing.B) {
			evaluator := compileTimeEvaluator{analyzer: &analyzer{}}
			module := &moduleScope{file: File{Path: "i64.fy"}}
			expression := &ast.Expression{Text: test.operator}
			want, code := exactCompileTimeI64(test.operator, test.left, test.right)
			if code != "" {
				b.Fatal(code)
			}
			b.ReportAllocs()
			for b.Loop() {
				value, ok := evaluator.applyI64Binary(module, expression, test.left, test.right)
				if !ok || value.i != want {
					b.Fatal("incorrect benchmark result")
				}
			}
		})
	}
}
