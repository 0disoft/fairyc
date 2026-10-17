package mir

import (
	"github.com/0disoft/fairyc/internal/hir"
	"testing"
)

func TestVerifyPlainStructFieldStore(t *testing.T) {
	var root = hir.ParseType("main::Token")
	var position = hir.ParseType("main::Position")
	var leaf = hir.ParseType("main::Span")
	makeProgram := func(nested bool) Program {
		value, field, ancestors := position, "position", []string(nil)
		if nested {
			value, field, ancestors = leaf, "span", []string{"position"}
		}
		return Program{StructTypes: []StructType{
			{Type: root, Name: "Token", Fields: []StructField{{Name: "position", Type: position}}},
			{Type: position, Name: "Position", Fields: []StructField{{Name: "span", Type: leaf}}},
			{Type: leaf, Name: "Span", Fields: []StructField{{Name: "start", Type: hir.TypeI64}, {Name: "end", Type: hir.TypeI64}}},
		}, Functions: []Function{{Symbol: 1, Name: "main::set", Parameters: []LocalID{2, 4, 6}, ParameterTypes: []hir.Type{hir.MutableBorrow(root), value, hir.TypeBool}, ResultType: hir.TypeVoid, Entry: 0,
			Blocks: []Block{{ID: 0, Operations: []Operation{{Kind: OperationFieldStore, Symbol: hir.NoSymbol, DestinationLocal: NoLocal, OwnerType: hir.MutableBorrow(root), Type: value, Field: field, Fields: ancestors, Operands: []Operand{{Kind: OperandLocal, Local: 2}, {Kind: OperandLocal, Local: 4}}}}, Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock}}}}}}
	}
	for _, nested := range []bool{false, true} {
		p := makeProgram(nested)
		if failures := Verify(p); len(failures) != 0 {
			t.Fatalf("valid nested=%v: %+v", nested, failures)
		}
		empty := makeProgram(nested)
		empty.StructTypes[2].Fields = nil
		if failures := Verify(empty); len(failures) != 0 {
			t.Fatalf("valid private empty leaf nested=%v: %+v", nested, failures)
		}
		for name, mutate := range map[string]func(*Program){
			"missing-field":  func(p *Program) { p.Functions[0].Blocks[0].Operations[0].Field = "missing" },
			"wrong-receiver": func(p *Program) { p.Functions[0].Blocks[0].Operations[0].Operands[0].Local = 6 },
			"wrong-value":    func(p *Program) { p.Functions[0].Blocks[0].Operations[0].Operands[1].Local = 6 },
			"fake-type":      func(p *Program) { p.Functions[0].Blocks[0].Operations[0].Type = leaf },
			"shared-receiver": func(p *Program) {
				p.Functions[0].ParameterTypes[0] = hir.SharedBorrow(root)
				p.Functions[0].Blocks[0].Operations[0].OwnerType = hir.SharedBorrow(root)
			},
			"invalid-array-leaf": func(p *Program) { p.StructTypes[2].Fields[0].Type = hir.FixedArray(hir.TypeI32, 0) },
			"borrow-leaf":        func(p *Program) { p.StructTypes[2].Fields[0].Type = hir.SharedBorrow(hir.TypeI64) },
			"pointer-leaf":       func(p *Program) { p.StructTypes[2].Fields[0].Type = hir.RawPointer(hir.TypeI64, true) },
			"cycle":              func(p *Program) { p.StructTypes[2].Fields[0].Type = root },
			"empty-repr-c": func(p *Program) {
				p.StructTypes[2].Fields = nil
				p.StructTypes[2].ReprC = true
			},
			"missing-definition": func(p *Program) { p.StructTypes = p.StructTypes[:2] },
			"immediate-aggregate": func(p *Program) {
				p.Functions[0].Blocks[0].Operations[0].Operands[1] = Operand{Kind: OperandI64, I64: 42}
			},
			"resource-root": func(p *Program) {
				p.StructTypes = p.StructTypes[1:]
				p.ResourceTypes = []ResourceType{{Type: root, Fields: []ResourceField{{Name: "position", Type: position}}}}
			},
			"hidden-nonplain-sibling": func(p *Program) {
				p.StructTypes[0].Fields = append(p.StructTypes[0].Fields, StructField{Name: "other", Type: hir.Option(hir.TypeI32)})
			},
		} {
			t.Run(name+map[bool]string{false: "/direct", true: "/nested"}[nested], func(t *testing.T) {
				p := makeProgram(nested)
				mutate(&p)
				// In the nested case the leaf annotation is already correct; forge the parent instead.
				if name == "fake-type" && nested {
					p.Functions[0].Blocks[0].Operations[0].Type = position
				}
				if len(Verify(p)) == 0 {
					t.Fatal("invalid aggregate store accepted")
				}
			})
		}
	}
}
