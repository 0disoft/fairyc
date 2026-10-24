package mir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func distinctArithmeticChecks(count int) Function {
	function := Function{Symbol: 1, Name: "test::checks", ResultType: hir.TypeVoid,
		Entry: 0, Blocks: make([]Block, count*3+1)}
	_, categories, _ := checkedArithmeticSpec("+", 2)
	for index := range count {
		check, success, trap := BlockID(index*3), BlockID(index*3+1), BlockID(index*3+2)
		operands := []Operand{{Kind: OperandI32, I32: int32(index), Type: hir.TypeI32}, {Kind: OperandI32, I32: 1, Type: hir.TypeI32}}
		function.Blocks[check] = Block{ID: check, Operations: []Operation{{Kind: OperationArithmeticCheck,
			Operator: "+", Operands: operands, Type: hir.TypeI32, OwnerType: hir.TypeI32,
			DestinationLocal: NoLocal, TrapCategories: categories}}, Terminator: Terminator{
			Kind: TerminatorSwitch, SwitchKind: "checked-arithmetic", Operator: "+", SwitchType: hir.TypeI32,
			SwitchOperands: operands, Successors: []BlockID{success, trap}, UnwindSuccessor: NoBlock}}
		function.Blocks[success] = Block{ID: success, Operations: []Operation{{Kind: OperationBinary,
			Operator: "+", Operands: operands, Type: hir.TypeI32, OwnerType: hir.TypeI32, DestinationLocal: LocalID(index)}},
			Terminator: Terminator{Kind: TerminatorGoto, Successors: []BlockID{check + 3}, UnwindSuccessor: NoBlock}}
		function.Blocks[trap] = Block{ID: trap, Terminator: Terminator{Kind: TerminatorTrap,
			TrapCategories: categories, UnwindSuccessor: NoBlock}}
	}
	last := BlockID(len(function.Blocks) - 1)
	function.Blocks[last] = Block{ID: last, Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock}}
	return function
}

func TestVerifyDistinctArithmeticChecks(t *testing.T) {
	function := distinctArithmeticChecks(8)
	if failures := Verify(Program{Functions: []Function{function}}); len(failures) != 0 {
		t.Fatalf("invalid benchmark graph: %+v", failures)
	}
}

func TestIndexedChecksUseLastMatchingMarker(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*Operation)
		valid  bool
	}{
		{"last valid", func(op *Operation) {}, true},
		{"last wrong categories", func(op *Operation) { op.TrapCategories = nil }, false},
		{"last wrong owner", func(op *Operation) { op.OwnerType = hir.TypeU32 }, false},
		{"last wrong result", func(op *Operation) { op.Type = hir.TypeU32 }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := distinctArithmeticChecks(1)
			first := function.Blocks[0].Operations[0]
			last := first
			first.Entity, last.Entity = "first-marker", "last-marker"
			test.mutate(&last)
			function.Blocks[0].Operations = []Operation{first, last}
			failures := verifyTrapCheckEdges(&function)
			if test.valid {
				if len(failures) != 0 {
					t.Fatalf("valid last marker rejected: %+v", failures)
				}
			} else if len(failures) != 1 || failures[0].Code != "V1501" || failures[0].Entity != last.Entity {
				t.Fatalf("last malformed marker lost its diagnostic: %+v", failures)
			}
		})
	}
}

func TestIndexedChecksStillAcceptLaterCompleteDominator(t *testing.T) {
	function := distinctArithmeticChecks(2)
	early := function.Blocks[3].Operations[0]
	early.Entity, early.TrapCategories = "early-invalid-marker", nil
	function.Blocks[0].Operations = append(function.Blocks[0].Operations, early)
	if failures := verifyTrapCheckEdges(&function); len(failures) != 0 {
		t.Fatalf("earlier invalid candidate hid the later complete check: %+v", failures)
	}
}

func BenchmarkVerifyDistinctArithmeticChecks(b *testing.B) {
	for _, count := range []int{64, 256, 512} {
		b.Run(fmt.Sprintf("checks_%d", count), func(b *testing.B) {
			function := distinctArithmeticChecks(count)
			b.ReportAllocs()
			for b.Loop() {
				if failures := verifyTrapCheckEdges(&function); len(failures) != 0 {
					b.Fatalf("verify: %+v", failures)
				}
			}
		})
	}
}
