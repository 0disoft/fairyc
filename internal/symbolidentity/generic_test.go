package symbolidentity_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

func genericDeclaration() semanticsnapshot.Declaration {
	return semanticsnapshot.Declaration{
		Key:            semanticsnapshot.DeclarationKey{PackageInstance: "instance", Module: "main", Kind: semanticsnapshot.DeclarationFunction, Name: "choose"},
		TypeParameters: []string{"T", "U"},
		Type:           semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeParameter, Name: "0"},
		Parameters: []semanticsnapshot.Parameter{
			{Name: "first", Type: semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeParameter, Name: "0"}},
			{Name: "second", Type: semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeParameter, Name: "1"}},
		},
	}
}

func genericSymbols(value semanticsnapshot.Declaration) ([]symbolidentity.Symbol, error) {
	return symbolidentity.FromDeclarations([]semanticsnapshot.Declaration{value}, map[string]string{"instance": "example.generic"}, map[string]string{"coordinate": "example.generic"})
}

func TestGenericIdentityUsesBinderPositionsNotNames(t *testing.T) {
	base, err := genericSymbols(genericDeclaration())
	if err != nil {
		t.Fatal(err)
	}
	renamed := genericDeclaration()
	renamed.TypeParameters = []string{"Value", "Other"}
	renamed.Parameters[0].Name = "renamed"
	renamed.Public = true
	other, err := genericSymbols(renamed)
	if err != nil || other[0].ID != base[0].ID {
		t.Fatalf("source names changed identity: %v", err)
	}
	for _, change := range []func(*semanticsnapshot.Declaration){
		func(v *semanticsnapshot.Declaration) { v.TypeParameters = append(v.TypeParameters, "Unused") },
		func(v *semanticsnapshot.Declaration) { v.Type.Name = "1" },
		func(v *semanticsnapshot.Declaration) {
			v.Type = semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeBuiltin, Name: "i32"}
		},
	} {
		value := genericDeclaration()
		change(&value)
		changed, err := genericSymbols(value)
		if err != nil || changed[0].ID == base[0].ID {
			t.Fatalf("semantic generic shape did not change identity: %v", err)
		}
	}
}

func TestGenericIdentityRejectsUnboundAndMalformedParameters(t *testing.T) {
	for _, change := range []func(*semanticsnapshot.Declaration){
		func(v *semanticsnapshot.Declaration) { v.TypeParameters = nil },
		func(v *semanticsnapshot.Declaration) { v.TypeParameters = []string{"T", "T"} },
		func(v *semanticsnapshot.Declaration) { v.TypeParameters = []string{"", "U"} },
		func(v *semanticsnapshot.Declaration) { v.Type.Name = "2" },
		func(v *semanticsnapshot.Declaration) { v.Type.Name = "00" },
		func(v *semanticsnapshot.Declaration) { v.Type.Name = "T" },
		func(v *semanticsnapshot.Declaration) { v.Type.Mutable = true },
		func(v *semanticsnapshot.Declaration) { v.Type.PackageCoordinate = "coordinate" },
		func(v *semanticsnapshot.Declaration) { v.Type.Length = 1 },
		func(v *semanticsnapshot.Declaration) {
			v.Type.Arguments = []semanticsnapshot.TypeRef{{Kind: semanticsnapshot.TypeBuiltin, Name: "i32"}}
		},
		func(v *semanticsnapshot.Declaration) { v.Foreign = true },
		func(v *semanticsnapshot.Declaration) { v.Key.Kind = semanticsnapshot.DeclarationStruct },
	} {
		value := genericDeclaration()
		change(&value)
		if _, err := genericSymbols(value); err == nil {
			t.Fatalf("invalid generic identity accepted: %+v", value)
		}
	}
}

func TestGenericIdentityTracksNestedBinderReferences(t *testing.T) {
	value := genericDeclaration()
	value.Type = semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeOption, Arguments: []semanticsnapshot.TypeRef{{Kind: semanticsnapshot.TypeParameter, Name: "0"}}}
	first, err := genericSymbols(value)
	if err != nil {
		t.Fatal(err)
	}
	value.Type.Arguments[0].Name = "1"
	second, err := genericSymbols(value)
	if err != nil || first[0].ID == second[0].ID {
		t.Fatalf("nested binder identity: %v", err)
	}
	value.Type.Arguments[0].Name = "2"
	if _, err := genericSymbols(value); err == nil {
		t.Fatal("nested out-of-range binder accepted")
	}
}
