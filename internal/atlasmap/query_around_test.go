package atlasmap

import (
	"errors"
	"maps"
	"reflect"
	"slices"
	"testing"
)

func TestAroundAdjacentEndpointsAreOrderedUniqueAndReadOnly(t *testing.T) {
	index := &QueryIndex{
		nodes:     []indexedQueryNode{{symbol: Symbol{Full: "z", Visible: "source"}}, {symbol: Symbol{Full: "a", Visible: "target"}}},
		byVisible: map[string]int{"source": 0, "target": 1},
	}
	for _, tc := range []struct {
		name      string
		direction QueryDirection
		frontier  map[string]struct{}
		target    *QueryTarget
		want      []int
	}{
		{"outgoing", DirectionOutgoing, map[string]struct{}{"source": {}}, &QueryTarget{Symbol: "target"}, []int{1}},
		{"incoming", DirectionIncoming, map[string]struct{}{"target": {}}, &QueryTarget{Symbol: "target"}, []int{0}},
		{"both-in-canonical-order", DirectionBoth, map[string]struct{}{"source": {}, "target": {}}, &QueryTarget{Symbol: "target"}, []int{1, 0}},
		{"self-edge-once", DirectionBoth, map[string]struct{}{"source": {}}, &QueryTarget{Symbol: "source"}, []int{0}},
		{"unselected", DirectionBoth, nil, &QueryTarget{Symbol: "target"}, nil},
		{"terminal", DirectionBoth, map[string]struct{}{"source": {}}, nil, nil},
		{"unjoined-target", DirectionOutgoing, map[string]struct{}{"source": {}}, &QueryTarget{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := maps.Clone(tc.frontier)
			relation := QueryRelation{Source: "source", Target: tc.target}
			positions, count := index.adjacentNodes(tc.frontier, relation, tc.direction)
			if !slices.Equal(positions[:count], tc.want) {
				t.Fatalf("neighbors = %v; want %v", positions[:count], tc.want)
			}
			if !maps.Equal(before, tc.frontier) {
				t.Fatal("neighbor selection mutated frontier membership")
			}
		})
	}
}

func TestAroundBudgetStopsKeepSelectedRelationAndExactBytes(t *testing.T) {
	index, root := wideQueryFixture(t, 2)
	request := wideQueryRequest(2, DirectionOutgoing, 4)
	before := slices.Clone(request.Relations)
	full, err := index.Around(root, request)
	if err != nil {
		t.Fatal(err)
	}
	request.Budget.MaxRecords = 1 // The mandatory context/root record is already selected.
	mandatory, err := index.Around(root, request)
	if err != nil || mandatory.Usage.Records != 1 || len(mandatory.Relations) != 0 {
		t.Fatalf("mandatory record: %+v, %v", mandatory, err)
	}
	request.Budget.MaxRecords = 2 // The relation fits, its new target node does not.
	partial, err := index.Around(root, request)
	if err != nil || len(partial.Nodes) != 1 || len(partial.Relations) != 1 || partial.Usage.Depth != 0 {
		t.Fatalf("node budget boundary: %+v, %v", partial, err)
	}
	if !reflect.DeepEqual(partial.Relations[0], full.Relations[0]) || !hasOmission(partial.Omissions, OmissionBudgetRecords) || hasOmission(partial.Omissions, OmissionDepthLimit) {
		t.Fatalf("selected relation or omission precedence changed: %+v", partial)
	}
	request.Budget.MaxRecords = 100
	for _, tc := range []struct {
		bytes         uint64
		wantError     bool
		wantRelations int
	}{
		{mandatory.Usage.SelectionBytes - 1, true, 0},
		{mandatory.Usage.SelectionBytes, false, 0},
		{mandatory.Usage.SelectionBytes + 1, false, 0},
		{partial.Usage.SelectionBytes - 1, false, 0},
		{partial.Usage.SelectionBytes, false, 1},
		{partial.Usage.SelectionBytes + 1, false, 1},
	} {
		request.Budget.MaxSelectionBytes = tc.bytes
		got, err := index.Around(root, request)
		if tc.wantError {
			if !errors.Is(err, ErrQueryBudgetTooSmall) {
				t.Fatalf("bytes=%d: error = %v", tc.bytes, err)
			}
			continue
		}
		if err != nil || len(got.Relations) != tc.wantRelations || got.Usage.SelectionBytes > tc.bytes {
			t.Fatalf("bytes=%d: %+v, %v", tc.bytes, got, err)
		}
	}
	if !slices.Equal(request.Relations, before) {
		t.Fatal("Around mutated the request filter")
	}
	restored, err := index.Around(root, wideQueryRequest(2, DirectionOutgoing, 4))
	if err != nil || !reflect.DeepEqual(restored, full) {
		t.Fatal("bounded queries mutated the index")
	}
}

func BenchmarkAroundAdjacentEndpoints(b *testing.B) {
	index := &QueryIndex{
		nodes:     []indexedQueryNode{{symbol: Symbol{Full: "z"}}, {symbol: Symbol{Full: "a"}}},
		byVisible: map[string]int{"source": 0, "target": 1},
	}
	frontier := map[string]struct{}{"source": {}, "target": {}}
	relation := QueryRelation{Source: "source", Target: &QueryTarget{Symbol: "target"}}
	b.ReportAllocs()
	for b.Loop() {
		positions, count := index.adjacentNodes(frontier, relation, DirectionBoth)
		if count != 2 || positions[0] != 1 || positions[1] != 0 {
			b.Fatal("incorrect canonical endpoints")
		}
	}
}
