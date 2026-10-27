package atlasmap

import (
	"fmt"
	"maps"
	"testing"
)

func TestUnselectedExpansionPreservesDirectionFilterAndSeenState(t *testing.T) {
	index := &QueryIndex{
		nodes: []indexedQueryNode{{symbol: Symbol{Visible: "a"}}, {symbol: Symbol{Visible: "b"}}, {symbol: Symbol{Visible: "c"}}},
		relations: []QueryRelation{
			{Source: "a", Target: &QueryTarget{Symbol: "a"}, Kind: RelationCalls},
			{Source: "a", Target: &QueryTarget{Symbol: "b"}, Kind: RelationCalls},
			{Source: "b", Target: &QueryTarget{Symbol: "a"}, Kind: RelationCalls},
			{Source: "a", Kind: RelationDrops}, // Terminal facts still count as unselected relations.
			{Source: "c", Target: &QueryTarget{Symbol: "b"}, Kind: RelationCalls},
			{Source: "c", Target: &QueryTarget{Symbol: "a"}, Kind: RelationDrops},
		},
		outgoing: map[string][]int{"a": {0, 1, 3}, "b": {2}, "c": {4, 5}},
		incoming: map[string][]int{"a": {0, 2, 5}, "b": {1, 4}},
	}
	for _, test := range []struct {
		name      string
		frontier  []int
		direction QueryDirection
		kind      RelationKind
		seen      []int
		want      bool
	}{
		{"empty", nil, DirectionBoth, RelationCalls, nil, false},
		{"outgoing", []int{0}, DirectionOutgoing, RelationCalls, nil, true},
		{"incoming", []int{0}, DirectionIncoming, RelationCalls, nil, true},
		{"terminal", []int{0}, DirectionOutgoing, RelationDrops, nil, true},
		{"filtered", []int{0}, DirectionBoth, RelationUsesUnsafe, nil, false},
		{"outgoing-seen", []int{0}, DirectionOutgoing, RelationCalls, []int{0, 1}, false},
		{"incoming-seen", []int{0}, DirectionIncoming, RelationCalls, []int{0, 2}, false},
		{"both-incoming-remains", []int{0}, DirectionBoth, RelationCalls, []int{0, 1}, true},
		{"both-outgoing-remains", []int{0}, DirectionBoth, RelationCalls, []int{0, 2}, true},
		{"both-seen", []int{0}, DirectionBoth, RelationCalls, []int{0, 1, 2}, false},
		{"later-frontier-node", []int{0, 2}, DirectionBoth, RelationCalls, []int{0, 1, 2}, true},
		{"all-frontier-seen", []int{0, 2}, DirectionBoth, RelationCalls, []int{0, 1, 2, 4}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			seen := make(map[int]struct{})
			for _, relation := range test.seen {
				seen[relation] = struct{}{}
			}
			before := maps.Clone(seen)
			filter := map[RelationKind]struct{}{test.kind: {}}
			if got := index.hasUnselectedExpansion(test.frontier, test.direction, filter, seen); got != test.want {
				t.Fatalf("hasUnselectedExpansion=%t, want %t", got, test.want)
			}
			if !maps.Equal(seen, before) {
				t.Fatal("depth-limit probe changed selected relations")
			}
		})
	}
}

func BenchmarkAroundDepthBoundary(b *testing.B) {
	for _, width := range []int{64, 512} {
		b.Run(fmt.Sprintf("width=%d", width), func(b *testing.B) {
			index, root := wideQueryFixture(b, width)
			request := wideQueryRequest(width, DirectionOutgoing, 0)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				result, err := index.Around(root, request)
				if err != nil || len(result.Nodes) != 1 || len(result.Relations) != 0 || !hasOmission(result.Omissions, OmissionDepthLimit) {
					b.Fatalf("incorrect depth boundary: %+v, %v", result, err)
				}
			}
		})
	}
}
