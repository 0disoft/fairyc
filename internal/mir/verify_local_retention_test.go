package mir

import (
	"math/rand"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestVerificationRetentionBoundsIncludeEveryReachableBlock(t *testing.T) {
	random := rand.New(rand.NewSource(23))
	for range 100 {
		function := Function{Blocks: make([]Block, 8)}
		for i := range function.Blocks {
			function.Blocks[i].ID = BlockID(i)
			function.Blocks[i].Operations = []Operation{{Symbol: hir.SymbolID(i + 1), DestinationLocal: NoLocal}}
			for range random.Intn(4) {
				function.Blocks[i].Terminator.Successors = append(function.Blocks[i].Terminator.Successors, BlockID(random.Intn(8)))
			}
		}
		retention := verificationLocalRetention(&function)
		for root := range function.Blocks {
			live := make(map[hir.SymbolID]bool)
			for i := range function.Blocks {
				live[hir.SymbolID(i+1)] = true
			}
			retention.retire(live, []BlockID{BlockID(root)}, nil)
			seen := map[BlockID]bool{}
			queue := []BlockID{BlockID(root)}
			for len(queue) > 0 {
				id := queue[0]
				queue = queue[1:]
				if seen[id] {
					continue
				}
				seen[id] = true
				if !live[hir.SymbolID(id+1)] {
					t.Fatalf("root %d discarded a mention in reachable block %d: bounds %v", root, id, retention.floor)
				}
				queue = append(queue, function.Blocks[id].Terminator.Successors...)
			}
		}
	}
}

func TestVerificationRetentionKeepsResourcesAndFutureMentions(t *testing.T) {
	function := Function{Blocks: []Block{{ID: 0}, {ID: 1}}}
	live := make(map[hir.SymbolID]bool)
	for symbol := hir.SymbolID(1); symbol <= 1024; symbol++ {
		function.Blocks[0].Operations = append(function.Blocks[0].Operations, Operation{Symbol: symbol, DestinationLocal: NoLocal})
		live[symbol] = true
	}
	function.Blocks[0].Terminator.Successors = []BlockID{1}
	function.Blocks[1].Terminator.ReturnValue = Operand{Kind: OperandLocal, Local: 1024}
	resources := map[hir.SymbolID]ResourceLocal{1: {Symbol: 1}}
	live[1] = false // Consumed resources must remain in join-consistency checks.
	retention := verificationLocalRetention(&function)
	retention.retire(live, []BlockID{1}, resources)
	if len(live) != 2 || live[1] || !live[1024] {
		t.Fatalf("retained state = %v; want consumed resource and future return", live)
	}
	// A backedge makes earlier mentions reachable again, including scalar writes.
	function.Blocks[1].Terminator.Successors = []BlockID{0}
	retention = verificationLocalRetention(&function)
	live[2] = true
	retention.retire(live, []BlockID{1}, resources)
	if !live[2] {
		t.Fatal("backedge lost an earlier scalar mention")
	}
}

func TestVerificationRetentionWithEarlyAllocatedContinuation(t *testing.T) {
	function := Function{Entry: 0, Blocks: []Block{
		{ID: 0, Terminator: Terminator{Successors: []BlockID{2}}},
		{ID: 1, Terminator: Terminator{ReturnValue: Operand{Kind: OperandLocal, Local: 1024}}},
		{ID: 2, Terminator: Terminator{Successors: []BlockID{3, 4}}},
		{ID: 3, Terminator: Terminator{Successors: []BlockID{1}}},
		{ID: 4, Terminator: Terminator{Successors: []BlockID{1}}},
	}}
	live := make(map[hir.SymbolID]bool)
	for symbol := hir.SymbolID(1); symbol <= 1024; symbol++ {
		function.Blocks[2].Operations = append(function.Blocks[2].Operations, Operation{Symbol: symbol, DestinationLocal: NoLocal})
		live[symbol] = true
	}
	resources := map[hir.SymbolID]ResourceLocal{1: {Symbol: 1}}
	retention := verificationLocalRetention(&function)
	retention.retire(live, []BlockID{3, 4}, resources)
	if len(live) != 2 || !live[1] || !live[1024] {
		t.Fatalf("early continuation retained %d entries; want resource and future return", len(live))
	}
}
