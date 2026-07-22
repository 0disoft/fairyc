package packageartifact

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStableContextRebindsWithoutProcessLocalTypeOrForeignState(t *testing.T) {
	t.Parallel()
	leftContext, leftIdentities, leftBindings := stableContextFixture(3, 41, 51)
	descriptor, err := DescribeContext(leftContext, leftIdentities, leftBindings)
	if err != nil {
		t.Fatal(err)
	}
	coordinate := leftIdentities[3]
	rightIdentities := map[hir.PackageID]packagegraph.CoordinateID{9: coordinate}
	dropID := leftBindings[41]
	rebound, err := RebindContextDescriptor(descriptor, rightIdentities, map[FunctionID]hir.SymbolID{dropID: 801}, 901)
	if err != nil {
		t.Fatal(err)
	}
	if rebound.StructTypes[0].Span.Package != 9 || rebound.StructTypes[0].Type != "$p9::types::Point" {
		t.Fatalf("rebound struct = %+v", rebound.StructTypes[0])
	}
	if rebound.ResourceTypes[0].DropSymbol != 801 || rebound.ForeignFunctions[0].Symbol != 901 {
		t.Fatalf("rebound symbols: resource=%d foreign=%d", rebound.ResourceTypes[0].DropSymbol, rebound.ForeignFunctions[0].Symbol)
	}
	roundTrip, err := DescribeContext(rebound, rightIdentities, map[hir.SymbolID]FunctionID{801: dropID})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(roundTrip, descriptor) {
		t.Fatalf("rebound context differs:\nwant: %+v\ngot:  %+v", descriptor, roundTrip)
	}
}

func TestStableContextRebindRejectsMissingAmbiguousAndCollidingBindings(t *testing.T) {
	t.Parallel()
	context, identities, bindings := stableContextFixture(3, 41, 51)
	descriptor, err := DescribeContext(context, identities, bindings)
	if err != nil {
		t.Fatal(err)
	}
	coordinate := identities[3]
	dropID := bindings[41]
	if _, err := RebindContextDescriptor(descriptor, map[hir.PackageID]packagegraph.CoordinateID{9: coordinate}, nil, 901); err == nil || !strings.Contains(err.Error(), "drop target") {
		t.Fatalf("missing drop binding error = %v", err)
	}
	if _, err := RebindContextDescriptor(descriptor, map[hir.PackageID]packagegraph.CoordinateID{9: coordinate, 10: coordinate}, map[FunctionID]hir.SymbolID{dropID: 801}, 901); err == nil || !strings.Contains(err.Error(), "multiple local packages") {
		t.Fatalf("ambiguous package error = %v", err)
	}
	if _, err := RebindContextDescriptor(descriptor, map[hir.PackageID]packagegraph.CoordinateID{9: coordinate}, map[FunctionID]hir.SymbolID{dropID: 901}, 901); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("symbol collision error = %v", err)
	}
}
