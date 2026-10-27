package atlasmap

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func TestRelationIDsMatchCanonicalFactsAndDetachResults(t *testing.T) {
	index := &QueryIndex{
		relations: []QueryRelation{
			{Source: "a", Target: &QueryTarget{Symbol: "a"}, Kind: RelationCalls},
			{Source: "a", Target: &QueryTarget{Symbol: "b"}, Kind: RelationCalls},
			{Source: "b", Target: &QueryTarget{Symbol: "a"}, Kind: RelationCalls},
			{Source: "a", Kind: RelationDrops},
			{Source: "c", Target: &QueryTarget{Symbol: "b"}, Kind: RelationCalls},
			{Source: "c", Target: &QueryTarget{Symbol: "a"}, Kind: RelationDrops},
		},
		outgoing: make(map[string][]int), incoming: make(map[string][]int),
	}
	slices.SortFunc(index.relations, compareQueryRelation)
	for position, relation := range index.relations {
		index.outgoing[relation.Source] = append(index.outgoing[relation.Source], position)
		if relation.Target != nil && relation.Target.Symbol != "" {
			index.incoming[relation.Target.Symbol] = append(index.incoming[relation.Target.Symbol], position)
		}
	}
	for _, symbol := range []string{"a", "b", "c", "empty"} {
		for _, direction := range []QueryDirection{DirectionOutgoing, DirectionIncoming, DirectionBoth} {
			for _, filter := range []map[RelationKind]struct{}{nil, {RelationCalls: {}}, {RelationCalls: {}, RelationDrops: {}}} {
				t.Run(fmt.Sprintf("%s/%s/kinds=%d", symbol, direction, len(filter)), func(t *testing.T) {
					want := make([]int, 0)
					// Scan canonical facts independently of the adjacency indexes.
					for position, relation := range index.relations {
						outgoing := (direction == DirectionOutgoing || direction == DirectionBoth) && relation.Source == symbol
						incoming := (direction == DirectionIncoming || direction == DirectionBoth) && relation.Target != nil && relation.Target.Symbol == symbol
						if _, included := filter[relation.Kind]; included && (outgoing || incoming) {
							want = append(want, position)
						}
					}
					beforeOutgoing, beforeIncoming := slices.Clone(index.outgoing[symbol]), slices.Clone(index.incoming[symbol])
					got := index.relationIDs(symbol, direction, filter)
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("relation IDs = %v, want %v", got, want)
					}
					if len(got) != 0 {
						got[0] = -1
					}
					if !slices.Equal(index.outgoing[symbol], beforeOutgoing) || !slices.Equal(index.incoming[symbol], beforeIncoming) {
						t.Fatal("query result aliases or changes an adjacency index")
					}
				})
			}
		}
	}
}

func BenchmarkShowOutgoingWithManyCallers(b *testing.B) {
	for _, width := range []int{64, 2048} {
		b.Run(fmt.Sprint(width), func(b *testing.B) {
			index, _ := wideQueryFixture(b, width)
			var symbol string
			for _, node := range index.nodes {
				if node.declaration.Key.Name == "left" {
					symbol = node.symbol.Visible
				}
			}
			if symbol == "" || len(index.incoming[symbol]) != width || len(index.outgoing[symbol]) != 1 {
				b.Fatal("fixture must have many callers and exactly one outgoing call")
			}
			request := wideQueryRequest(width, DirectionOutgoing, 1)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				result, err := index.Show(symbol, request)
				if err != nil || len(result.Relations) != 1 {
					b.Fatalf("Show outgoing relations = %d, error = %v", len(result.Relations), err)
				}
			}
		})
	}
}
