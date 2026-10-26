package mir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestVerifyPreservesEntrySnapshotAcrossBurnBackedge(t *testing.T) {
	coin := hir.ParseType("main::SnapshotCoin")
	program := Program{
		ResourceTypes: []ResourceType{{Type: coin, Fields: []ResourceField{{Name: "amount", Type: hir.TypeI32}}, DropKind: hir.ResourceDropEmpty, DropSymbol: hir.NoSymbol}},
		Functions: []Function{{
			Symbol: 1, Name: "main::loop", ResultType: hir.TypeVoid, Entry: 0,
			Parameters: []LocalID{1}, ParameterTypes: []hir.Type{coin},
			Resources: []ResourceLocal{{Symbol: 1, Type: coin, Parameter: true}},
			Blocks:    []Block{{ID: 0, Terminator: Terminator{Kind: TerminatorGoto, Successors: []BlockID{0}, UnwindSuccessor: NoBlock}}},
		}},
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("non-consuming loop rejected: %+v", failures)
	}
	program.Functions[0].Blocks[0].Operations = []Operation{{Kind: OperationBurn, Entity: "burn:coin", Symbol: 1, DestinationLocal: NoLocal, Type: coin}}
	for range 2 {
		failures := Verify(program)
		if len(failures) != 1 || failures[0].Code != "V1002" {
			t.Fatalf("consuming backedge must disagree with the entry snapshot: %+v", failures)
		}
	}
}

func BenchmarkVerifyStateSnapshots(b *testing.B) {
	for _, size := range []int{64, 256} {
		b.Run(fmt.Sprintf("blocks=%d/locals=%d", size, size), func(b *testing.B) {
			function := Function{Symbol: 1, Name: "main::snapshots", ResultType: hir.TypeVoid, Entry: 0,
				Parameters: make([]LocalID, size), ParameterTypes: make([]hir.Type, size), Blocks: make([]Block, size)}
			for i := range size {
				function.Parameters[i] = LocalID(i + 1)
				function.ParameterTypes[i] = hir.TypeI32
				term := Terminator{Kind: TerminatorGoto, Successors: []BlockID{BlockID(i + 1)}, UnwindSuccessor: NoBlock}
				if i == size-1 {
					term.Kind, term.Successors = TerminatorReturn, nil
				}
				function.Blocks[i] = Block{ID: BlockID(i), Terminator: term}
			}
			program := Program{Functions: []Function{function}}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if failures := Verify(program); len(failures) != 0 {
					b.Fatalf("Verify: %+v", failures)
				}
			}
		})
	}
}
