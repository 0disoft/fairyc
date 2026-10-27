package atlasmap

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestCatalogPaginationPreservesFilteredOrderAndIsolation(t *testing.T) {
	fixture := localCallGraphFixture(t)
	index, err := NewQueryIndex(fixture.partition, fixture.expectation)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, module, declaration string
		kinds                     []semanticsnapshot.DeclarationKind
	}{
		{"all", "", "", AllDeclarationKinds()},
		{"functions", "", "", []semanticsnapshot.DeclarationKind{semanticsnapshot.DeclarationFunction}},
		{"empty-kind", "", "", []semanticsnapshot.DeclarationKind{semanticsnapshot.DeclarationEnum}},
		{"module", "main", "", AllDeclarationKinds()},
		{"name", "", "leaf", AllDeclarationKinds()},
		{"module-name", "other", "leaf", AllDeclarationKinds()},
		{"empty-name", "", "absent", AllDeclarationKinds()},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := SymbolCatalogRequest{Module: test.module, Name: test.declaration, Kinds: test.kinds, Budget: SymbolCatalogBudget{MaxSymbols: 1, MaxSelectionBytes: 8 << 20}}
			// Independent scan/sort oracle mirrors the previous selection policy.
			want := make([]SymbolCatalogEntry, 0)
			for position, node := range index.nodes {
				key := node.declaration.Key
				if (test.module == "" || key.Module == test.module) && (test.declaration == "" || key.Name == test.declaration) && slices.Contains(test.kinds, key.Kind) {
					want = append(want, index.catalogEntry(position))
				}
			}
			slices.SortFunc(want, func(left, right SymbolCatalogEntry) int { return strings.Compare(left.Symbol.Full, right.Symbol.Full) })
			selected := make([]SymbolCatalogEntry, 0)
			for {
				page, err := index.CatalogSymbols(request)
				if err != nil || page.Total != len(want) {
					t.Fatalf("total=%d, want %d, error=%v", page.Total, len(want), err)
				}
				selected = append(selected, page.Symbols...)
				if page.Usage.Symbols != len(page.Symbols) {
					t.Fatalf("usage=%+v, symbols=%d", page.Usage, len(page.Symbols))
				}
				if len(page.Symbols) > 0 {
					page.Symbols[0].Symbol.Full = "caller mutation"
				}
				if page.Complete {
					break
				}
				if page.NextAfter == "" || page.NextAfter == request.After || len(selected) > len(want) {
					t.Fatal("pagination did not advance within its result set")
				}
				if len(page.Omissions) != 1 || page.Omissions[0].Code != OmissionBudgetSymbols || !page.Omissions[0].CountKnown || page.Omissions[0].Count != len(want)-len(selected) {
					t.Fatalf("incorrect remaining count: %+v", page.Omissions)
				}
				request.After = page.NextAfter
			}
			if !reflect.DeepEqual(selected, want) {
				t.Fatalf("pages differ from scan/sort: got %+v, want %+v", selected, want)
			}
			for _, node := range index.nodes {
				request.After = node.symbol.Full
				_, err := index.CatalogSymbols(request)
				included := slices.ContainsFunc(want, func(entry SymbolCatalogEntry) bool { return entry.Symbol.Full == request.After })
				if (err == nil) != included {
					t.Fatalf("cursor %s: included=%t, error=%v", request.After, included, err)
				}
			}
		})
	}
}

func BenchmarkCatalogSymbolsPagination(b *testing.B) {
	// wideQueryFixture adds main, left, right and tail to each branch width.
	// Every operation traverses the complete catalog in 64-symbol pages.
	for _, width := range []int{64, 512, 2048} {
		b.Run(fmt.Sprint(width), func(b *testing.B) {
			index, _ := wideQueryFixture(b, width)
			request := SymbolCatalogRequest{Kinds: AllDeclarationKinds(), Budget: SymbolCatalogBudget{MaxSymbols: 64, MaxSelectionBytes: 8 << 20}}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				request.After = ""
				selected := 0
				for {
					page, err := index.CatalogSymbols(request)
					if err != nil {
						b.Fatal(err)
					}
					selected += len(page.Symbols)
					if page.Complete {
						break
					}
					if page.NextAfter == "" || page.NextAfter == request.After {
						b.Fatal("pagination did not advance")
					}
					request.After = page.NextAfter
				}
				if selected != len(index.nodes) {
					b.Fatalf("selected %d of %d symbols", selected, len(index.nodes))
				}
			}
			b.ReportMetric(float64(len(index.nodes)), "symbols/op")
		})
	}
}
