package mir

import (
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

// Execution follows the copy dependencies, but block storage runs against them.
// The unused Region parameter enables actual-local type tracking.
func reverseFieldStoreChain(copies int) Function {
	function := Function{
		Symbol: 1, Name: "test::type_chain", ResultType: hir.TypeI32,
		Parameters: []LocalID{0, LocalID(copies + 1)}, ParameterTypes: []hir.Type{hir.TypeRegion, hir.TypeI32},
		Entry: BlockID(copies - 1), Blocks: make([]Block, copies),
	}
	for index := range copies {
		block := Block{ID: BlockID(index), Operations: []Operation{{
			Kind: OperationAssign, DestinationLocal: LocalID(index + 1),
			Operands: []Operand{{Kind: OperandLocal, Local: LocalID(index + 2), Type: hir.TypeI32}},
		}}, Terminator: Terminator{Kind: TerminatorGoto,
			Successors: []BlockID{BlockID(index - 1)}, UnwindSuccessor: NoBlock}}
		if index == 0 {
			block.Terminator = Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock,
				ReturnValue: Operand{Kind: OperandLocal, Local: 1, Type: hir.TypeI32}}
		}
		function.Blocks[index] = block
	}
	return function
}

func TestFieldStoreLocalTypesReverseVerifiedChain(t *testing.T) {
	t.Parallel()
	function := reverseFieldStoreChain(32)
	if failures := Verify(Program{Functions: []Function{function}}); len(failures) != 0 {
		t.Fatalf("invalid benchmark graph: %+v", failures)
	}
	types, err := fieldStoreLocalTypes(&function, nil, nil)
	if err != nil || len(types) != 34 || types[0] != hir.TypeRegion {
		t.Fatalf("type tracking = %v, %v", types, err)
	}
	for local := LocalID(1); local <= 33; local++ {
		if types[local] != hir.TypeI32 {
			t.Fatalf("local %d: got %s, want i32", local, types[local])
		}
	}
}

func TestFieldStoreLocalTypesDeferredCopies(t *testing.T) {
	t.Parallel()
	copyLocal := func(destination, source LocalID) Operation {
		return Operation{Kind: OperationAssign, DestinationLocal: destination,
			Operands: []Operand{{Kind: OperandLocal, Local: source, Type: hir.TypeI32}}}
	}
	seed := func(local LocalID, kind OperandKind) Operation {
		return Operation{Kind: OperationAssign, DestinationLocal: local, Operands: []Operand{{Kind: kind}}}
	}
	initialize := copyLocal(2, 1)
	initialize.Kind = OperationInitialize
	for _, test := range []struct {
		name       string
		operations []Operation
		want       map[LocalID]hir.Type
		conflict   string
	}{
		{"forward", []Operation{seed(0, OperandU32), copyLocal(1, 0)},
			map[LocalID]hir.Type{0: hir.TypeU32, 1: hir.TypeU32}, ""},
		{"fanout", []Operation{copyLocal(3, 0), copyLocal(2, 0), copyLocal(1, 2), seed(0, OperandU32)},
			map[LocalID]hir.Type{0: hir.TypeU32, 1: hir.TypeU32, 2: hir.TypeU32, 3: hir.TypeU32}, ""},
		{"initialize", []Operation{initialize, seed(1, OperandI64)},
			map[LocalID]hir.Type{1: hir.TypeI64, 2: hir.TypeI64}, ""},
		{"unseeded-cycle", []Operation{copyLocal(0, 1), copyLocal(1, 0)},
			map[LocalID]hir.Type{}, ""},
		{"seeded-cycle", []Operation{copyLocal(0, 1), copyLocal(1, 0), seed(0, OperandU32)},
			map[LocalID]hir.Type{0: hir.TypeU32, 1: hir.TypeU32}, ""},
		{"independent-chains", []Operation{copyLocal(0, 1), copyLocal(2, 3), seed(1, OperandU32), seed(3, OperandI64)},
			map[LocalID]hir.Type{0: hir.TypeU32, 1: hir.TypeU32, 2: hir.TypeI64, 3: hir.TypeI64}, ""},
		{"conflict", []Operation{copyLocal(2, 1), seed(2, OperandU32), seed(1, OperandI32)},
			nil, "local 2 has conflicting types"},
		{"no-destination", []Operation{copyLocal(NoLocal, 0), seed(0, OperandU32)},
			map[LocalID]hir.Type{0: hir.TypeU32}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := Function{Parameters: []LocalID{99}, ParameterTypes: []hir.Type{hir.TypeRegion},
				Blocks: []Block{{Operations: test.operations}}}
			got, err := fieldStoreLocalTypes(&function, nil, nil)
			if test.conflict != "" {
				if got != nil || err == nil || !strings.Contains(err.Error(), test.conflict) {
					t.Fatalf("got %v, %v; want %s", got, err, test.conflict)
				}
				return
			}
			test.want[99] = hir.TypeRegion
			if err != nil || !maps.Equal(got, test.want) {
				t.Fatalf("got %v, %v; want %v", got, err, test.want)
			}
		})
	}
}

