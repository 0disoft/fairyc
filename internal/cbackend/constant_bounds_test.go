package cbackend

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestConstantByteViewFailureBoundaries(t *testing.T) {
	bytes := func(text string) mir.Operand { return mir.Operand{Kind: mir.OperandByteString, StaticData: text} }
	index := func(n uint64) mir.Operand { return mir.Operand{Kind: mir.OperandUSize, U64: n} }
	for _, test := range []struct {
		name, kind string
		operands   []mir.Operand
		fail       bool
	}{
		{"empty index", "checked-view-index", []mir.Operand{bytes(""), index(0)}, true},
		{"index at end", "checked-view-index", []mir.Operand{bytes("a"), index(1)}, true},
		{"valid index", "checked-view-index", []mir.Operand{bytes("a"), index(0)}, false},
		{"UTF-8 byte index", "checked-view-index", []mir.Operand{bytes("한글"), index(5)}, false},
		{"unknown index", "checked-view-index", []mir.Operand{bytes("a"), {Kind: mir.OperandLocal, Local: 1}}, false},
		{"unknown view", "checked-view-index", []mir.Operand{{Kind: mir.OperandLocal, Local: 1}, index(1)}, false},
		{"reversed slice", "checked-view-slice", []mir.Operand{bytes("ab"), index(2), index(1)}, true},
		{"slice past end", "checked-view-slice", []mir.Operand{bytes("a"), index(0), index(2)}, true},
		{"empty slice at end", "checked-view-slice", []mir.Operand{bytes("a"), index(1), index(1)}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := literalCheckedFailure(mir.Terminator{SwitchKind: test.kind}, test.operands); got != test.fail {
				t.Fatalf("failure = %v, want %v", got, test.fail)
			}
		})
	}
}

func TestByteViewFailureProofInvalidation(t *testing.T) {
	local := mir.Operand{Kind: mir.OperandLocal, Local: 1}
	makeFunction := func() mir.Function {
		return mir.Function{Symbol: 1, Entry: 0, Blocks: []mir.Block{
			{ID: 0, Operations: []mir.Operation{{Kind: mir.OperationAssign, DestinationLocal: 1, Operands: []mir.Operand{{Kind: mir.OperandByteString, StaticData: "a"}}}}, Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}},
			{ID: 1, Operations: []mir.Operation{{Kind: mir.OperationViewIndexCheck, DestinationLocal: mir.NoLocal}}, Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, SwitchKind: "checked-view-index", SwitchOperands: []mir.Operand{local, {Kind: mir.OperandUSize, U64: 1}}, Successors: []mir.BlockID{2, 3}, UnwindSuccessor: mir.NoBlock}},
			{ID: 2, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock}},
			{ID: 3, Terminator: mir.Terminator{Kind: mir.TerminatorTrap, UnwindSuccessor: mir.NoBlock}},
		}}
	}
	f := makeFunction()
	proof := assertedCheckedFailuresForC([]mir.Function{f})[hir.SymbolID(1)]
	if next, ok := proof[1]; !ok || next != 3 || !nonReturningFunctionsForC([]mir.Function{f}, checkedFailureProofs{1: proof})[1] {
		t.Fatalf("constant bounds did not establish a non-returning callee: %v", proof)
	}
	for name, change := range map[string]func(*mir.Function){
		"rebound to valid view": func(f *mir.Function) {
			f.Blocks[1].Operations = []mir.Operation{{Kind: mir.OperationAssign, DestinationLocal: 1, Operands: []mir.Operand{{Kind: mir.OperandByteString, StaticData: "ab"}}}}
		},
		"indirect write": func(f *mir.Function) { f.Blocks[1].Operations = []mir.Operation{{Kind: mir.OperationDerefStore}} },
		"unknown call": func(f *mir.Function) {
			f.Blocks[0].Terminator = mir.Terminator{Kind: mir.TerminatorCall, Callee: 99, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}
		},
		"join": func(f *mir.Function) {
			f.Blocks[2].Terminator = mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := makeFunction()
			change(&f)
			if got := assertedCheckedFailuresForC([]mir.Function{f})[1]; len(got) != 0 {
				t.Fatalf("stale bounds proof = %v", got)
			}
		})
	}
}
