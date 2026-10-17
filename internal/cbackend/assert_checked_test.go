package cbackend

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"math"
	"testing"
)

func TestAssertCheckedProofBoundaries(t *testing.T) {
	local := func(id mir.LocalID) mir.Operand { return mir.Operand{Kind: mir.OperandLocal, Local: id} }
	maximum := mir.Operand{Kind: mir.OperandI64, I64: math.MaxInt64}
	makeFunction := func() mir.Function {
		return mir.Function{Symbol: 1, Entry: 0, Parameters: []mir.LocalID{1}, ParameterTypes: []hir.Type{hir.TypeI64}, Blocks: []mir.Block{
			{ID: 0, Operations: []mir.Operation{{Kind: mir.OperationBinary, DestinationLocal: 2, Operator: "==", OwnerType: hir.TypeI64, Operands: []mir.Operand{local(1), maximum}}}, Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, SwitchKind: "assert", SwitchOperands: []mir.Operand{local(2)}, Successors: []mir.BlockID{1, 2}, UnwindSuccessor: mir.NoBlock}},
			{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, SwitchKind: "checked-arithmetic", SwitchType: hir.TypeI64, Operator: "+", SwitchOperands: []mir.Operand{local(1), {Kind: mir.OperandI64, I64: 1}}, Successors: []mir.BlockID{3, 4}, UnwindSuccessor: mir.NoBlock}},
			{ID: 2, Terminator: mir.Terminator{Kind: mir.TerminatorTrap, UnwindSuccessor: mir.NoBlock}},
			{ID: 3, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock}},
			{ID: 4, Terminator: mir.Terminator{Kind: mir.TerminatorTrap, UnwindSuccessor: mir.NoBlock}},
		}}
	}
	f := makeFunction()
	proof := assertedCheckedFailuresForC([]mir.Function{f})[1]
	if target, ok := proof[1]; !ok || target != 4 {
		t.Fatalf("assertion success proof: %v", proof)
	}
	reachable := reachableBlocksForC(f, nil, proof)
	if reachable[3] || !reachable[2] || !reachable[4] {
		t.Fatalf("trap ordering: %v", reachable)
	}
	for name, change := range map[string]func(*mir.Function){
		"rebind": func(f *mir.Function) {
			f.Blocks[1].Operations = []mir.Operation{{Kind: mir.OperationAssign, DestinationLocal: 1, Operands: []mir.Operand{{Kind: mir.OperandI64, I64: 40}}}}
		},
		"indirect-write": func(f *mir.Function) { f.Blocks[1].Operations = []mir.Operation{{Kind: mir.OperationDerefStore}} },
		"unknown-op":     func(f *mir.Function) { f.Blocks[1].Operations = []mir.Operation{{Kind: mir.OperationScalarCast}} },
		"wrong-place":    func(f *mir.Function) { f.Blocks[1].Terminator.SwitchOperands[0] = local(9) },
		"false-edge":     func(f *mir.Function) { f.Blocks[0].Terminator.Successors = []mir.BlockID{2, 1} },
		"merge": func(f *mir.Function) {
			f.Blocks[2].Terminator = mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}
		},
		"backedge": func(f *mir.Function) {
			f.Blocks[3].Terminator = mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}
		},
		"unwind": func(f *mir.Function) { f.Blocks[0].Terminator.UnwindSuccessor = 1 },
		"unknown-call": func(f *mir.Function) {
			f.Blocks[0].Terminator = mir.Terminator{Kind: mir.TerminatorCall, Callee: 99, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := makeFunction()
			change(&f)
			if got := assertedCheckedFailuresForC([]mir.Function{f})[1]; len(got) != 0 {
				t.Fatalf("unproven failure: %v", got)
			}
		})
	}
}

func TestAssertCheckedAddressedLocalInvalidation(t *testing.T) {
	s := newAssertionState()
	root := mir.Operand{Kind: mir.OperandLocal, Local: 1}
	s.operation(mir.Operation{Kind: mir.OperationUnary, Operator: "&mut", DestinationLocal: 2, Operands: []mir.Operand{root}})
	s.facts["v1"] = mir.Operand{Kind: mir.OperandI64, I64: math.MaxInt64}
	s.operation(mir.Operation{Kind: mir.OperationAssign, DestinationLocal: 1, Operands: []mir.Operand{{Kind: mir.OperandI64, I64: 40}}})
	if len(s.facts) != 0 || len(s.addressed) != 0 {
		t.Fatal("addressed-local write retained stale facts")
	}
}

func TestAssertCheckedPureReturnBoundary(t *testing.T) {
	f := mir.Function{Symbol: 1, Parameters: []mir.LocalID{1}, ParameterTypes: []hir.Type{hir.ParseType("&i64")}, ResultType: hir.ParseType("&i64"), Blocks: []mir.Block{{Terminator: mir.Terminator{Kind: mir.TerminatorReturn, ReturnValue: mir.Operand{Kind: mir.OperandLocal, Local: 1}, UnwindSuccessor: mir.NoBlock}}}}
	if len(pureFieldReturns([]mir.Function{f})) != 1 {
		t.Fatal("identity reference helper not recognized")
	}
	f.Blocks[0].Operations = []mir.Operation{{Kind: mir.OperationDerefStore}}
	if len(pureFieldReturns([]mir.Function{f})) != 0 {
		t.Fatal("writing helper summarized as pure")
	}
}
