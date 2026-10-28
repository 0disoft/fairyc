package mir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func wideFieldReadProgram(count int) Program {
	owner := hir.ParseType("main::Record")
	fields := make([]StructField, count)
	operations := make([]Operation, count)
	for i := range fields {
		fields[i] = StructField{Name: fmt.Sprintf("field_%04d", i), Type: hir.TypeI32}
		operations[i] = Operation{Kind: OperationFieldRead, DestinationLocal: LocalID(i + 1),
			OwnerType: owner, Type: hir.TypeI32, Field: fields[i].Name,
			Operands: []Operand{{Kind: OperandLocal, Local: 0}}}
	}
	return Program{StructTypes: []StructType{{Type: owner, Name: "Record", Fields: fields}},
		Functions: []Function{{Symbol: 1, Name: "main::read", Entry: 0,
			Parameters: []LocalID{0}, ParameterTypes: []hir.Type{owner}, ResultType: hir.TypeI32,
			Blocks: []Block{{ID: 0, Operations: operations, Terminator: Terminator{
				Kind: TerminatorReturn, UnwindSuccessor: NoBlock,
				ReturnValue: Operand{Kind: OperandLocal, Local: LocalID(count)},
			}}},
		}}}
}

func TestAggregateFieldIndexMatchesLinearLookup(t *testing.T) {
	for _, count := range []int{0, 14, 15, 256} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			var index aggregateFieldIndex
			structure := wideFieldReadProgram(count).StructTypes[0]
			resource := ResourceType{Type: structure.Type}
			for _, field := range structure.Fields {
				resource.Fields = append(resource.Fields, ResourceField(field))
			}
			// Keep malformed names and duplicate declarations observable too.
			structure.Fields = append(structure.Fields, StructField{Name: "", Type: hir.TypeU32}, StructField{Name: "field_0000", Type: hir.TypeBool})
			resource.Fields = append(resource.Fields, ResourceField{Name: "", Type: hir.TypeU8}, ResourceField{Name: "field_0000", Type: hir.TypeI64})
			for _, name := range []string{"field_0000", "field_0016", "field_0255", "", "absent"} {
				got, found := index.structField(&structure, name)
				want, exists := structField(structure.Fields, name)
				if found != exists || got != want.Type {
					t.Fatalf("struct %q: got %s/%t, want %s/%t", name, got, found, want.Type, exists)
				}
				got, found = index.resourceField(&resource, name)
				other, exists := resourceField(resource.Fields, name)
				if found != exists || got != other.Type {
					t.Fatalf("resource %q: got %s/%t, want %s/%t", name, got, found, other.Type, exists)
				}
			}
		})
	}
}

func TestVerifyWideAggregateFieldAccess(t *testing.T) {
	for _, resource := range []bool{false, true} {
		for _, kind := range []OperationKind{OperationFieldRead, OperationFieldBorrow, OperationFieldStore} {
			t.Run(fmt.Sprintf("resource_%t/%s", resource, kind), func(t *testing.T) {
				program := wideFieldReadProgram(64)
				owner := program.StructTypes[0].Type
				if resource {
					definition := ResourceType{Type: owner, DropKind: hir.ResourceDropEmpty, DropSymbol: hir.NoSymbol}
					for _, field := range program.StructTypes[0].Fields {
						definition.Fields = append(definition.Fields, ResourceField(field))
					}
					program.ResourceTypes = []ResourceType{definition}
					program.StructTypes = nil
				}
				function := &program.Functions[0]
				function.ParameterTypes[0] = hir.SharedBorrow(owner)
				block := &function.Blocks[0]
				op := block.Operations[len(block.Operations)-1]
				op.Kind, op.Symbol, op.DestinationLocal = kind, hir.NoSymbol, 1
				op.OwnerType = function.ParameterTypes[0]
				block.Terminator.ReturnValue.Local = 1
				if kind == OperationFieldBorrow {
					op.Type = hir.SharedBorrow(hir.TypeI32)
					function.ResultType = op.Type
				}
				if kind == OperationFieldStore {
					op.OwnerType = hir.MutableBorrow(owner)
					function.ParameterTypes[0] = op.OwnerType
					op.DestinationLocal = NoLocal
					op.Operands = append(op.Operands, Operand{Kind: OperandI32, I32: 42})
					function.ResultType = hir.TypeVoid
					block.Terminator.ReturnValue = Operand{}
				}
				block.Operations = []Operation{op}
				if failures := Verify(program); len(failures) != 0 {
					t.Fatalf("valid field access: %+v", failures)
				}
				for _, wrongType := range []bool{false, true} {
					block.Operations[0] = op
					if wrongType {
						if resource {
							program.ResourceTypes[0].Fields[63].Type = hir.TypeBool
						} else {
							program.StructTypes[0].Fields[63].Type = hir.TypeBool
						}
					} else {
						block.Operations[0].Field = "absent"
					}
					if failures := Verify(program); len(failures) == 0 {
						t.Fatalf("accepted invalid field access (wrong type: %t)", wrongType)
					}
				}
			})
		}
	}
}

func BenchmarkVerifyWideFieldReads(b *testing.B) {
	for _, count := range []int{8, 256, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			program := wideFieldReadProgram(count)
			if failures := Verify(program); len(failures) != 0 {
				b.Fatalf("invalid fixture: %+v", failures)
			}
			b.ReportAllocs()
			for b.Loop() {
				if failures := Verify(program); len(failures) != 0 {
					b.Fatal(failures)
				}
			}
		})
	}
}