func TestFieldStoreLocalTypesDeferredCallResult(t *testing.T) {
	t.Parallel()
	for _, foreign := range []bool{false, true} {
		t.Run(fmt.Sprintf("foreign_%t", foreign), func(t *testing.T) {
			function := Function{Parameters: []LocalID{99}, ParameterTypes: []hir.Type{hir.TypeRegion},
				Blocks: []Block{{
					Operations: []Operation{{Kind: OperationAssign, DestinationLocal: 2,
						Operands: []Operand{{Kind: OperandLocal, Local: 1}}}},
					Terminator: Terminator{Kind: TerminatorCall, Callee: 7, DestinationLocal: 1},
				}}}
			var functions map[hir.SymbolID]*Function
			var externals map[hir.SymbolID]*ForeignFunction
			if foreign {
				externals = map[hir.SymbolID]*ForeignFunction{7: {ResultType: hir.TypeI64}}
			} else {
				functions = map[hir.SymbolID]*Function{7: {ResultType: hir.TypeI64}}
			}
			got, err := fieldStoreLocalTypes(&function, functions, externals)
			if err != nil || got[1] != hir.TypeI64 || got[2] != hir.TypeI64 {
				t.Fatalf("call result propagation: %v, %v", got, err)
			}
		})
	}
}

func BenchmarkFieldStoreLocalTypesReverseChain(b *testing.B) {
	for _, copies := range []int{128, 1024, 4096} {
		b.Run(fmt.Sprintf("copies_%d", copies), func(b *testing.B) {
			function := reverseFieldStoreChain(copies)
			b.ReportAllocs()
			for b.Loop() {
				types, err := fieldStoreLocalTypes(&function, nil, nil)
				if err != nil || len(types) != copies+2 || types[1] != hir.TypeI32 {
					b.Fatalf("type tracking: %v, %d locals", err, len(types))
				}
			}
		})
	}
}

func BenchmarkFieldStoreLocalTypesForwardChain(b *testing.B) {
	function := reverseFieldStoreChain(1024)
	operations := make([]Operation, 0, 1024)
	for index := len(function.Blocks) - 1; index >= 0; index-- {
		operations = append(operations, function.Blocks[index].Operations...)
	}
	function.Entry = 0
	function.Blocks = []Block{{ID: 0, Operations: operations, Terminator: function.Blocks[0].Terminator}}
	b.ReportAllocs()
	for b.Loop() {
		types, err := fieldStoreLocalTypes(&function, nil, nil)
		if err != nil || len(types) != 1026 || types[1] != hir.TypeI32 {
			b.Fatalf("type tracking: %v, %d locals", err, len(types))
		}
	}
}
