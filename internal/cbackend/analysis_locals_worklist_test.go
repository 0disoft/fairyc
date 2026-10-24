package cbackend

import (
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// Storage order runs against the dependency chain, while execution still
// initializes every source before use: entry -> seed -> ... -> alias -> return.
func reverseLocalChain(aliases int) mir.Function {
	function := mir.Function{Symbol: 1, Name: "test::chain", ResultType: hir.TypeU32, Entry: 0,
		Blocks: make([]mir.Block, aliases+2)}
	function.Blocks[0] = mir.Block{ID: 0, Terminator: mir.Terminator{
		Kind: mir.TerminatorGoto, Successors: []mir.BlockID{mir.BlockID(aliases + 1)}, UnwindSuccessor: mir.NoBlock,
	}}
	for index := 1; index <= aliases+1; index++ {
		local := mir.LocalID(aliases + 1 - index)
		operand := mir.Operand{Kind: mir.OperandLocal, Local: local - 1, Type: hir.TypeU32}
		if local == 0 {
			operand = mir.Operand{Kind: mir.OperandU32, U64: 42, Type: hir.TypeU32}
		}
		block := mir.Block{ID: mir.BlockID(index), Operations: []mir.Operation{{
			Kind: mir.OperationAssign, DestinationLocal: local, Operands: []mir.Operand{operand},
		}}, Terminator: mir.Terminator{Kind: mir.TerminatorGoto,
			Successors: []mir.BlockID{mir.BlockID(index - 1)}, UnwindSuccessor: mir.NoBlock}}
		if index == 1 {
			block.Terminator = mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock,
				ReturnValue: mir.Operand{Kind: mir.OperandLocal, Local: local, Type: hir.TypeU32}}
		}
		function.Blocks[index] = block
	}
	return function
}

func TestInferLocalTypesReverseVerifiedChain(t *testing.T) {
	function := reverseLocalChain(8)
	if failures := mir.Verify(mir.Program{Functions: []mir.Function{function}}); len(failures) != 0 {
		t.Fatalf("invalid benchmark graph: %+v", failures)
	}
	analysis := emissionAnalysis{functions: []mir.Function{function}}
	types, err := analysis.inferLocalTypes(0)
	if err != nil {
		t.Fatal(err)
	}
	for local := range 9 {
		if got := types[mir.LocalID(local)]; got != hir.TypeU32 {
			t.Fatalf("local %d: got %s, want u32", local, got)
		}
	}
}

func TestInferLocalTypesDeferredCopies(t *testing.T) {
	copyLocal := func(destination, source mir.LocalID) mir.Operation {
		return mir.Operation{Kind: mir.OperationAssign, DestinationLocal: destination,
			Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: source}}}
	}
	seed := func(local mir.LocalID, kind mir.OperandKind, value hir.Type) mir.Operation {
		return mir.Operation{Kind: mir.OperationAssign, DestinationLocal: local,
			Operands: []mir.Operand{{Kind: kind, Type: value}}}
	}
	initialize := copyLocal(2, 1)
	initialize.Kind = mir.OperationInitialize
	for _, test := range []struct {
		name       string
		operations []mir.Operation
		want       map[mir.LocalID]hir.Type
		failure    string
	}{
		{"fanout", []mir.Operation{copyLocal(3, 0), copyLocal(2, 0), copyLocal(1, 2), seed(0, mir.OperandU32, hir.TypeU32)},
			map[mir.LocalID]hir.Type{0: hir.TypeU32, 1: hir.TypeU32, 2: hir.TypeU32, 3: hir.TypeU32}, ""},
		{"initialize", []mir.Operation{initialize, seed(1, mir.OperandI64, hir.TypeI64)},
			map[mir.LocalID]hir.Type{1: hir.TypeI64, 2: hir.TypeI64}, ""},
		{"unseeded-cycle", []mir.Operation{copyLocal(0, 1), copyLocal(1, 0)},
			map[mir.LocalID]hir.Type{0: hir.TypeI32, 1: hir.TypeI32}, ""},
		{"seeded-cycle", []mir.Operation{copyLocal(0, 1), copyLocal(1, 0), seed(0, mir.OperandU32, hir.TypeU32)},
			map[mir.LocalID]hir.Type{0: hir.TypeU32, 1: hir.TypeU32}, ""},
		{"conflict", []mir.Operation{copyLocal(2, 1), seed(2, mir.OperandU32, hir.TypeU32), seed(1, mir.OperandI32, hir.TypeBool)},
			nil, "operation assign in test::deferred: local 2 has conflicting types"},
		{"compatible-bool-i32", []mir.Operation{copyLocal(2, 1), seed(2, mir.OperandI32, hir.TypeI32), seed(1, mir.OperandI32, hir.TypeBool)},
			map[mir.LocalID]hir.Type{1: hir.TypeBool, 2: hir.TypeI32}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			analysis := emissionAnalysis{functions: []mir.Function{{Name: "test::deferred", Blocks: []mir.Block{{Operations: test.operations}}}}}
			got, err := analysis.inferLocalTypes(0)
			if test.failure != "" {
				if err == nil || !strings.Contains(err.Error(), test.failure) {
					t.Fatalf("got %v, want %s", err, test.failure)
				}
				return
			}
			if err != nil || !maps.Equal(got, test.want) {
				t.Fatalf("got %v, %v; want %v", got, err, test.want)
			}
		})
	}
}

