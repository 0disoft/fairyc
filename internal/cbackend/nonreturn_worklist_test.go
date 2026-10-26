package cbackend

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func nonreturnChainFixture(count int) []mir.Function {
	functions := make([]mir.Function, count)
	for i := range functions {
		functions[i] = mir.Function{Symbol: hir.SymbolID(i + 1), Name: fmt.Sprintf("main::f%d", i), ResultType: hir.TypeVoid, Entry: 0,
			Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorTrap, UnwindSuccessor: mir.NoBlock}}}}
		if i+1 < count {
			functions[i].Blocks = []mir.Block{
				{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorCall, Callee: hir.SymbolID(i + 2), DestinationLocal: mir.NoLocal, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}},
				{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock}},
			}
		}
	}
	return functions
}

func TestNonReturningCallChainOrder(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		functions := nonreturnChainFixture(64)
		if reverse {
			slices.Reverse(functions)
		}
		if failures := mir.Verify(mir.Program{Functions: functions}); len(failures) != 0 {
			t.Fatal(failures[0])
		}
		if got := nonReturningFunctionsForC(functions, nil); len(got) != len(functions) {
			t.Fatalf("reverse=%t: inferred %d of %d functions", reverse, len(got), len(functions))
		}
	}
}

func TestNonReturningCallBoundaries(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func([]mir.Function)
		want   map[hir.SymbolID]bool
	}{
		{"returning-leaf", func(f []mir.Function) { f[2].Blocks[0].Terminator.Kind = mir.TerminatorReturn }, map[hir.SymbolID]bool{}},
		{"unknown-callee", func(f []mir.Function) { f[1].Blocks[0].Terminator.Callee = 99 }, map[hir.SymbolID]bool{3: true}},
		{"recursive-cycle", func(f []mir.Function) { f[1].Blocks[0].Terminator.Callee = 1 }, map[hir.SymbolID]bool{3: true}},
		{"unwind-return", func(f []mir.Function) { f[1].Blocks[0].Terminator.UnwindSuccessor = 1 }, map[hir.SymbolID]bool{3: true}},
		{"dead-call", func(f []mir.Function) { f[1].Entry = 1 }, map[hir.SymbolID]bool{3: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			functions := nonreturnChainFixture(3)
			test.change(functions)
			if got := nonReturningFunctionsForC(functions, nil); !maps.Equal(got, test.want) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestNonReturningMultipleCallees(t *testing.T) {
	for _, returningLeaf := range []bool{false, true} {
		functions := nonreturnChainFixture(3)
		functions[0].Parameters = []mir.LocalID{1}
		functions[0].ParameterTypes = []hir.Type{hir.TypeBool}
		functions[0].Blocks = []mir.Block{
			{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, SwitchKind: "bool", SwitchOperands: []mir.Operand{{Kind: mir.OperandLocal, Type: hir.TypeBool, Local: 1}}, Successors: []mir.BlockID{1, 2}, UnwindSuccessor: mir.NoBlock}},
			{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorCall, Callee: 2, DestinationLocal: mir.NoLocal, Successors: []mir.BlockID{3}, UnwindSuccessor: mir.NoBlock}},
			{ID: 2, Terminator: mir.Terminator{Kind: mir.TerminatorCall, Callee: 3, DestinationLocal: mir.NoLocal, Successors: []mir.BlockID{3}, UnwindSuccessor: mir.NoBlock}},
			{ID: 3, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock}},
		}
		functions[1].Blocks = []mir.Block{{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorTrap, UnwindSuccessor: mir.NoBlock}}}
		if returningLeaf {
			functions[2].Blocks[0].Terminator.Kind = mir.TerminatorReturn
		}
		if failures := mir.Verify(mir.Program{Functions: functions}); len(failures) != 0 {
			t.Fatal(failures[0])
		}
		got := nonReturningFunctionsForC(functions, nil)
		if got[1] != !returningLeaf || !got[2] || got[3] != !returningLeaf {
			t.Fatalf("returningLeaf=%t: incorrect branch propagation: %v", returningLeaf, got)
		}
	}
}

func BenchmarkNonReturningCallChain(b *testing.B) {
	for _, count := range []int{64, 256, 1024} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			functions := nonreturnChainFixture(count)
			if failures := mir.Verify(mir.Program{Functions: functions}); len(failures) != 0 {
				b.Fatal(failures[0])
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if got := nonReturningFunctionsForC(functions, nil); len(got) != count {
					b.Fatal("incomplete call-chain propagation")
				}
			}
		})
	}
}
