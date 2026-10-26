package mir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestVerifyPreservesExplicitAndFallbackDiagnosticEntities(t *testing.T) {
	for _, entity := range []string{"", "source:명령:42"} {
		for _, operationFailure := range []bool{true, false} {
			t.Run(fmt.Sprintf("entity=%q/operation=%t", entity, operationFailure), func(t *testing.T) {
				function := Function{Symbol: 1, Name: "main::invalid", Entry: 0, ResultType: hir.TypeVoid,
					Blocks: []Block{{ID: 0, Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock}}}}
				wantCode, wantEntity := CodeOperationKind, "block:0.operation:0"
				if operationFailure {
					function.Blocks[0].Operations = []Operation{{Kind: OperationInvalid, Entity: entity}}
				} else {
					function.Blocks[0].Terminator.Entity = entity
					function.Blocks[0].Terminator.ReturnValue = Operand{Kind: OperandI32, Type: hir.TypeI32, I32: 1}
					wantCode, wantEntity = CodeReturnShape, "block:0.terminator"
				}
				if entity != "" {
					wantEntity = entity
				}
				failures := Verify(Program{Functions: []Function{function}})
				if len(failures) != 1 || failures[0].Code != wantCode || failures[0].Entity != wantEntity {
					t.Fatalf("Verify() = %+v, want one %s at %q", failures, wantCode, wantEntity)
				}
			})
		}
	}
}

func BenchmarkVerifyNamedEntities(b *testing.B) {
	for _, size := range []int{256, 4096} {
		b.Run(fmt.Sprintf("operations=%d", size), func(b *testing.B) {
			function := Function{Symbol: 1, Name: "main::named", Entry: 0, ResultType: hir.TypeI32,
				Parameters: []LocalID{1}, ParameterTypes: []hir.Type{hir.TypeI32},
				Blocks: []Block{{ID: 0, Operations: make([]Operation, size),
					Terminator: Terminator{Kind: TerminatorReturn, Entity: "return:source", UnwindSuccessor: NoBlock,
						ReturnValue: Operand{Kind: OperandLocal, Local: 1, Type: hir.TypeI32}}}}}
			for index := range function.Blocks[0].Operations {
				function.Blocks[0].Operations[index] = Operation{Kind: OperationUse, Symbol: 1, Entity: fmt.Sprintf("use:source:%d", index)}
			}
			program := Program{Functions: []Function{function}}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if failures := Verify(program); len(failures) != 0 {
					b.Fatal(failures)
				}
			}
		})
	}
}
