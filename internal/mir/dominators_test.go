package mir

import (
	"fmt"
	"maps"
	"math/rand"
	"slices"
	"testing"
)

func TestBlockDominatorsMatchesReference(t *testing.T) {
	graphs := []struct {
		name  string
		entry BlockID
		edges map[BlockID][]BlockID
	}{
		{"diamond", 10, map[BlockID][]BlockID{10: {20, 30}, 20: {40}, 30: {40}, 40: nil}},
		{"loop", 20, map[BlockID][]BlockID{10: {30}, 20: {10}, 30: {10, 40}, 40: nil}},
		{"unreachable", 10, map[BlockID][]BlockID{10: {20}, 20: nil, 30: {40}, 40: {30}, 50: nil}},
		{"unknown-successor", 10, map[BlockID][]BlockID{10: {20, 999}, 20: {20, 30}, 30: nil}},
	}
	for _, graph := range graphs {
		t.Run(graph.name, func(t *testing.T) {
			function := Function{Entry: graph.entry}
			for _, id := range slices.Sorted(maps.Keys(graph.edges)) {
				function.Blocks = append(function.Blocks, Block{ID: id, Terminator: Terminator{Successors: graph.edges[id]}})
			}
			assertDominatorsMatchReference(t, &function, false)
		})
	}
	// Exercise sparse IDs, missing entries, self edges and 64-bit word boundaries.
	random := rand.New(rand.NewSource(42))
	for _, count := range []int{0, 1, 2, 7, 63, 64, 65, 129} {
		for sample := range 8 {
			t.Run(fmt.Sprintf("random_%d_%d", count, sample), func(t *testing.T) {
				function := Function{Entry: NoBlock, Blocks: make([]Block, count)}
				if count > 0 && sample%3 != 0 {
					function.Entry = BlockID(random.Intn(count)*3 + 5)
				}
				for index := range function.Blocks {
					block := &function.Blocks[index]
					block.ID = BlockID(index*3 + 5)
					for range random.Intn(4) {
						block.Terminator.Successors = append(block.Terminator.Successors, BlockID(random.Intn(count)*3+5))
					}
				}
				assertDominatorsMatchReference(t, &function, sample%2 == 0)
			})
		}
	}
}

func assertDominatorsMatchReference(t *testing.T, function *Function, unknownPredecessor bool) {
	t.Helper()
	predecessors := make(map[BlockID][]BlockID)
	for _, block := range function.Blocks {
		for _, successor := range block.Terminator.Successors {
			predecessors[successor] = append(predecessors[successor], block.ID)
		}
	}
	if unknownPredecessor && len(function.Blocks) > 0 {
		id := function.Blocks[len(function.Blocks)-1].ID
		predecessors[id] = append(predecessors[id], NoBlock)
	}
	actual := blockDominators(function, predecessors)
	expected := referenceBlockDominators(function, predecessors)
	for _, target := range function.Blocks {
		for _, candidate := range function.Blocks {
			_, want := expected[target.ID][candidate.ID]
			if got := actual.contains(target.ID, candidate.ID); got != want {
				t.Fatalf("target %d candidate %d: got %t, want %t", target.ID, candidate.ID, got, want)
			}
		}
		if actual.contains(target.ID, NoBlock) || actual.contains(NoBlock, target.ID) {
			t.Fatal("unknown block must not alias the first dense index")
		}
	}
	if actual.contains(NoBlock, NoBlock) {
		t.Fatal("unknown blocks must not dominate each other")
	}
}

// Keep the previous map-based fixed point as an independent test oracle.
func referenceBlockDominators(function *Function, predecessors map[BlockID][]BlockID) map[BlockID]map[BlockID]struct{} {
	all := make(map[BlockID]struct{})
	for _, block := range function.Blocks {
		all[block.ID] = struct{}{}
	}
	dominators := make(map[BlockID]map[BlockID]struct{})
	for _, block := range function.Blocks {
		if block.ID == function.Entry {
			dominators[block.ID] = map[BlockID]struct{}{block.ID: {}}
		} else {
			dominators[block.ID] = maps.Clone(all)
		}
	}
	for changed := true; changed; {
		changed = false
		for _, block := range function.Blocks {
			if block.ID == function.Entry {
				continue
			}
			next := make(map[BlockID]struct{})
			if incoming := predecessors[block.ID]; len(incoming) > 0 {
				for candidate := range dominators[incoming[0]] {
					if slices.ContainsFunc(incoming[1:], func(predecessor BlockID) bool {
						_, found := dominators[predecessor][candidate]
						return !found
					}) {
						continue
					}
					next[candidate] = struct{}{}
				}
			}
			next[block.ID] = struct{}{}
			if !maps.Equal(dominators[block.ID], next) {
				dominators[block.ID] = next
				changed = true
			}
		}
	}
	return dominators
}

func BenchmarkTrapCheckDominators(b *testing.B) {
	for _, branches := range []int{32, 128, 341} {
		function := branchOnlyFunction(branches)
		b.Run(fmt.Sprintf("blocks_%d", len(function.Blocks)), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				blocks, _ := trapCheckGraph(&function)
				if len(blocks) != len(function.Blocks) {
					b.Fatal("missing block index")
				}
			}
		})
	}
}

func dominatorChain(count int, reverse bool) Function {
	function := Function{Blocks: make([]Block, count)}
	if reverse {
		function.Entry = BlockID(count - 1)
	}
	for index := range count {
		block := Block{ID: BlockID(index), Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock}}
		next := index + 1
		if reverse {
			next = index - 1
		}
		if next >= 0 && next < count {
			block.Terminator = Terminator{Kind: TerminatorGoto, Successors: []BlockID{BlockID(next)}, UnwindSuccessor: NoBlock}
		}
		function.Blocks[index] = block
	}
	return function
}

func TestBlockDominatorsReverseChain(t *testing.T) {
	function := dominatorChain(65, true)
	assertDominatorsMatchReference(t, &function, false)
}

func TestDominatorBlockOrderPredecessorsFirst(t *testing.T) {
	function := dominatorChain(1024, true)
	indices := make(map[BlockID]int, len(function.Blocks))
	predecessors := make(map[BlockID][]BlockID, len(function.Blocks))
	for index, block := range function.Blocks {
		indices[block.ID] = index
		for _, successor := range block.Terminator.Successors {
			predecessors[successor] = append(predecessors[successor], block.ID)
		}
	}
	order := dominatorBlockOrder(&function, indices, predecessors)
	if len(order) != len(indices) {
		t.Fatalf("order has %d blocks, want %d", len(order), len(indices))
	}
	for index, id := range order {
		if want := BlockID(len(order) - 1 - index); id != want {
			t.Fatalf("position %d: got %d, want predecessor %d first", index, id, want)
		}
	}
}

func BenchmarkTrapCheckDominatorsChain(b *testing.B) {
	for _, reverse := range []bool{false, true} {
		for _, count := range []int{128, 512, 1024} {
			b.Run(fmt.Sprintf("reverse_%t/blocks_%d", reverse, count), func(b *testing.B) {
				function := dominatorChain(count, reverse)
				b.ReportAllocs()
				for b.Loop() {
					_, sets := trapCheckGraph(&function)
					if !sets.contains(BlockID(count-1)-function.Entry, function.Entry) {
						b.Fatal("entry must dominate the last executed block")
					}
				}
			})
		}
	}
}