func TestInferLocalTypesDeferredCallResult(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		t.Run(fmt.Sprintf("foreign_%t", foreign), func(t *testing.T) {
			function := mir.Function{Blocks: []mir.Block{{
				Operations: []mir.Operation{{Kind: mir.OperationAssign, DestinationLocal: 2,
					Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 1}}}},
				Terminator: mir.Terminator{Kind: mir.TerminatorCall, Callee: 2, DestinationLocal: 1},
			}}}
			analysis := emissionAnalysis{functions: []mir.Function{function}}
			if foreign {
				analysis.foreignFunctions = []mir.ForeignFunction{{Symbol: 2, ResultType: hir.TypeI64}}
				analysis.foreignBySymbol = map[hir.SymbolID]int{2: 0}
			} else {
				analysis.functions = append(analysis.functions, mir.Function{Symbol: 2, ResultType: hir.TypeI64})
				analysis.bySymbol = map[hir.SymbolID]int{2: 1}
			}
			got, err := analysis.inferLocalTypes(0)
			if err != nil || got[1] != hir.TypeI64 || got[2] != hir.TypeI64 {
				t.Fatalf("call result propagation: %v, %v", got, err)
			}
		})
	}
}

func BenchmarkInferLocalTypesReverseChain(b *testing.B) {
	for _, aliases := range []int{128, 1024, 4096} {
		b.Run(fmt.Sprintf("aliases_%d", aliases), func(b *testing.B) {
			analysis := emissionAnalysis{functions: []mir.Function{reverseLocalChain(aliases)}}
			b.ReportAllocs()
			for b.Loop() {
				types, err := analysis.inferLocalTypes(0)
				if err != nil || len(types) != aliases+1 || types[mir.LocalID(aliases)] != hir.TypeU32 {
					b.Fatalf("inference: %v, %d locals", err, len(types))
				}
			}
		})
	}
}

func BenchmarkInferLocalTypesForwardChain(b *testing.B) {
	function := reverseLocalChain(1024)
	operations := make([]mir.Operation, 0, 1025)
	for index := len(function.Blocks) - 1; index > 0; index-- {
		operations = append(operations, function.Blocks[index].Operations...)
	}
	function.Blocks = []mir.Block{{ID: 0, Operations: operations, Terminator: function.Blocks[1].Terminator}}
	analysis := emissionAnalysis{functions: []mir.Function{function}}
	b.ReportAllocs()
	for b.Loop() {
		types, err := analysis.inferLocalTypes(0)
		if err != nil || len(types) != 1025 || types[1024] != hir.TypeU32 {
			b.Fatalf("inference: %v, %d locals", err, len(types))
		}
	}
}
