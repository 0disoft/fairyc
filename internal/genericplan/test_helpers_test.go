package genericplan

import "testing"

const (
	addressType TypeID        = "sdk::Address"
	u64Type     TypeID        = "sdk::U64"
	coinType    TypeID        = "token::Coin"
	opaqueType  TypeID        = "sdk::Opaque"
	codecCap    CapabilityID  = "sdk::Codec"
	encodeDecl  DeclarationID = "sdk::Encode"
	writeDecl   DeclarationID = "sdk::StorageWrite"
)

type catalogVariant struct {
	reverse             bool
	addressProviderBody string
	includeOpaqueCodec  bool
}

func testCatalog(t *testing.T, variant catalogVariant) *Catalog {
	t.Helper()
	builder, err := NewBuilder(DefaultCatalogLimits())
	if err != nil {
		t.Fatal(err)
	}

	types := []TypeDescriptor{
		{ID: addressType, Owner: "sdk", Abilities: []Ability{AbilityScalar, AbilityKey}, Definition: DigestText("type", "Address/v1")},
		{ID: u64Type, Owner: "sdk", Abilities: []Ability{AbilityScalar}, Definition: DigestText("type", "U64/v1")},
		{ID: coinType, Owner: "token", Abilities: []Ability{AbilityResource, AbilityStore}, Definition: DigestText("type", "Coin/v1")},
		{ID: opaqueType, Owner: "sdk", Abilities: []Ability{AbilityInert, AbilityStore}, Definition: DigestText("type", "Opaque/v1")},
	}
	capabilities := []CapabilityDescriptor{
		{ID: codecCap, Owner: "sdk", Arity: 1, Definition: DigestText("capability", "Codec<T>/v1")},
	}
	implementations := []Implementation{
		{
			Capability: codecCap, Arguments: []TypeID{addressType}, Owner: "sdk",
			Provider: "sdk::encode_address", ProviderDigest: DigestText("provider", variant.addressProviderBody),
		},
		{
			Capability: codecCap, Arguments: []TypeID{u64Type}, Owner: "sdk",
			Provider: "sdk::encode_u64", ProviderDigest: DigestText("provider", "encode-u64/v1"),
		},
		{
			Capability: codecCap, Arguments: []TypeID{coinType}, Owner: "token",
			Provider: "token::encode_coin", ProviderDigest: DigestText("provider", "encode-coin/v1"),
		},
	}
	if variant.includeOpaqueCodec {
		implementations = append(implementations, Implementation{
			Capability: codecCap, Arguments: []TypeID{opaqueType}, Owner: "sdk",
			Provider: "sdk::encode_opaque", ProviderDigest: DigestText("provider", "encode-opaque/v1"),
		})
	}
	declarations := []Declaration{
		{
			ID: encodeDecl, Owner: "sdk",
			Parameters:  []TypeParameter{{Name: "T", Abilities: []Ability{AbilityStore}}},
			Constraints: []CapabilityConstraint{{Capability: codecCap, Arguments: []TypeExpression{ParameterType("T")}}},
			Definition:  DigestText("declaration", "Encode<T>/v1"),
		},
		{
			ID: writeDecl, Owner: "sdk",
			Parameters: []TypeParameter{
				{Name: "K", Abilities: []Ability{AbilityKey}},
				{Name: "V", Abilities: []Ability{AbilityStore}},
			},
			Constraints: []CapabilityConstraint{
				{Capability: codecCap, Arguments: []TypeExpression{ParameterType("K")}},
				{Capability: codecCap, Arguments: []TypeExpression{ParameterType("V")}},
			},
			Dependencies: []Dependency{
				{Declaration: encodeDecl, Arguments: []TypeExpression{ParameterType("K")}},
				{Declaration: encodeDecl, Arguments: []TypeExpression{ParameterType("V")}},
			},
			Definition: DigestText("declaration", "StorageWrite<K,V>/v1"),
		},
	}

	if variant.reverse {
		reverse(types)
		reverse(capabilities)
		reverse(implementations)
		for index := range declarations {
			reverse(declarations[index].Constraints)
			reverse(declarations[index].Dependencies)
		}
		reverse(declarations)
	}
	for _, value := range types {
		if err := builder.AddType(value); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range capabilities {
		if err := builder.AddCapability(value); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range implementations {
		if err := builder.AddImplementation(value); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range declarations {
		if err := builder.AddDeclaration(value); err != nil {
			t.Fatal(err)
		}
	}
	catalog, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func reverse[T any](values []T) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func specializationBy(t *testing.T, plan Plan, declaration DeclarationID, arguments ...TypeID) Specialization {
	t.Helper()
	for _, specialization := range plan.Specializations {
		if specialization.Declaration != declaration || len(specialization.Bindings) != len(arguments) {
			continue
		}
		matches := true
		for index, argument := range arguments {
			matches = matches && specialization.Bindings[index].Type == argument
		}
		if matches {
			return specialization
		}
	}
	t.Fatalf("specialization %s%v not found", declaration, arguments)
	return Specialization{}
}
