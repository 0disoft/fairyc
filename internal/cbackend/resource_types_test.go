package cbackend

import (
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestNestedResourceDefinitionOrder(t *testing.T) {
	definitions := []mir.ResourceType{
		{Type: hir.ParseType("main::Outer"), Fields: []mir.ResourceField{{Name: "child", Type: hir.ParseType("main::ZLeaf")}}},
		{Type: hir.ParseType("main::Another"), Fields: []mir.ResourceField{{Name: "child", Type: hir.ParseType("main::ZLeaf")}}},
		{Type: hir.ParseType("main::ZLeaf"), Fields: []mir.ResourceField{{Name: "value", Type: hir.TypeI32}}},
	}
	for _, reverse := range []bool{false, true} {
		if reverse {
			slices.Reverse(definitions)
		}
		ordered, err := orderResourceDefinitions(definitions)
		if err != nil || len(ordered) != 3 || ordered[0].Type != hir.ParseType("main::ZLeaf") || ordered[1].Type != hir.ParseType("main::Another") || ordered[2].Type != hir.ParseType("main::Outer") {
			t.Fatalf("dependency order: %+v %v", ordered, err)
		}
	}
}

func TestNestedResourceDefinitionRejectsCycleAndDuplicate(t *testing.T) {
	for _, definitions := range [][]mir.ResourceType{
		{{Type: hir.ParseType("A"), Fields: []mir.ResourceField{{Name: "self", Type: hir.ParseType("A")}}}},
		{{Type: hir.ParseType("A"), Fields: []mir.ResourceField{{Name: "b", Type: hir.ParseType("B")}}}, {Type: hir.ParseType("B"), Fields: []mir.ResourceField{{Name: "a", Type: hir.ParseType("A")}}}},
		{{Type: hir.ParseType("A")}, {Type: hir.ParseType("A")}},
	} {
		if _, err := orderResourceDefinitions(definitions); err == nil {
			t.Fatal("invalid resource graph accepted")
		}
	}
}

func TestNestedResourceDropSymbols(t *testing.T) {
	definitions := map[hir.Type]mir.ResourceType{
		hir.ParseType("Box"): {Type: hir.ParseType("Box"), DropSymbol: 11, Fields: []mir.ResourceField{
			{Name: "first", Type: hir.ParseType("Leaf")}, {Name: "last", Type: hir.ParseType("Leaf")}}},
		hir.ParseType("Leaf"): {Type: hir.ParseType("Leaf"), DropSymbol: 22},
	}
	for _, typ := range []hir.Type{hir.ParseType("Box"), hir.Option(hir.ParseType("Box")), hir.Result(hir.ParseType("Box"), hir.TypeI32)} {
		if actual := dischargeDropSymbols(typ, definitions); !slices.Equal(actual, []hir.SymbolID{11, 22}) {
			t.Fatalf("cleanup reachability for %s: %v", typ, actual)
		}
	}
}
