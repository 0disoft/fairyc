package semantic

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func BenchmarkNonDroppableSharedOwners(b *testing.B) {
	for _, depth := range []int{8, 12, 16} {
		b.Run(fmt.Sprint(depth), func(b *testing.B) {
			a, root, _ := resourceOwnerGraphFixture(depth)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				owners := a.nonDroppableResourceOwners(root, make(map[hir.Type]struct{}))
				if len(owners) == 0 {
					b.Fatal("shared no-drop owner was missed")
				}
			}
		})
	}
}

func TestNonDroppableSharedOwnersAreUniqueAndOrdered(t *testing.T) {
	a, root, asset := resourceOwnerGraphFixture(8)
	other := hir.ParseType("main::Other")
	otherSymbol := &symbol{typ: other, resource: &ast.Resource{}}
	a.resourceTypes[other] = &resourceInfo{symbol: otherSymbol, abilities: resourceAbilityDeclaration{valid: true, explicit: true}}
	a.resourceTypes[root].orderedFields = append(a.resourceTypes[root].orderedFields, resourceField{typ: other})
	for _, typ := range []hir.Type{root, hir.Option(root), hir.Result(root, root)} {
		owners := a.nonDroppableResourceOwners(typ, make(map[hir.Type]struct{}))
		if len(owners) != 2 || owners[0] != asset || owners[1] != otherSymbol {
			t.Fatalf("shared owners must retain first-discovery order without duplicates: %d owners", len(owners))
		}
	}
}

func TestNonDroppableOwnersHandleCyclesAndFreshSearches(t *testing.T) {
	a, root, asset := resourceOwnerGraphFixture(4)
	a.resourceTypes[root].orderedFields = append([]resourceField{{typ: root}}, a.resourceTypes[root].orderedFields...)
	owners := a.nonDroppableResourceOwners(root, make(map[hir.Type]struct{}))
	if len(owners) != 1 || owners[0] != asset {
		t.Fatal("cycle hid or duplicated a reachable no-drop owner")
	}
	a.resourceTypes[asset.typ].abilities.abilities = resourceAbilityDrop
	if owners := a.nonDroppableResourceOwners(root, make(map[hir.Type]struct{})); len(owners) != 0 {
		t.Fatal("a fresh search reused the previous no-drop result")
	}
}

func resourceOwnerGraphFixture(depth int) (*analyzer, hir.Type, *symbol) {
	a := &analyzer{resourceTypes: make(map[hir.Type]*resourceInfo, depth+1)}
	types := make([]hir.Type, depth+1)
	for i := range types {
		types[i] = hir.ParseType(fmt.Sprintf("main::Node_%02d", i))
	}
	for i, typ := range types {
		owner := &symbol{typ: typ, resource: &ast.Resource{}}
		info := &resourceInfo{symbol: owner, abilities: resourceAbilityDeclaration{valid: true, explicit: true}}
		if i < depth {
			info.abilities.abilities = resourceAbilityDrop
			info.orderedFields = []resourceField{{typ: types[i+1]}, {typ: types[i+1]}}
		}
		a.resourceTypes[typ] = info
	}
	return a, types[0], a.resourceTypes[types[depth]].symbol
}
