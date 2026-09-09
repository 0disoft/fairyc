package mir

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestVerifyFieldStore(t *testing.T) {
	owner := hir.Type("main::State")
	op := Operation{Kind: OperationFieldStore, Symbol: hir.NoSymbol, DestinationLocal: NoLocal,
		OwnerType: hir.MutableBorrow(owner), Type: hir.TypeI32, Field: "count",
		Operands: []Operand{{Kind: OperandLocal, Local: 2}, {Kind: OperandI32, I32: 42}}}
	program := Program{StructTypes: []StructType{{Type: owner, Name: "State", Fields: []StructField{{Name: "count", Type: hir.TypeI32}}}},
		Functions: []Function{{Symbol: 1, Name: "main::set", Parameters: []LocalID{2, 4}, ParameterTypes: []hir.Type{hir.MutableBorrow(owner), hir.TypeBool},
			ResultType: hir.TypeVoid, Entry: 0, Blocks: []Block{{ID: 0, Operations: []Operation{op},
				Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock}}}}}}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("valid field store: %+v", failures)
	}
	for name, mutate := range map[string]func(*Operation){
		"shared owner":     func(op *Operation) { op.OwnerType = hir.SharedBorrow(owner) },
		"missing field":    func(op *Operation) { op.Field = "absent" },
		"wrong type":       func(op *Operation) { op.Type = hir.TypeBool },
		"missing value":    func(op *Operation) { op.Operands = op.Operands[:1] },
		"wrong scalar":     func(op *Operation) { op.Operands = []Operand{op.Operands[0], {Kind: OperandUSize, U64: 42}} },
		"destination":      func(op *Operation) { op.DestinationLocal = 3 },
		"transfer":         func(op *Operation) { op.MovePayload = true },
		"wrong receiver":   func(op *Operation) { op.Operands = []Operand{{Kind: OperandLocal, Local: 4}, op.Operands[1]} },
		"wrong live value": func(op *Operation) { op.Operands = []Operand{op.Operands[0], {Kind: OperandLocal, Local: 2}} },
	} {
		t.Run(name, func(t *testing.T) {
			bad := op
			mutate(&bad)
			program.Functions[0].Blocks[0].Operations[0] = bad
			if failures := Verify(program); len(failures) == 0 {
				t.Fatal("invalid field store accepted")
			}
		})
	}
	for _, scalar := range []struct {
		typ  hir.Type
		kind OperandKind
		max  uint64
	}{
		{hir.TypeU8, OperandU8, 255}, {hir.TypeU32, OperandU32, 4294967295},
	} {
		t.Run(string(scalar.typ), func(t *testing.T) {
			store := op
			store.Type = scalar.typ
			store.Operands = []Operand{op.Operands[0], {Kind: scalar.kind, U64: scalar.max}}
			program.StructTypes[0].Fields[0].Type = scalar.typ
			program.Functions[0].Blocks[0].Operations[0] = store
			if failures := Verify(program); len(failures) != 0 {
				t.Fatalf("unsigned store: %+v", failures)
			}
			store.Operands[1].U64++
			program.Functions[0].Blocks[0].Operations[0] = store
			if failures := Verify(program); len(failures) == 0 {
				t.Fatal("out-of-range store accepted")
			}
		})
	}
}
