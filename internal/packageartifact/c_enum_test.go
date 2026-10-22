package packageartifact

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestCEnumContextIdentityAndRebind(t *testing.T) {
	context, identities, bindings := stableContextFixture(3, 41, 51)
	context.EnumTypes[0].CBase = "c.uint"
	for i := range context.EnumTypes[0].Variants {
		context.EnumTypes[0].Variants[i].Fields = nil
	}
	first, err := DescribeContext(context, identities, bindings)
	if err != nil {
		t.Fatal(err)
	}
	firstID, err := ComputeContextID(first, contextArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeContextDescriptor(first, contextArtifactLimit)
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
	if rebound.EnumTypes[0].CBase != "c.uint" {
		t.Fatal("C enum base lost during rebind")
	}
	context.EnumTypes[0].CBase = "c.ushort"
	second, err := DescribeContext(context, identities, bindings)
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := ComputeContextID(second, contextArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if firstID == secondID {
		t.Fatal("enum base did not invalidate context identity")
	}
}
