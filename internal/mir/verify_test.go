package mir

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestVerifyMergesBranchLocalInitializationByIntersection(t *testing.T) {
	t.Parallel()
	program := Program{Functions: []Function{{
		Name:       "main::main",
		ResultType: hir.TypeI32,
		Entry:      0,
		Blocks: []Block{
			{
				ID: 0,
				Terminator: Terminator{
					Kind: TerminatorSwitch, Successors: []BlockID{1, 2}, UnwindSuccessor: NoBlock,
					SwitchKind: "bool", SwitchOperands: []Operand{{Kind: OperandI32, I32: 1}},
				},
			},
			{
				ID: 1,
				Operations: []Operation{
					{Kind: OperationAssign, DestinationLocal: 10, Operands: []Operand{{Kind: OperandI32, I32: 7}}},
					{Kind: OperationAssign, DestinationLocal: 20, Operands: []Operand{{Kind: OperandLocal, Local: 10}}},
				},
				Terminator: Terminator{Kind: TerminatorGoto, Successors: []BlockID{3}, UnwindSuccessor: NoBlock},
			},
			{
				ID: 2,
				Operations: []Operation{
					{Kind: OperationAssign, DestinationLocal: 20, Operands: []Operand{{Kind: OperandI32, I32: 9}}},
				},
				Terminator: Terminator{Kind: TerminatorGoto, Successors: []BlockID{3}, UnwindSuccessor: NoBlock},
			},
			{
				ID:         3,
				Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock, ReturnValue: Operand{Kind: OperandLocal, Local: 20}},
			},
		},
	}}}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestVerifyRejectsMalformedBoolSwitch(t *testing.T) {
	t.Parallel()
	program := Program{Functions: []Function{{
		Name:  "main::main",
		Entry: 0,
		Blocks: []Block{{
			ID: 0,
			Terminator: Terminator{
				Kind: TerminatorSwitch, Successors: []BlockID{1}, UnwindSuccessor: NoBlock,
				SwitchKind: "bool", SwitchOperands: []Operand{{Kind: OperandI32, I32: 1}},
			},
		}, {
			ID: 1, Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock, ReturnValue: Operand{Kind: OperandI32, I32: 0}},
		}},
	}}}
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != "V1001" {
		t.Fatalf("Verify() failures = %+v, want one V1001", failures)
	}
}

func TestVerifyRejectsMalformedResultOperation(t *testing.T) {
	t.Parallel()
	program := Program{Functions: []Function{{
		Name:  "main::main",
		Entry: 0,
		Blocks: []Block{{
			ID: 0,
			Operations: []Operation{{
				Kind: OperationResultConstruct, DestinationLocal: NoLocal, Variant: "unknown",
			}},
			Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock, ReturnValue: Operand{Kind: OperandI32, I32: 0}},
		}},
	}}}
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != "V1001" {
		t.Fatalf("Verify() failures = %+v, want one compacted V1001", failures)
	}
}

func TestVerifyRejectsMalformedResultSwitch(t *testing.T) {
	t.Parallel()
	program := Program{Functions: []Function{{
		Name:  "main::main",
		Entry: 0,
		Blocks: []Block{{
			ID: 0,
			Terminator: Terminator{
				Kind: TerminatorSwitch, Successors: []BlockID{1, 2}, UnwindSuccessor: NoBlock,
				SwitchKind: "result",
			},
		}, {
			ID: 1, Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock, ReturnValue: Operand{Kind: OperandI32, I32: 0}},
		}, {
			ID: 2, Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock, ReturnValue: Operand{Kind: OperandI32, I32: 0}},
		}},
	}}}
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != "V1001" {
		t.Fatalf("Verify() failures = %+v, want one V1001", failures)
	}
}
