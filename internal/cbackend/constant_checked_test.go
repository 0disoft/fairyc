package cbackend

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"math"
	"math/big"
	"testing"
)

func TestConstantCheckedFailureFacts(t *testing.T) {
	makeBlock := func() mir.Block {
		return mir.Block{Operations: []mir.Operation{
			{Kind: mir.OperationAssign, DestinationLocal: 4, Operands: []mir.Operand{{Kind: mir.OperandI64, I64: math.MaxInt64}}},
			{Kind: mir.OperationAssign, DestinationLocal: 5, Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 4}}},
			{Kind: mir.OperationArithmeticCheck},
		}, Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, SwitchKind: "checked-arithmetic", SwitchType: hir.TypeI64, Operator: "+", SwitchOperands: []mir.Operand{{Kind: mir.OperandLocal, Local: 5}, {Kind: mir.OperandI64, I64: 1}}, Successors: []mir.BlockID{1, 2}}}
	}
	if target, ok := constantCheckedFailure(makeBlock()); !ok || target != 2 {
		t.Fatalf("copy chain: %v %v", target, ok)
	}
	for name, change := range map[string]func(*mir.Block){
		"indirect-write":    func(b *mir.Block) { b.Operations = append(b.Operations, mir.Operation{Kind: mir.OperationDerefStore}) },
		"unknown-operation": func(b *mir.Block) { b.Operations = append(b.Operations, mir.Operation{Kind: mir.OperationFieldRead}) },
		"reassign": func(b *mir.Block) {
			b.Operations = append(b.Operations, mir.Operation{Kind: mir.OperationAssign, DestinationLocal: 5, Operands: []mir.Operand{{Kind: mir.OperandI64, I64: 40}}})
		},
		"unknown-reassign": func(b *mir.Block) {
			b.Operations = append(b.Operations, mir.Operation{Kind: mir.OperationAssign, DestinationLocal: 5, Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 99}}})
		},
		"unknown-input":        func(b *mir.Block) { b.Operations = nil },
		"wrong-literal-kind":   func(b *mir.Block) { b.Terminator.SwitchOperands[1] = mir.Operand{Kind: mir.OperandI32, I32: 1} },
		"no-cross-call":        func(b *mir.Block) { b.Terminator.Kind = mir.TerminatorCall },
		"unsupported-division": func(b *mir.Block) { b.Terminator.SwitchKind = "checked-division"; b.Terminator.Operator = "/" },
		"unsupported-shift":    func(b *mir.Block) { b.Terminator.SwitchKind = "checked-shift"; b.Terminator.Operator = "<<" },
		"wrong-arity":          func(b *mir.Block) { b.Terminator.Successors = nil },
	} {
		t.Run(name, func(t *testing.T) {
			b := makeBlock()
			change(&b)
			if _, ok := constantCheckedFailure(b); ok {
				t.Fatal("unproven failure accepted")
			}
		})
	}
	b := makeBlock()
	b.Terminator.UnwindSuccessor = 3
	fn := mir.Function{Entry: 0, Blocks: []mir.Block{b, {ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock}}, {ID: 2, Terminator: mir.Terminator{Kind: mir.TerminatorTrap, UnwindSuccessor: mir.NoBlock}}, {ID: 3, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock}}}}
	reach := reachableBlocksForC(fn, nil, nil)
	if reach[1] || !reach[2] || !reach[3] {
		t.Fatalf("failure/unwind reachability: %v", reach)
	}
}

// The oracle performs mathematical arithmetic in big.Int, independently of the
// fixed-width comparisons used by the C emission proof. Only tests use big.Int.
func TestLiteralCheckedOverflowMathematicalOracle(t *testing.T) {
	for _, typ := range []hir.Type{hir.TypeI32, hir.TypeI64, hir.TypeU8, hir.TypeU32, hir.TypeUSize} {
		var values []mir.Operand
		lower, upper := new(big.Int), new(big.Int)
		if typ == hir.TypeI32 || typ == hir.TypeI64 {
			lo, hi := int64(math.MinInt64), int64(math.MaxInt64)
			if typ == hir.TypeI32 {
				lo, hi = math.MinInt32, math.MaxInt32
			}
			lower.SetInt64(lo)
			upper.SetInt64(hi)
			for _, n := range []int64{lo, lo + 1, -2, -1, 0, 1, 2, hi - 1, hi} {
				o := mir.Operand{Kind: mir.OperandI64, I64: n}
				if typ == hir.TypeI32 {
					o = mir.Operand{Kind: mir.OperandI32, I32: int32(n)}
				}
				values = append(values, o)
			}
		} else {
			k, max := mir.OperandUSize, uint64(math.MaxUint64)
			if typ == hir.TypeU8 {
				k, max = mir.OperandU8, math.MaxUint8
			}
			if typ == hir.TypeU32 {
				k, max = mir.OperandU32, math.MaxUint32
			}
			upper.SetUint64(max)
			for _, n := range []uint64{0, 1, 2, max / 2, max/2 + 1, max - 1, max} {
				values = append(values, mir.Operand{Kind: k, U64: n})
			}
		}
		number := func(o mir.Operand) *big.Int {
			switch o.Kind {
			case mir.OperandI32:
				return big.NewInt(int64(o.I32))
			case mir.OperandI64:
				return big.NewInt(o.I64)
			default:
				return new(big.Int).SetUint64(o.U64)
			}
		}
		for _, l := range values {
			if typ == hir.TypeI32 || typ == hir.TypeI64 {
				result := new(big.Int).Neg(number(l))
				want := result.Cmp(lower) < 0 || result.Cmp(upper) > 0
				if got := literalCheckedOverflow(typ, "-", "checked-unary", []mir.Operand{l}); got != want {
					t.Fatalf("%s negate %+v: %v want %v", typ, l, got, want)
				}
			}
			for _, r := range values {
				for _, op := range []string{"+", "-", "*"} {
					result := new(big.Int)
					switch op {
					case "+":
						result.Add(number(l), number(r))
					case "-":
						result.Sub(number(l), number(r))
					case "*":
						result.Mul(number(l), number(r))
					}
					want := result.Cmp(lower) < 0 || result.Cmp(upper) > 0
					if got := literalCheckedOverflow(typ, op, "checked-arithmetic", []mir.Operand{l, r}); got != want {
						t.Fatalf("%s %+v %s %+v: %v want %v", typ, l, op, r, got, want)
					}
				}
			}
		}
	}
}
