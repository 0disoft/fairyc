package packageartifact

import (
	"fmt"
	"reflect"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// Stable bodies bind helper IDs, not process-local symbols or native pointers.
// The complete rebound program still enters mir.Verify before publication.
func validateStableClosureRecords(fn mir.Function) error {
	if fn.ClosureTarget || fn.ClosureCaptures != 0 {
		target := mir.Clone(mir.Program{Functions: []mir.Function{fn}})
		target.Functions[0].Symbol = 1
		if failures := mir.Verify(target); len(failures) != 0 {
			return fmt.Errorf("stable copied closure target: %w", failures[0])
		}
	}
	for _, block := range fn.Blocks {
		for _, op := range block.Operations {
			if op.Kind != mir.OperationClosureCreate {
				continue
			}
			canonical := mir.Operation{Kind: mir.OperationClosureCreate, Entity: op.Entity, Span: op.Span, Symbol: hir.NoSymbol, Type: hir.TypeClosureI32, DestinationLocal: op.DestinationLocal, Operands: op.Operands, Variant: op.Variant}
			if op.DestinationLocal < 0 || len(op.Operands) > 4 || !reflect.DeepEqual(op, canonical) {
				return fmt.Errorf("invalid stable copied closure construction")
			}
			for _, value := range op.Operands {
				if value.Kind != mir.OperandLocal && value.Kind != mir.OperandI32 || value.Type != (hir.Type{}) && value.Type != hir.TypeI32 || value.I64 != 0 || value.U64 != 0 || value.StaticData != "" || value.Kind == mir.OperandLocal && (value.Local < 0 || value.I32 != 0) {
					return fmt.Errorf("invalid stable copied closure capture")
				}
			}
		}
		term := block.Terminator
		if term.Kind != mir.TerminatorClosureCall {
			continue
		}
		canonical := mir.Terminator{Kind: mir.TerminatorClosureCall, Entity: term.Entity, Span: term.Span, DestinationLocal: term.DestinationLocal, Arguments: term.Arguments, Successors: term.Successors, UnwindSuccessor: mir.NoBlock}
		if term.DestinationLocal < 0 || len(term.Arguments) != 2 || len(term.Successors) != 1 || !reflect.DeepEqual(term, canonical) || term.Arguments[0].Kind != mir.OperandLocal {
			return fmt.Errorf("invalid stable copied closure invocation")
		}
	}
	return nil
}
