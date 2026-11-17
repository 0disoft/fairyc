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
			for range random.Intn(4) {
				function.Blocks[i].Terminator.Successors = append(function.Blocks[i].Terminator.Successors, BlockID(random.Intn(8)))
			}
		}
		retention := verificationLocalRetention(&function)
		for root := range function.Blocks {
			seen := map[BlockID]bool{}
			queue := []BlockID{BlockID(root)}
			for len(queue) > 0 {
				id := queue[0]
				queue = queue[1:]
				if seen[id] {
					continue
				}
				seen[id] = true
				if retention.floor[root] > int(id) {
					t.Fatalf("root %d discarded reachable block %d: bounds %v", root, id, retention.floor)
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
