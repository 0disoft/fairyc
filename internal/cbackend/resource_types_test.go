package cbackend

import (
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestNestedResourceDefinitionOrder(t *testing.T) {
	definitions := []mir.ResourceType{
		{Type: "main::Outer", Fields: []mir.ResourceField{{Name: "child", Type: "main::ZLeaf"}}},
		{Type: "main::Another", Fields: []mir.ResourceField{{Name: "child", Type: "main::ZLeaf"}}},
		{Type: "main::ZLeaf", Fields: []mir.ResourceField{{Name: "value", Type: hir.TypeI32}}},
	}
	for _, reverse := range []bool{false, true} {
		if reverse {
			slices.Reverse(definitions)
		}
		ordered, err := orderResourceDefinitions(definitions)
		if err != nil || len(ordered) != 3 || ordered[0].Type != "main::ZLeaf" || ordered[1].Type != "main::Another" || ordered[2].Type != "main::Outer" {
			t.Fatalf("dependency order: %+v %v", ordered, err)
		}
	}
}

func TestNestedResourceDefinitionRejectsCycleAndDuplicate(t *testing.T) {
	for _, definitions := range [][]mir.ResourceType{
		{{Type: "A", Fields: []mir.ResourceField{{Name: "self", Type: "A"}}}},
		{{Type: "A", Fields: []mir.ResourceField{{Name: "b", Type: "B"}}}, {Type: "B", Fields: []mir.ResourceField{{Name: "a", Type: "A"}}}},
		{{Type: "A"}, {Type: "A"}},
	} {
		if _, err := orderResourceDefinitions(definitions); err == nil {
			t.Fatal("invalid resource graph accepted")
		}
	}
}

func TestNestedResourceDropSymbols(t *testing.T) {
	definitions := map[hir.Type]mir.ResourceType{
		"Box": {Type: "Box", DropSymbol: 11, Fields: []mir.ResourceField{
			{Name: "first", Type: "Leaf"}, {Name: "last", Type: "Leaf"}}},
		"Leaf": {Type: "Leaf", DropSymbol: 22},
	}
	for _, typ := range []hir.Type{"Box", hir.Option("Box"), hir.Result("Box", hir.TypeI32)} {
		if actual := dischargeDropSymbols(typ, definitions); !slices.Equal(actual, []hir.SymbolID{11, 22}) {
			t.Fatalf("cleanup reachability for %s: %v", typ, actual)
		}
	}
}
