package packageartifact

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestCScalarPointerContextRebind(t *testing.T) {
	for _, leaf := range hir.CScalarTypes() {
		for _, typ := range []hir.Type{hir.RawPointer(leaf, false), hir.RawPointer(leaf, true), hir.RawPointer(hir.RawPointer(leaf, true), true)} {
			context, identities, bindings := stableContextFixture(3, 41, 51)
			context.StructTypes[0].Fields[0].Type = typ
			context.ForeignFunctions[0].ParameterTypes = []hir.Type{typ}
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
			rebound, err := RebindContextDescriptor(decoded, identities, map[FunctionID]hir.SymbolID{bindings[41]: 141}, 151)
			if err != nil {
				t.Fatal(err)
			}
			if rebound.StructTypes[0].Fields[0].Type != typ || rebound.ForeignFunctions[0].ParameterTypes[0] != typ {
				t.Fatalf("%s lost pointee/depth/mutability", typ)
			}
		}
	}
}
