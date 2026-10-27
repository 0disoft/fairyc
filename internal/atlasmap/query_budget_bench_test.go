package atlasmap

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func TestFrontierRelationsMatchCanonicalFactsAfterEarlyStop(t *testing.T) {
	index, _ := wideQueryFixture(t, 4)
	all := make([]int, len(index.nodes))
	for i := range all {
		all[i] = i
	}
	copyLists := func(values map[string][]int) map[string][]int {
		result := make(map[string][]int, len(values))
		for key, list := range values {
			result[key] = slices.Clone(list)
		}
		return result
	}
	outgoing, incoming := copyLists(index.outgoing), copyLists(index.incoming)
	for _, frontier := range [][]int{nil, {0}, {0, 1, 0}, all} {
		active := make(map[string]bool)
		for _, position := range frontier {
			active[index.nodes[position].symbol.Visible] = true
		}
		for _, direction := range []QueryDirection{DirectionOutgoing, DirectionIncoming, DirectionBoth} {
			for _, filter := range []map[RelationKind]struct{}{nil, {RelationCalls: {}}, {RelationDrops: {}}} {
				seen := map[int]struct{}{0: {}, 3: {}}
				var want []int
				// Derive membership from canonical facts, independently of adjacency lists.
				for position, relation := range index.relations {
					if _, found := seen[position]; found {
						continue
					}
					if _, included := filter[relation.Kind]; !included {
						continue
					}
					out := direction != DirectionIncoming && active[relation.Source]
					in := direction != DirectionOutgoing && relation.Target != nil && active[relation.Target.Symbol]
					if out || in {
						want = append(want, position)
					}
				}
				sequence := index.frontierRelations(frontier, direction, filter, seen)
				for first := range sequence {
					if len(want) == 0 || first != want[0] {
						t.Fatalf("first relation = %d, want %v", first, want)
					}
					break
				}
				if got := slices.Collect(sequence); !slices.Equal(got, want) {
					t.Fatalf("frontier=%v, direction=%s: got %v, want %v", frontier, direction, got, want)
				}
				if len(seen) != 2 {
					t.Fatal("iteration changed the selected relation set")
				}
			}
		}
	}
	if !reflect.DeepEqual(outgoing, index.outgoing) || !reflect.DeepEqual(incoming, index.incoming) {
		t.Fatal("iteration changed the immutable adjacency indexes")
	}
}

func TestAroundRelationBudgetKeepsCanonicalPrefix(t *testing.T) {
	index, root := wideQueryFixture(t, 8)
	for _, direction := range []QueryDirection{DirectionOutgoing, DirectionIncoming, DirectionBoth} {
		t.Run(string(direction), func(t *testing.T) {
			request := wideQueryRequest(8, direction, 4)
			full, err := index.Around(root, request)
			if err != nil {
				t.Fatal(err)
			}
			for _, limit := range []int{0, 1, 3} {
				request.Budget.MaxRelations = limit
				got, err := index.Around(root, request)
				if err != nil || len(got.Relations) != limit || !hasOmission(got.Omissions, OmissionBudgetRelations) {
					t.Fatalf("limit=%d: relations=%d, omissions=%+v, error=%v", limit, len(got.Relations), got.Omissions, err)
				}
				for i, relation := range got.Relations {
					if !reflect.DeepEqual(relation, full.Relations[i]) {
						t.Fatalf("limit=%d: relation %d differs from the complete traversal", limit, i)
					}
				}
			}
		})
	}
}

func BenchmarkAroundSmallRelationBudget(b *testing.B) {
	for _, width := range []int{64, 2048} {
		b.Run(fmt.Sprintf("width=%d", width), func(b *testing.B) {
			index, root := wideQueryFixture(b, width)
			request := wideQueryRequest(width, DirectionOutgoing, 4)
			request.Budget.MaxRelations = 1
			b.ReportAllocs()
			for b.Loop() {
				result, err := index.Around(root, request)
				if err != nil || len(result.Relations) != 1 || !hasOmission(result.Omissions, OmissionBudgetRelations) {
					b.Fatalf("relations=%d, error=%v", len(result.Relations), err)
				}
			}
		})
	}
}
