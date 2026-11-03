package cbackend

import (
	"math"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestAssertedCheckedI32LiteralNegation(t *testing.T) {
	i32 := func(n int32) mir.Operand { return mir.Operand{Kind: mir.OperandI32, Type: hir.TypeI32, I32: n} }
	for _, item := range []struct {
		name    string
		operand mir.Operand
		clobber bool
		failure bool
	}{
		{"negative-one", i32(1), false, true},
		{"implicit-i32-negative-one", mir.Operand{Kind: mir.OperandI32, I32: 1}, false, true},
		{"positive-one", i32(-1), false, false},
		{"zero", i32(0), false, false},
		{"minimum-does-not-wrap", i32(math.MinInt32), false, false},
		{"unknown-input", mir.Operand{Kind: mir.OperandLocal, Type: hir.TypeI32, Local: 99}, false, false},
		{"bool-is-not-an-i32-fact", mir.Operand{Kind: mir.OperandI32, Type: hir.TypeBool, I32: 1}, false, false},
		{"other-width", mir.Operand{Kind: mir.OperandI64, Type: hir.TypeI64, I64: 1}, false, false},
		{"indirect-write-clears-facts", i32(1), true, false},
	} {
		t.Run(item.name, func(t *testing.T) {
			operations := []mir.Operation{{Kind: mir.OperationUnary, Operator: "-", OwnerType: hir.TypeI32, DestinationLocal: 1, Operands: []mir.Operand{item.operand}}}
			if item.clobber {
				operations = append(operations, mir.Operation{Kind: mir.OperationDerefStore})
			}
			function := mir.Function{Symbol: 1, Entry: 0, Blocks: []mir.Block{
				{ID: 0, Operations: operations, Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}},
				{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, SwitchKind: "checked-arithmetic", SwitchType: hir.TypeI32, Operator: "-", SwitchOperands: []mir.Operand{i32(math.MaxInt32), {Kind: mir.OperandLocal, Type: hir.TypeI32, Local: 1}}, Successors: []mir.BlockID{2, 3}, UnwindSuccessor: mir.NoBlock}},
				{ID: 2, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, ReturnValue: i32(0), UnwindSuccessor: mir.NoBlock}},
				{ID: 3, Terminator: mir.Terminator{Kind: mir.TerminatorTrap, TrapCategories: []string{"integer-overflow"}, UnwindSuccessor: mir.NoBlock}},
			}}
			proofs := assertedCheckedFailuresForC([]mir.Function{function})
			selected, found := proofs[1][1]
			if found != item.failure || found && selected != 3 {
				t.Fatalf("failure proof: %v %v", selected, found)
			}
			if got := nonReturningFunctionsForC([]mir.Function{function}, proofs)[1]; got != item.failure {
				t.Fatalf("non-returning proof: %v", got)
			}
		})
	}
}
