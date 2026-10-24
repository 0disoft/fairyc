package mir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func branchOnlyFunction(branches int) Function {
	function := Function{
		Symbol: 1, Name: "test::branches", Entry: 0, ResultType: hir.TypeVoid,
		Parameters: []LocalID{0}, ParameterTypes: []hir.Type{hir.TypeBool},
		Blocks: make([]Block, branches*3+1),
	}
	for n := range branches {
		entry := BlockID(n * 3)
		function.Blocks[entry] = Block{ID: entry, Terminator: Terminator{
			Kind: TerminatorSwitch, SwitchKind: "bool", UnwindSuccessor: NoBlock,
			SwitchOperands: []Operand{{Kind: OperandLocal, Local: 0, Type: hir.TypeBool}},
			Successors:     []BlockID{entry + 1, entry + 2},
		}}
		for _, offset := range []BlockID{1, 2} {
			function.Blocks[entry+offset] = Block{ID: entry + offset, Terminator: Terminator{
				Kind: TerminatorGoto, Successors: []BlockID{entry + 3}, UnwindSuccessor: NoBlock,
			}}
		}
	}
	last := BlockID(len(function.Blocks) - 1)
	function.Blocks[last] = Block{ID: last, Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock}}
	return function
}

func TestVerifyStillRejectsLateUncheckedArithmetic(t *testing.T) {
	function := branchOnlyFunction(3)
	program := Program{Functions: []Function{function}}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("branch-only function: %+v", failures)
	}
	last := &function.Blocks[len(function.Blocks)-1]
	last.Operations = []Operation{{
		Kind: OperationBinary, Entity: "late-add", Operator: "+", Type: hir.TypeI32, OwnerType: hir.TypeI32, DestinationLocal: 1,
		Operands: []Operand{{Kind: OperandI32, Type: hir.TypeI32, I32: 1}, {Kind: OperandI32, Type: hir.TypeI32, I32: 2}},
	}}
	if failures := Verify(program); len(failures) != 1 || failures[0].Code != "V1501" || failures[0].Entity != "late-add" {
		t.Fatalf("unchecked arithmetic after branch-only blocks: %+v", failures)
	}
}

func BenchmarkVerifyBranchOnlyFunction(b *testing.B) {
	for _, branches := range []int{32, 128, 341} {
		function := branchOnlyFunction(branches)
		b.Run(fmt.Sprintf("blocks_%d", len(function.Blocks)), func(b *testing.B) {
			program := Program{Functions: []Function{function}}
			b.ReportAllocs()
			for b.Loop() {
				if failures := Verify(program); len(failures) != 0 {
					b.Fatalf("Verify: %+v", failures)
				}
			}
		})
	}
}
