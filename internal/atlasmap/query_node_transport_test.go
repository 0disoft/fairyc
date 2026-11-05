package atlasmap

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestQueryNodesSurviveMapTransport(t *testing.T) {
	f := localCallGraphFixture(t)
	gw, gr := loadGenericPlanWorkspace(t, true)
	_, gp, err := PrepareWorkspaceV2(gw, gr, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	ge, err := CurrentWorkspaceExpectationsV2(gw, gr, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	aw, ar := loadAggregateMapWorkspace(t, true)
	_, ap, err := PrepareWorkspaceV3(aw, ar, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	ae, err := CurrentWorkspaceExpectationsV3(aw, ar, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name   string
		p      Partition
		e      LoadExpectation
		encode func(Partition, uint64) ([]byte, error)
		decode func([]byte, uint64) (Partition, error)
		index  func(Partition, LoadExpectation) (*QueryIndex, error)
	}{
		{"ordinary", f.partition, f.expectation, Encode, Decode, NewQueryIndex},
		{"generic", gp[0], ge[gp[0].Header().PackageInstance], EncodeV2, DecodeV2, NewQueryIndexV2},
		{"aggregate-cleanup", ap[0], ae[ap[0].Header().PackageInstance], EncodeV3, DecodeV3, NewQueryIndexV3},
	} {
		t.Run(c.name, func(t *testing.T) {
			encoded, err := c.encode(c.p, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			restored, err := c.decode(encoded, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			original, err := c.index(c.p, c.e)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := c.index(restored, c.e)
			if err != nil {
				t.Fatal(err)
			}
			request := RelationRequest{Direction: DirectionBoth, Relations: AllRelationKinds(), Budget: generousQueryBudget(4)}
			for _, node := range original.nodes {
				for _, around := range []bool{false, true} {
					query := func(index *QueryIndex, budget QueryBudget) ([]byte, uint64, error) {
						r := request
						r.Budget = budget
						if around {
							view, err := index.Around(node.symbol.Full, r)
							if err != nil {
								return nil, 0, err
							}
							data, err := json.Marshal(view)
							return data, view.Usage.SelectionBytes, err
						}
						view, err := index.Show(node.symbol.Full, r)
						if err != nil {
							return nil, 0, err
						}
						data, err := json.Marshal(view)
						return data, view.Usage.SelectionBytes, err
					}
					want, _, err := query(original, request.Budget)
					if err != nil {
						t.Fatal(err)
					}
					got, _, err := query(decoded, request.Budget)
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(want, got) {
						t.Fatalf("unchanged map changed %s around=%t node or selection bytes:\noriginal=%s\nstored=%s", node.symbol.Full, around, want, got)
					}
					limit := request.Budget
					limit.MaxRecords = 1
					limit.MaxRelations = 0
					limit.MaxDepth = 0
					_, mandatory, err := query(original, limit)
					if err != nil {
						t.Fatal(err)
					}
					for _, maximum := range []uint64{mandatory - 1, mandatory, mandatory + 1} {
						limit.MaxSelectionBytes = maximum
						for _, index := range []*QueryIndex{original, decoded} {
							_, _, err := query(index, limit)
							if (err != nil) != (maximum < mandatory) {
								t.Fatalf("around=%t boundary %d/%d: %v", around, maximum, mandatory, err)
							}
						}
					}
				}
			}
			after, err := c.encode(c.p, 4<<20)
			if err != nil || !bytes.Equal(encoded, after) {
				t.Fatalf("query mutated source partition: %v", err)
			}
		})
	}
}
