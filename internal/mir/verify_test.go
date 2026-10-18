package mir

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestVerifyRejectsMissingInternalCallTarget(t *testing.T) {
	t.Parallel()
	program := Program{Functions: []Function{{
		Symbol: 1, Name: "test::caller", ResultType: hir.TypeVoid, Entry: 0,
		Blocks: []Block{
			{ID: 0, Terminator: Terminator{Kind: TerminatorCall, Entity: "terminator:call", Callee: 2, Successors: []BlockID{1}, UnwindSuccessor: NoBlock}},
			{ID: 1, Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock}},
		},
	}}}
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != CodeCallSignature || failures[0].Entity != "terminator:call" {
		t.Fatalf("Verify() = %+v", failures)
	}
}

func TestVerifyBurnRequiresOneLiveDirectResource(t *testing.T) {
	t.Parallel()
	coinType := hir.ParseType("main::Coin")
	program := Program{
		ResourceTypes: []ResourceType{{
			Type: coinType, Fields: []ResourceField{{Name: "amount", Type: hir.TypeI32}},
			DropKind: hir.ResourceDropEmpty, DropSymbol: hir.NoSymbol,
		}},
		Functions: []Function{{
			Symbol: 1, Name: "main::burn_coin", Parameters: []LocalID{2}, ParameterTypes: []hir.Type{coinType},
			ResultType: hir.TypeVoid, Entry: 0,
			Resources: []ResourceLocal{{Symbol: 2, Type: coinType, Parameter: true, Entity: "resource:coin"}},
			Blocks: []Block{{
				ID: 0,
				Operations: []Operation{{
					Kind: OperationBurn, Entity: "burn:coin", Symbol: 2, DestinationLocal: NoLocal, Type: coinType,
				}},
				Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock},
			}},
		}},
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify(valid burn) = %+v", failures)
	}
	program.Functions[0].Blocks[0].Operations[0].Type = hir.TypeI32
	failures := Verify(program)
	found := false
	for _, failure := range failures {
		found = found || failure.Code == "V1104"
	}
	if !found {
		t.Fatalf("Verify(malformed burn) = %+v", failures)
	}
}

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
	if len(failures) != 1 || failures[0].Code != CodeControlFlow {
		t.Fatalf("Verify() failures = %+v, want the classified shape failure", failures)
	}
}

func TestVerifyRejectsMalformedBorrowedViewLength(t *testing.T) {
	t.Parallel()
	program := Program{Functions: []Function{{
		Name:       "main::main",
		ResultType: hir.TypeI32,
		Entry:      0,
		Blocks: []Block{{
			ID: 0,
			Operations: []Operation{{
				Kind: OperationViewLength, DestinationLocal: 1,
				Operands:  []Operand{{Kind: OperandString, Type: hir.SharedBorrow(hir.TypeStr), StaticData: "fairy"}},
				OwnerType: hir.SharedBorrow(hir.TypeStr), Type: hir.TypeUSize, Field: "capacity",
			}},
			Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock, ReturnValue: Operand{Kind: OperandI32, I32: 0}},
		}},
	}}}
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() failures = %+v, want the classified shape failure", failures)
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
	expected := []VerificationError{
		{Code: CodeOperationShape, Entity: "block:0.operation:0", Detail: "Result operation has an invalid variant"},
		{Code: CodeOperationShape, Entity: "block:0.operation:0", Detail: "Result operation must have one operand and one destination"},
		{Code: CodeOperationType, Entity: "block:0.operation:0", Detail: "Result construction has no concrete Result type"},
	}
	if !slices.Equal(failures, expected) {
		t.Fatalf("Verify() failures = %+v, want all independent result failures", failures)
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
	if len(failures) != 1 || failures[0].Code != CodeControlFlow {
		t.Fatalf("Verify() failures = %+v, want the classified shape failure", failures)
	}
}

func TestVerifyRejectsOptionNoneWithPayload(t *testing.T) {
	t.Parallel()
	program := Program{Functions: []Function{{
		Name:  "main::main",
		Entry: 0,
		Blocks: []Block{{
			ID: 0,
			Operations: []Operation{{
				Kind: OperationOptionConstruct, DestinationLocal: 1, Variant: "none", Type: hir.Option(hir.TypeI32),
				Operands: []Operand{{Kind: OperandI32, I32: 42}},
			}},
			Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock, ReturnValue: Operand{Kind: OperandI32, I32: 0}},
		}},
	}}}
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() failures = %+v, want the classified shape failure", failures)
	}
}

