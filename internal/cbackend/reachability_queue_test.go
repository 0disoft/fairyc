package cbackend

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestReachabilitySharedMixedEdgesAndCycle(t *testing.T) {
	resource := hir.ParseType("main::QueueResource")
	program := mir.Program{
		ResourceTypes: []mir.ResourceType{{Type: resource, DropSymbol: 3}},
		Functions: []mir.Function{
			{Symbol: 1, Name: "main::entry", Resources: []mir.ResourceLocal{{Symbol: 10, Type: resource}}, Blocks: []mir.Block{{
				Operations: []mir.Operation{{Kind: mir.OperationCCallback, Symbol: 2}, {Kind: mir.OperationDrop, Symbol: 10}},
				Terminator: mir.Terminator{Kind: mir.TerminatorCall, Callee: 2},
			}}},
			{Symbol: 2, Name: "main::shared", Blocks: []mir.Block{{
				Operations: []mir.Operation{{Kind: mir.OperationCCallback, Symbol: 3}},
				Terminator: mir.Terminator{Kind: mir.TerminatorCall, Callee: 99, ForeignABI: "C"},
			}}},
			{Symbol: 3, Name: "main::drop", Blocks: []mir.Block{{Terminator: mir.Terminator{Kind: mir.TerminatorCall, Callee: 1}}}},
			{Symbol: 4, Name: "main::unused", Blocks: []mir.Block{{Terminator: mir.Terminator{Kind: mir.TerminatorCall, Callee: 999}}}},
		},
	}
	bySymbol, entry, err := indexFunctions(program.Functions, "main::entry")
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		reachable, foreign, err := reachableFunctionIndices(program, program.Functions, bySymbol, entry)
		if err != nil || len(reachable) != 3 || len(foreign) != 1 {
			t.Fatalf("mixed graph: reachable=%v foreign=%v err=%v", reachable, foreign, err)
		}
		for _, index := range []int{0, 1, 2} {
			if _, found := reachable[index]; !found {
				t.Fatalf("missing reachable function %d", index)
			}
		}
		if _, found := foreign[99]; !found {
			t.Fatal("missing foreign dependency")
		}
	}
	program.Functions[2].Blocks[0].Operations = []mir.Operation{{Kind: mir.OperationCCallback, Symbol: 999}}
	if _, _, err := reachableFunctionIndices(program, program.Functions, bySymbol, entry); err == nil || !strings.Contains(err.Error(), "missing callback target") {
		t.Fatalf("missing callback in shared function: %v", err)
	}
}

func BenchmarkDenseCallReachability(b *testing.B) {
	for _, count := range []int{64, 256} {
		b.Run(fmt.Sprintf("functions=%d", count), func(b *testing.B) {
			program := mir.Program{Functions: make([]mir.Function, count)}
			for i := range count {
				function := &program.Functions[i]
				*function = mir.Function{Symbol: hir.SymbolID(i + 1), Name: fmt.Sprintf("main::f%03d", i), ResultType: hir.TypeVoid, Entry: 0}
				for target := count - 1; target > i; target-- {
					id := mir.BlockID(len(function.Blocks))
					function.Blocks = append(function.Blocks, mir.Block{ID: id, Terminator: mir.Terminator{
						Kind: mir.TerminatorCall, Callee: hir.SymbolID(target + 1), DestinationLocal: mir.NoLocal,
						Successors: []mir.BlockID{id + 1}, UnwindSuccessor: mir.NoBlock,
					}})
				}
				id := mir.BlockID(len(function.Blocks))
				function.Blocks = append(function.Blocks, mir.Block{ID: id, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock}})
			}
			if failures := mir.Verify(program); len(failures) != 0 {
				b.Fatalf("benchmark MIR: %+v", failures)
			}
			bySymbol, entry, err := indexFunctions(program.Functions, "main::f000")
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				reachable, _, err := reachableFunctionIndices(program, program.Functions, bySymbol, entry)
				if err != nil || len(reachable) != count {
					b.Fatalf("reachable=%d, err=%v", len(reachable), err)
				}
			}
		})
	}
}
