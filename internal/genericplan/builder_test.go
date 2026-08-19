package genericplan

import (
	"slices"
	"testing"
)

func TestAbilityClosureAndConflict(t *testing.T) {
	builder, err := NewBuilder(DefaultCatalogLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := builder.AddType(TypeDescriptor{
		ID: "sdk::Number", Owner: "sdk", Abilities: []Ability{AbilityScalar, AbilityKey},
		Definition: DigestText("type", "Number/v1"),
	}); err != nil {
		t.Fatal(err)
	}
	catalog, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	value, found := catalog.Type("sdk::Number")
	if !found {
		t.Fatal("type not found")
	}
	expected := []Ability{AbilityCopy, AbilityDrop, AbilityInert, AbilityKey, AbilityScalar, AbilityStore}
	if !slices.Equal(value.Abilities, expected) {
		t.Fatalf("abilities = %v, want %v", value.Abilities, expected)
	}

	conflict, err := NewBuilder(DefaultCatalogLimits())
	if err != nil {
		t.Fatal(err)
	}
	err = conflict.AddType(TypeDescriptor{
		ID: "token::Asset", Owner: "token", Abilities: []Ability{AbilityResource, AbilityCopy},
		Definition: DigestText("type", "Asset/v1"),
	})
	if ErrorCode(err) != CodeInvalidAbilitySet {
		t.Fatalf("error = %v, want %s", err, CodeInvalidAbilitySet)
	}
}

func TestCatalogDigestIgnoresInsertionOrderAndCatalogIsImmutable(t *testing.T) {
	first := testCatalog(t, catalogVariant{addressProviderBody: "encode-address/v1"})
	second := testCatalog(t, catalogVariant{reverse: true, addressProviderBody: "encode-address/v1"})
	if first.Digest() != second.Digest() {
		t.Fatalf("catalog digest changed with insertion order: %s != %s", first.Digest(), second.Digest())
	}

	declaration, found := first.Declaration(writeDecl)
	if !found {
		t.Fatal("declaration not found")
	}
	declaration.Parameters[0].Abilities[0] = AbilityResource
	declaration.Dependencies[0].Arguments[0] = ConcreteType(coinType)
	again, _ := first.Declaration(writeDecl)
	if again.Parameters[0].Abilities[0] != AbilityKey {
		t.Fatal("catalog parameter abilities were mutated through a returned descriptor")
	}
	if again.Dependencies[0].Arguments[0].Parameter != "K" {
		t.Fatal("catalog dependency was mutated through a returned descriptor")
	}
}

func TestBuildRejectsOverlappingExactImplementations(t *testing.T) {
	builder, err := NewBuilder(DefaultCatalogLimits())
	if err != nil {
		t.Fatal(err)
	}
	mustAddType(t, builder, TypeDescriptor{ID: addressType, Owner: "sdk", Abilities: []Ability{AbilityScalar}, Definition: DigestText("type", "Address/v1")})
	mustAddCapability(t, builder, CapabilityDescriptor{ID: codecCap, Owner: "sdk", Arity: 1, Definition: DigestText("capability", "Codec/v1")})
	for _, provider := range []ProviderID{"sdk::codec_a", "sdk::codec_b"} {
		if err := builder.AddImplementation(Implementation{
			Capability: codecCap, Arguments: []TypeID{addressType}, Owner: "sdk", Provider: provider,
			ProviderDigest: DigestText("provider", string(provider)),
		}); err != nil {
			t.Fatal(err)
		}
	}
	_, err = builder.Build()
	if ErrorCode(err) != CodeDuplicateCatalogItem {
		t.Fatalf("error = %v, want %s", err, CodeDuplicateCatalogItem)
	}
}

func TestBuildEnforcesOrphanCoherence(t *testing.T) {
	builder, err := NewBuilder(DefaultCatalogLimits())
	if err != nil {
		t.Fatal(err)
	}
	mustAddType(t, builder, TypeDescriptor{ID: addressType, Owner: "sdk", Abilities: []Ability{AbilityScalar}, Definition: DigestText("type", "Address/v1")})
	mustAddCapability(t, builder, CapabilityDescriptor{ID: codecCap, Owner: "sdk", Arity: 1, Definition: DigestText("capability", "Codec/v1")})
	if err := builder.AddImplementation(Implementation{
		Capability: codecCap, Arguments: []TypeID{addressType}, Owner: "attacker", Provider: "attacker::codec",
		ProviderDigest: DigestText("provider", "attacker-codec"),
	}); err != nil {
		t.Fatal(err)
	}
	_, err = builder.Build()
	if ErrorCode(err) != CodeCoherenceViolation {
		t.Fatalf("error = %v, want %s", err, CodeCoherenceViolation)
	}
}

func TestBuilderRejectsMissingDefinitionIdentity(t *testing.T) {
	builder, err := NewBuilder(DefaultCatalogLimits())
	if err != nil {
		t.Fatal(err)
	}
	err = builder.AddType(TypeDescriptor{ID: "sdk::MissingDigest", Owner: "sdk", Abilities: []Ability{AbilityScalar}})
	if ErrorCode(err) != CodeInvalidDeclaration {
		t.Fatalf("error = %v, want %s", err, CodeInvalidDeclaration)
	}
}

func mustAddType(t *testing.T, builder *Builder, value TypeDescriptor) {
	t.Helper()
	if err := builder.AddType(value); err != nil {
		t.Fatal(err)
	}
}

func mustAddCapability(t *testing.T, builder *Builder, value CapabilityDescriptor) {
	t.Helper()
	if err := builder.AddCapability(value); err != nil {
		t.Fatal(err)
	}
}
