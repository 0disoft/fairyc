package interpreter

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// Build and verify outside the timer; each measured entry still constructs its
// own aggregate and execution state, including any lazy field lookup tables.
func aggregateFieldReadProgram(count int, resource bool) mir.Program {
	owner := hir.ParseType("main::Record")
	fields := make([]mir.StructField, count)
	values := make([]mir.Operand, count)
	names := make([]string, count)
	operations := make([]mir.Operation, count+1)
	for i := range fields {
		fields[i] = mir.StructField{Name: fmt.Sprintf("field_%04d", i), Type: hir.TypeI32}
		names[i] = fields[i].Name
		values[i] = mir.Operand{Kind: mir.OperandI32, I32: int32(i)}
		operations[i+1] = mir.Operation{Kind: mir.OperationFieldRead, DestinationLocal: 1,
			OwnerType: owner, Type: hir.TypeI32, Field: fields[i].Name,
			Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 0}}}
	}
	operations[0] = mir.Operation{Kind: mir.OperationStructConstruct, DestinationLocal: 0,
		Type: owner, Fields: names, Operands: values}
	program := mir.Program{StructTypes: []mir.StructType{{Type: owner, Name: "Record", Fields: fields}},
		Functions: []mir.Function{{Symbol: 1, Name: "main::read", Entry: 0, ResultType: hir.TypeI32,
			Blocks: []mir.Block{{ID: 0, Operations: operations, Terminator: mir.Terminator{
				Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock,
				ReturnValue: mir.Operand{Kind: mir.OperandLocal, Local: 1},
			}}},
		}}}
	if resource {
		definition := mir.ResourceType{Type: owner, DropKind: hir.ResourceDropEmpty, DropSymbol: hir.NoSymbol}
		for _, field := range fields {
			definition.Fields = append(definition.Fields, mir.ResourceField(field))
		}
		program.ResourceTypes = []mir.ResourceType{definition}
		program.StructTypes = nil
		operations[0].Kind = mir.OperationResourceConstruct
	}
	return program
}

func TestExecuteWideAggregateFieldReads(t *testing.T) {
	for _, resource := range []bool{false, true} {
		for _, count := range []int{8, 16, 17, 256} {
			t.Run(fmt.Sprintf("resource_%t/%d", resource, count), func(t *testing.T) {
				outcome, err := Execute(aggregateFieldReadProgram(count, resource), "main::read")
				if err != nil || outcome.Kind != OutcomeReturn || outcome.Value != int32(count-1) {
					t.Fatalf("execution: %+v, %v", outcome, err)
				}
			})
		}
	}
}

func BenchmarkExecuteWideAggregateFieldReads(b *testing.B) {
	for _, resource := range []bool{false, true} {
		for _, count := range []int{8, 256, 4096} {
			b.Run(fmt.Sprintf("resource_%t/%d", resource, count), func(b *testing.B) {
				prepared, err := Prepare(aggregateFieldReadProgram(count, resource))
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
					if err != nil || outcome.Kind != OutcomeReturn || outcome.Value != int32(count-1) {
						b.Fatalf("execution: %+v, %v", outcome, err)
					}
				}
			})
		}
	}
}
