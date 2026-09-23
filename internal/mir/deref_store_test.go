package mir

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestVerifyFieldStoreScalarDeref(t *testing.T) {
	op := Operation{Kind: OperationDerefStore, Symbol: hir.NoSymbol, DestinationLocal: NoLocal,
		OwnerType: hir.MutableBorrow(hir.TypeI32), Type: hir.TypeI32,
		Operands: []Operand{{Kind: OperandLocal, Local: 2}, {Kind: OperandI32, I32: 42}}}
	program := Program{Functions: []Function{{Symbol: 1, Name: "main::set", Parameters: []LocalID{2, 4},
		ParameterTypes: []hir.Type{hir.MutableBorrow(hir.TypeI32), hir.TypeBool}, ResultType: hir.TypeVoid,
		Entry: 0, Blocks: []Block{{ID: 0, Operations: []Operation{op},
			Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock}}}}}}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("valid dereference store: %+v", failures)
	}
	for name, mutate := range map[string]func(*Operation){
		"shared reference": func(op *Operation) { op.OwnerType = hir.SharedBorrow(hir.TypeI32) },
		"wrong receiver":   func(op *Operation) { op.Operands[0].Local = 4 },
		"wrong value":      func(op *Operation) { op.Operands[1] = Operand{Kind: OperandUSize, U64: 42} },
		"missing value":    func(op *Operation) { op.Operands = op.Operands[:1] },
		"destination":      func(op *Operation) { op.DestinationLocal = 3 },
		"field":            func(op *Operation) { op.Field = "value" },
		"move":             func(op *Operation) { op.MovePayload = true },
	} {
		t.Run(name, func(t *testing.T) {
			bad := op
			bad.Operands = append([]Operand(nil), op.Operands...)
			mutate(&bad)
			program.Functions[0].Blocks[0].Operations[0] = bad
			if failures := Verify(program); len(failures) == 0 {
				t.Fatal("invalid dereference store accepted")
			}
		})
	}
}
