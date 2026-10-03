package mir

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestVerifyNestedFieldStore(t *testing.T) {
	const root = hir.Type("main::Token")
	const child = hir.Type("main::Span")
	makeProgram := func() Program {
		return Program{StructTypes: []StructType{
			{Type: root, Name: "Token", Fields: []StructField{{Name: "position", Type: child}, {Name: "other", Type: hir.TypeBool}}},
			{Type: child, Name: "Span", Fields: []StructField{{Name: "start", Type: hir.TypeI64}, {Name: "end", Type: hir.TypeI64}}}},
			Functions: []Function{{Symbol: 1, Name: "main::set", Parameters: []LocalID{2, 4}, ParameterTypes: []hir.Type{hir.MutableBorrow(root), hir.TypeBool}, ResultType: hir.TypeVoid, Entry: 0,
				Blocks: []Block{{ID: 0, Operations: []Operation{{Kind: OperationFieldStore, Symbol: hir.NoSymbol, DestinationLocal: NoLocal, OwnerType: hir.MutableBorrow(root), Type: hir.TypeI64, Field: "start", Fields: []string{"position"}, Operands: []Operand{{Kind: OperandLocal, Local: 2}, {Kind: OperandI64, I64: 42}}}}, Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock}}}}}}
	}
	if failures := Verify(makeProgram()); len(failures) != 0 {
		t.Fatalf("valid: %+v", failures)
	}
	for name, mutate := range map[string]func(*Program){
		"missing ancestor": func(p *Program) { p.Functions[0].Blocks[0].Operations[0].Fields = []string{"missing"} },
		"empty ancestor":   func(p *Program) { p.Functions[0].Blocks[0].Operations[0].Fields = []string{""} },
		"scalar ancestor":  func(p *Program) { p.Functions[0].Blocks[0].Operations[0].Fields = []string{"other"} },
		"missing leaf":     func(p *Program) { p.Functions[0].Blocks[0].Operations[0].Field = "missing" },
		"shared root":      func(p *Program) { p.Functions[0].Blocks[0].Operations[0].OwnerType = hir.SharedBorrow(root) },
		"wrong receiver":   func(p *Program) { p.Functions[0].Blocks[0].Operations[0].Operands[0].Local = 4 },
		"wrong value": func(p *Program) {
			p.Functions[0].Blocks[0].Operations[0].Operands[1] = Operand{Kind: OperandLocal, Local: 4}
		},
		"borrow crossing": func(p *Program) { p.StructTypes[0].Fields[0].Type = hir.MutableBorrow(child) },
		"cycle":           func(p *Program) { p.StructTypes[1].Fields[0].Type = root },
		"wrong leaf type": func(p *Program) { p.StructTypes[1].Fields[0].Type = hir.TypeBool },
		"aggregate leaf":  func(p *Program) { p.Functions[0].Blocks[0].Operations[0].Type = child },
	} {
		t.Run(name, func(t *testing.T) {
			p := makeProgram()
			mutate(&p)
			if len(Verify(p)) == 0 {
				t.Fatal("invalid nested store accepted")
			}
		})
	}
}
