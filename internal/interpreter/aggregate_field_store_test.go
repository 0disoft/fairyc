package interpreter

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func aggregateFieldStoreProgram(count int, resource bool) mir.Program {
	program := aggregateFieldReadProgram(count, resource)
	function := &program.Functions[0]
	operations := function.Blocks[0].Operations
	owner := operations[0].Type
	for i := 1; i < len(operations); i++ {
		operations[i].Kind = mir.OperationFieldStore
		operations[i].Symbol = hir.NoSymbol
		operations[i].DestinationLocal = mir.NoLocal
		operations[i].Operands = append(operations[i].Operands, mir.Operand{Kind: mir.OperandI32, I32: int32(count + i - 1)})
	}
	function.Blocks[0].Operations = append(operations, mir.Operation{
		Kind: mir.OperationFieldRead, DestinationLocal: 1, OwnerType: owner,
		Type: hir.TypeI32, Field: fmt.Sprintf("field_%04d", count-1),
		Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 0}},
	})
	return program
}

func TestExecuteWideAggregateFieldStores(t *testing.T) {
	for _, resource := range []bool{false, true} {
		for _, count := range []int{8, 16, 17, 256} {
			t.Run(fmt.Sprintf("resource_%t/%d", resource, count), func(t *testing.T) {
				outcome, err := Execute(aggregateFieldStoreProgram(count, resource), "main::read")
				if err != nil || outcome.Kind != OutcomeReturn || outcome.Value != int32(2*count-1) {
					t.Fatalf("execution: %+v, %v", outcome, err)
				}
			})
		}
	}
}

func TestIndexedFieldStoresPreserveSelectionAndErrors(t *testing.T) {
	for _, kind := range []string{"struct", "resource", "nested"} {
		t.Run(kind, func(t *testing.T) {
			definition := aggregateFieldReadProgram(64, false).StructTypes[0]
			definition.Fields[0].Type = hir.TypeI64
			// The old scan chose the first matching name AND type, even with
			// malformed duplicate names. Indexing must not change that choice.
			definition.Fields = append(definition.Fields,
				mir.StructField{Name: "duplicate", Type: hir.TypeI64},
				mir.StructField{Name: "duplicate", Type: hir.TypeI32})
			values := make([]runtimeValue, len(definition.Fields))
			for i, field := range definition.Fields {
				values[i] = i32Value(int32(i))
				if field.Type == hir.TypeI64 {
					values[i] = i64Value(int64(i))
				}
			}
			owner, err := structValue(&definition, values)
			if err != nil {
				t.Fatal(err)
			}
			fields := owner.structure().fields
			op := mir.Operation{Kind: mir.OperationFieldStore, Symbol: hir.NoSymbol,
				DestinationLocal: mir.NoLocal, Type: hir.TypeI32,
				Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 0}, {Kind: mir.OperandI32, I32: 73}}}
			if kind == "resource" {
				resource := mir.ResourceType{Type: definition.Type, DropKind: hir.ResourceDropEmpty}
				for _, field := range definition.Fields {
					resource.Fields = append(resource.Fields, mir.ResourceField(field))
				}
				owner, err = resourceValue(&resource, values)
				if err != nil {
					t.Fatal(err)
				}
				fields = owner.resource().fields
			} else if kind == "nested" {
				root := aggregateFieldReadProgram(64, false).StructTypes[0]
				root.Type = hir.ParseType("main::Root")
				root.Fields[63].Type = definition.Type
				rootValues := make([]runtimeValue, len(root.Fields))
				for i := range rootValues {
					rootValues[i] = i32Value(0)
				}
				rootValues[63] = owner
				owner, err = structValue(&root, rootValues)
				if err != nil {
					t.Fatal(err)
				}
				fields = owner.structure().fields[63].structure().fields
				op.Fields = []string{root.Fields[63].Name}
			}
			op.OwnerType = owner.typ
			locals := map[mir.LocalID]runtimeValue{0: owner}
			var index aggregateFieldIndex
			for attempt, slots := range []*aggregateFieldIndex{nil, &index} {
				written := int32(73 + attempt)
				op.Operands[1].I32 = written
				for _, name := range []string{"field_0001", "field_0063", "duplicate"} {
					op.Field = name
					if err := storeField(locals, op, nil, slots); err != nil {
						t.Fatal(err)
					}
				}
				if fields[1].i32() != written || fields[63].i32() != written || fields[64].i64() != 64 || fields[65].i32() != written {
					t.Fatal("field selection or nested storage changed")
				}
				for _, name := range []string{"absent", "field_0000"} {
					op.Field = name
					if err := storeField(locals, op, nil, slots); err == nil {
						t.Fatalf("invalid field %q accepted", name)
					}
				}
			}
			if kind == "resource" {
				owner.resource().dropped = true
				op.Field = "field_0001"
				if err := storeField(locals, op, nil, &index); err == nil {
					t.Fatal("cached store accepted a dropped resource")
				}
			} else if kind == "nested" {
				op.Fields = []string{"absent"}
				if err := storeField(locals, op, nil, &index); err == nil {
					t.Fatal("cached store accepted a missing path")
				}
			}
		})
	}
}

// Verification is outside the timer. Construction, stores, the final read and
// execution-local index construction are all included in each measured entry.
func BenchmarkExecuteWideAggregateFieldStores(b *testing.B) {
	for _, resource := range []bool{false, true} {
		for _, count := range []int{8, 256, 4096} {
			b.Run(fmt.Sprintf("resource_%t/%d", resource, count), func(b *testing.B) {
				prepared, err := Prepare(aggregateFieldStoreProgram(count, resource))
				if err != nil {
					b.Fatal(err)
				}
				options, err := (Options{}).normalized()
				if err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				for b.Loop() {
					outcome, err := prepared.executeEntry("main::read", hir.TypeI32, options)
					if err != nil || outcome.Kind != OutcomeReturn || outcome.Value != int32(2*count-1) {
						b.Fatalf("execution: %+v, %v", outcome, err)
					}
				}
			})
		}
	}
}
