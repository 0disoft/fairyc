package mir

import (
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestLowerFieldBorrowAssignmentFailsClosed(t *testing.T) {
	_, err := Lower(hir.Package{Modules: []hir.Module{{Path: "main", Functions: []hir.Function{{
		Symbol: 1, Name: "edit", ResultType: hir.TypeVoid,
		Body: hir.Block{Statements: []hir.Statement{{Kind: "assignment", Operator: "=",
			Target:     &hir.Expression{Kind: "field-expression", Type: hir.TypeI32},
			Expression: &hir.Expression{Kind: "integer", Type: hir.TypeI32, Value: "1"},
		}}},
	}}}}})
	var unsupported *Unsupported
	if !errors.As(err, &unsupported) || unsupported.Feature != "field assignment without executable field storage" {
		t.Fatalf("field assignment must not disappear during lowering: %v", err)
	}
}

func TestVerifyFieldBorrow(t *testing.T) {
	owner := hir.Type("main::Owner")
	program := Program{StructTypes: []StructType{{Type: owner, Name: "Owner", Fields: []StructField{{Name: "value", Type: hir.TypeI32}}}},
		Functions: []Function{{Symbol: 1, Name: "main::field", Parameters: []LocalID{2},
			ParameterTypes: []hir.Type{hir.SharedBorrow(owner)}, ResultType: hir.SharedBorrow(hir.TypeI32), Entry: 0,
			Blocks: []Block{{ID: 0, Operations: []Operation{{Kind: OperationFieldBorrow, Symbol: hir.NoSymbol,
				DestinationLocal: 3, OwnerType: hir.SharedBorrow(owner), Type: hir.SharedBorrow(hir.TypeI32),
				Field: "value", Operands: []Operand{{Kind: OperandLocal, Local: 2}}}},
				Terminator: Terminator{Kind: TerminatorReturn, ReturnValue: Operand{Kind: OperandLocal, Local: 3}, UnwindSuccessor: NoBlock}}}}}}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("valid field borrow: %+v", failures)
	}
	op := &program.Functions[0].Blocks[0].Operations[0]
	baseline := *op
	for name, mutate := range map[string]func(*Operation){
		"mutable result":     func(op *Operation) { op.Type = hir.MutableBorrow(hir.TypeI32) },
		"mutable owner":      func(op *Operation) { op.OwnerType = hir.MutableBorrow(owner) },
		"owned receiver":     func(op *Operation) { op.OwnerType = owner },
		"wrong field type":   func(op *Operation) { op.Type = hir.SharedBorrow(hir.TypeU32) },
		"missing field":      func(op *Operation) { op.Field = "missing" },
		"missing operand":    func(op *Operation) { op.Operands = nil },
		"clobbered owner":    func(op *Operation) { op.DestinationLocal = 2 },
		"ownership transfer": func(op *Operation) { op.MovePayload = true },
	} {
		t.Run(name, func(t *testing.T) {
			*op = baseline
			mutate(op)
			if failures := Verify(program); len(failures) == 0 {
				t.Fatal("malformed field borrow accepted")
			}
		})
	}
}
