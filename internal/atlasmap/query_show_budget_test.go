package atlasmap

import (
	"fmt"
	"reflect"
	"testing"
)

func TestShowBudgetPreservesPrefixAndExactOmissions(t *testing.T) {
	index, root := wideQueryFixture(t, 8)
	for _, direction := range []QueryDirection{DirectionOutgoing, DirectionIncoming, DirectionBoth} {
		t.Run(string(direction), func(t *testing.T) {
			request := wideQueryRequest(8, direction, 1)
			full, err := index.Show(root, request)
			if err != nil {
				t.Fatal(err)
			}
			subjectBytes := full.Usage.SelectionBytes
			for _, relation := range full.Relations {
				cost, err := selectionRecordCost(relation)
				if err != nil {
					t.Fatal(err)
				}
				subjectBytes -= cost
			}
			for _, test := range []struct {
				name, code string
				limit      int
			}{
				{"zero-relations", OmissionBudgetRelations, 0},
				{"one-relation", OmissionBudgetRelations, 1},
				{"three-relations", OmissionBudgetRelations, 3},
				{"records", OmissionBudgetRecords, 0},
				{"bytes", OmissionBudgetBytes, 0},
			} {
				t.Run(test.name, func(t *testing.T) {
					bounded := request
					switch test.code {
					case OmissionBudgetRelations:
						bounded.Budget.MaxRelations = test.limit
					case OmissionBudgetRecords:
						bounded.Budget.MaxRecords = 1
					case OmissionBudgetBytes:
						bounded.Budget.MaxSelectionBytes = subjectBytes
					}
					got, err := index.Show(root, bounded)
					selected := min(test.limit, len(full.Relations))
					if err != nil || len(got.Relations) != selected || got.Usage.Relations != selected {
						t.Fatalf("relations=%d, usage=%+v, error=%v", len(got.Relations), got.Usage, err)
					}
					for i, relation := range got.Relations {
						if !reflect.DeepEqual(relation, full.Relations[i]) {
							t.Fatalf("relation %d differs from the complete selection", i)
						}
					}
					missing, found := len(full.Relations)-selected, false
					for _, omission := range got.Omissions {
						if omission.Code == test.code {
							if found || missing == 0 || !omission.CountKnown || omission.Count != missing {
								t.Fatalf("incorrect exact omission: %+v, want %d", omission, missing)
							}
							found = true
						}
					}
					if found != (missing > 0) {
						t.Fatalf("omission present=%t, missing=%d", found, missing)
					}
				})
			}
		})
	}
}

func BenchmarkShowSmallRelationBudget(b *testing.B) {
	for _, width := range []int{64, 2048} {
		b.Run(fmt.Sprintf("width=%d", width), func(b *testing.B) {
			index, root := wideQueryFixture(b, width)
			request := wideQueryRequest(width, DirectionOutgoing, 1)
			request.Budget.MaxRelations = 1
			b.ReportAllocs()
			for b.Loop() {
				got, err := index.Show(root, request)
				if err != nil || len(got.Relations) != 1 || !hasOmission(got.Omissions, OmissionBudgetRelations) {
					b.Fatalf("relations=%d, error=%v", len(got.Relations), err)
				}
			}
		})
	}
}