func TestVerifyRejectsTaggedOperationsWithoutConcreteTypes(t *testing.T) {
	t.Parallel()
	operations := []Operation{
		{Kind: OperationResultConstruct, DestinationLocal: 1, Variant: "ok", Operands: []Operand{{Kind: OperandI32, I32: 42}}},
		{Kind: OperationOptionConstruct, DestinationLocal: 1, Variant: "some", Operands: []Operand{{Kind: OperandI32, I32: 42}}},
		{Kind: OperationResultPayload, DestinationLocal: 1, Variant: "ok", Operands: []Operand{{Kind: OperandI32, I32: 42}}},
		{Kind: OperationOptionPayload, DestinationLocal: 1, Variant: "some", Operands: []Operand{{Kind: OperandI32, I32: 42}}},
	}
	for _, operation := range operations {
		operation := operation
		t.Run(operation.Kind.String(), func(t *testing.T) {
			t.Parallel()
			program := Program{Functions: []Function{{
				Name:  "main::main",
				Entry: 0,
				Blocks: []Block{{
					ID:         0,
					Operations: []Operation{operation},
					Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock, ReturnValue: Operand{Kind: OperandI32, I32: 0}},
				}},
			}}}
			failures := Verify(program)
			if len(failures) != 1 || failures[0].Code != CodeOperationType {
				t.Fatalf("Verify() failures = %+v, want the classified shape failure", failures)
			}
		})
	}
}

func TestVerifyRejectsResourceConstructionFieldMismatch(t *testing.T) {
	t.Parallel()
	program := Program{
		ResourceTypes: []ResourceType{{
			Type: hir.ParseType("main::Token"), Entity: "resource-type:main-token",
			Fields: []ResourceField{{Name: "value", Type: hir.TypeI32}}, DropKind: hir.ResourceDropEmpty, DropSymbol: hir.NoSymbol,
		}},
		Functions: []Function{{
			Name:  "main::main",
			Entry: 0,
			Blocks: []Block{{
				ID: 0,
				Operations: []Operation{{
					Kind: OperationResourceConstruct, DestinationLocal: 1, Type: hir.ParseType("main::Token"),
					Fields: []string{"value"}, Operands: []Operand{{Kind: OperandI32, I32: 1}, {Kind: OperandI32, I32: 2}},
				}},
				Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock, ReturnValue: Operand{Kind: OperandI32, I32: 0}},
			}},
		}}}
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() failures = %+v, want the classified shape failure", failures)
	}
}

func TestVerifyRejectsResourceDropGlueEffectMismatch(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@effects(filesystem)
resource Token {
    i32 value;
    cleanup { let i32 marker = this.value; }
}
@effects(filesystem)
fn i32 main() {
    let Token token = Token { .value = 42 };
    return 0;
}
`)
	for index := range program.Functions {
		if program.Functions[index].DropGlueFor != hir.ParseType("") {
			program.Functions[index].Effects = nil
		}
	}
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != CodeDropGlue || !strings.Contains(failures[0].Detail, "effects") {
		t.Fatalf("Verify() failures = %+v, want one drop-glue failure", failures)
	}
}

func TestVerifyRejectsStructConstructionFieldMismatch(t *testing.T) {
	t.Parallel()
	program := Program{
		StructTypes: []StructType{{
			Type: hir.ParseType("main::Packet"), Name: "Packet", Entity: "struct-type:main-packet", ReprC: true,
			Fields: []StructField{{Name: "code", Type: hir.TypeU32}},
		}},
		Functions: []Function{{
			Name: "main::main", Entry: 0,
			Blocks: []Block{{
				ID: 0,
				Operations: []Operation{{
					Kind: OperationStructConstruct, DestinationLocal: 1, Type: hir.ParseType("main::Packet"),
					Fields: []string{"wrong"}, Operands: []Operand{{Kind: OperandU32, U64: 1}},
				}},
				Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock, ReturnValue: Operand{Kind: OperandI32, I32: 0}},
			}},
		}},
	}
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != CodeAggregateUse {
		t.Fatalf("Verify() failures = %+v, want the classified shape failure", failures)
	}
	// An empty constructor is valid only for an actually fieldless definition.
	program.Functions[0].Blocks[0].Operations[0].Fields = nil
	program.Functions[0].Blocks[0].Operations[0].Operands = nil
	failures = Verify(program)
	if len(failures) != 1 || failures[0].Code != CodeAggregateUse {
		t.Fatalf("empty constructor for nonempty type: %+v", failures)
	}
}

func TestVerifyRejectsPayloadMoveOnNonPayloadOperation(t *testing.T) {
	t.Parallel()
	program := Program{Functions: []Function{{
		Name:  "main::main",
		Entry: 0,
		Blocks: []Block{{
			ID: 0,
			Operations: []Operation{{
				Kind: OperationAssign, DestinationLocal: 1, MovePayload: true,
				Operands: []Operand{{Kind: OperandI32, I32: 42}},
			}},
			Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock, ReturnValue: Operand{Kind: OperandI32, I32: 0}},
		}},
	}}}
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() failures = %+v, want the classified shape failure", failures)
	}
}
