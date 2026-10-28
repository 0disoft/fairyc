package atlasmap

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func BenchmarkCatalogSymbolsExactIdentity(b *testing.B) {
	for _, width := range []int{64, 2048} {
		b.Run(fmt.Sprint(width), func(b *testing.B) {
			index, _ := wideQueryFixture(b, width)
			for _, name := range []string{"branch_0", "absent"} {
				b.Run(name, func(b *testing.B) {
					request := SymbolCatalogRequest{
						Module: "main", Name: name, Kinds: AllDeclarationKinds(),
						Budget: SymbolCatalogBudget{MaxSymbols: 1, MaxSelectionBytes: 64 << 10},
					}
					want := 0
					if name == "branch_0" {
						want = 1
					}
					b.ReportAllocs()
					for b.Loop() {
						result, err := index.CatalogSymbols(request)
						if err != nil || !result.Complete || result.Total != want || len(result.Symbols) != want {
							b.Fatalf("exact lookup: total=%d, error=%v", result.Total, err)
						}
					}
				})
			}
		})
	}
}

func TestCatalogExactCandidatesMatchCanonicalScan(t *testing.T) {
	// Full-ID order can differ from declaration-kind order. Use multiple exact
	// records to check ordering and cursors independently of map iteration.
	index := &QueryIndex{header: Header{PackageInstance: "current"}, byKey: map[semanticsnapshot.DeclarationKey]int{}}
	for _, entry := range []struct {
		module, name string
		kind         semanticsnapshot.DeclarationKind
	}{
		{"main", "target", semanticsnapshot.DeclarationStruct},
		{"other", "target", semanticsnapshot.DeclarationFunction},
		{"main", "other", semanticsnapshot.DeclarationFunction},
		{"main", "target", semanticsnapshot.DeclarationFunction},
		{"main", "target", semanticsnapshot.DeclarationEnum},
	} {
		key := semanticsnapshot.DeclarationKey{PackageInstance: "current", Module: entry.module, Name: entry.name, Kind: entry.kind}
		index.byKey[key] = len(index.nodes)
		index.nodes = append(index.nodes, indexedQueryNode{declaration: semanticsnapshot.Declaration{Key: key}})
	}
	for _, module := range []string{"", "main", "other", "absent"} {
		for _, name := range []string{"", "target", "Target", "absent"} {
			for _, kinds := range []map[semanticsnapshot.DeclarationKind]struct{}{
				{semanticsnapshot.DeclarationFunction: {}},
				{semanticsnapshot.DeclarationFunction: {}, semanticsnapshot.DeclarationStruct: {}, semanticsnapshot.DeclarationEnum: {}},
			} {
				want := []int{}
				for position, node := range index.nodes {
					key := node.declaration.Key
					if _, selected := kinds[key.Kind]; selected && (module == "" || key.Module == module) && (name == "" || key.Name == name) {
						want = append(want, position)
					}
				}
				got := index.catalogCandidates(module, name, kinds)
				if got.count != len(want) {
					t.Fatalf("%q/%q: count=%d, want %v", module, name, got.count, want)
				}
				for position, expected := range want {
					if got.at(position) != expected {
						t.Fatalf("%q/%q: position %d differs from canonical scan", module, name, position)
					}
				}
				for node := range index.nodes {
					position, found := got.cursor(node)
					wantPosition, wantFound := slices.BinarySearch(want, node)
					if found != wantFound || found && position != wantPosition {
						t.Fatalf("%q/%q: cursor %d differs from canonical scan", module, name, node)
					}
				}
			}
		}
	}
}

func TestCatalogExactIdentityKeepsCursorScopeAndBudgets(t *testing.T) {
	fixture := localCallGraphFixture(t)
	index, err := NewQueryIndex(fixture.partition, fixture.expectation)
	if err != nil {
		t.Fatal(err)
	}
	request := SymbolCatalogRequest{
		Module: "main", Name: "leaf", Kinds: AllDeclarationKinds(),
		Budget: SymbolCatalogBudget{MaxSymbols: 1, MaxSelectionBytes: 64 << 10},
	}
	first, err := index.CatalogSymbols(request)
	if err != nil || !first.Complete || first.Total != 1 || len(first.Symbols) != 1 {
		t.Fatalf("exact identity: %+v, %v", first, err)
	}
	request.After = first.Symbols[0].Symbol.Full
	last, err := index.CatalogSymbols(request)
	if err != nil || !last.Complete || last.Total != 1 || len(last.Symbols) != 0 {
		t.Fatalf("last cursor: %+v, %v", last, err)
	}
	var outside string
	for _, node := range index.nodes {
		if node.declaration.Key.Module != request.Module && node.declaration.Key.Name == request.Name {
			outside = node.symbol.Full
		}
	}
	if outside == "" {
		t.Fatal("fixture has no same-name symbol in another module")
	}
	request.After = outside
	if _, err := index.CatalogSymbols(request); err == nil {
		t.Fatal("cursor outside exact identity was accepted")
	}
	request.After = ""
	request.Budget.MaxSelectionBytes = 1
	if _, err := index.CatalogSymbols(request); !errors.Is(err, ErrQueryBudgetTooSmall) {
		t.Fatalf("exact identity bypassed the selection-byte budget: %v", err)
	}
}
