package packageartifact

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestIntegerTransportContextRebind(t *testing.T) {
	for _, typ := range []hir.Type{hir.TypeI8, hir.TypeI16, hir.TypeU16, hir.TypeU64, hir.TypeISize} {
		context, identities, bindings := stableContextFixture(3, 41, 51)
		context.StructTypes[0].Fields[0].Type = typ
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
		rebound, err := RebindContextDescriptor(decoded, identities, map[FunctionID]hir.SymbolID{bindings[41]: 41}, 51)
		if err != nil {
			t.Fatal(err)
		}
		if rebound.StructTypes[0].Fields[0].Type != typ {
			t.Fatalf("%s field lost during rebind", typ)
		}
	}
}
