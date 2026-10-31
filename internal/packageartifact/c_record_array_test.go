package packageartifact

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestCRecordArrayContextRebind(t *testing.T) {
	context, identities, bindings := stableContextFixture(3, 41, 51)
	point := context.StructTypes[0].Type
	context.StructTypes = append(context.StructTypes, mir.StructType{
		Type: hir.ParseType("$p3::types::Packet"), Name: "Packet", ReprC: true, Span: hir.Span{Package: 3},
		Fields: []mir.StructField{{Name: "values", Type: hir.FixedArray(point, 3), Span: hir.Span{Package: 3}}},
	})
	descriptor, err := DescribeContext(context, identities, bindings)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeContextDescriptor(descriptor, contextArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeContextDescriptor(encoded, contextArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	newIdentities := make(map[hir.PackageID]packagegraph.CoordinateID)
	for _, id := range identities {
		newIdentities[19] = id
	}
	rebound, err := RebindContextDescriptor(decoded, newIdentities, map[FunctionID]hir.SymbolID{bindings[41]: 141}, 151)
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range rebound.StructTypes {
		if definition.Name == "Packet" {
			want := hir.FixedArray(hir.ParseType("$p19::types::Point"), 3)
			if !definition.ReprC || definition.Fields[0].Type != want {
				t.Fatalf("record array lost stable identity on rebind: %+v", definition)
			}
			return
		}
	}
	t.Fatal("rebound record array missing")
}
