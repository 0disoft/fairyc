package genericplan

import "fmt"

func ExampleCatalog_Plan() {
	builder, _ := NewBuilder(DefaultCatalogLimits())
	_ = builder.AddType(TypeDescriptor{
		ID: "sdk::Address", Owner: "sdk", Abilities: []Ability{AbilityScalar, AbilityKey},
		Definition: DigestText("type", "Address/v1"),
	})
	_ = builder.AddType(TypeDescriptor{
		ID: "sdk::U64", Owner: "sdk", Abilities: []Ability{AbilityScalar},
		Definition: DigestText("type", "U64/v1"),
	})
	_ = builder.AddCapability(CapabilityDescriptor{
		ID: "sdk::Codec", Owner: "sdk", Arity: 1,
		Definition: DigestText("capability", "Codec<T>/v1"),
	})
	for _, implementation := range []Implementation{
		{
			Capability: "sdk::Codec", Arguments: []TypeID{"sdk::Address"}, Owner: "sdk",
			Provider: "sdk::encode_address", ProviderDigest: DigestText("provider", "address/v1"),
		},
		{
			Capability: "sdk::Codec", Arguments: []TypeID{"sdk::U64"}, Owner: "sdk",
			Provider: "sdk::encode_u64", ProviderDigest: DigestText("provider", "u64/v1"),
		},
	} {
		_ = builder.AddImplementation(implementation)
	}
	_ = builder.AddDeclaration(Declaration{
		ID: "sdk::Encode", Owner: "sdk",
		Parameters:  []TypeParameter{{Name: "T", Abilities: []Ability{AbilityStore}}},
		Constraints: []CapabilityConstraint{{Capability: "sdk::Codec", Arguments: []TypeExpression{ParameterType("T")}}},
		Definition:  DigestText("declaration", "Encode<T>/v1"),
	})
	_ = builder.AddDeclaration(Declaration{
		ID: "sdk::StorageWrite", Owner: "sdk",
		Parameters: []TypeParameter{
			{Name: "K", Abilities: []Ability{AbilityKey}},
			{Name: "V", Abilities: []Ability{AbilityStore}},
		},
		Constraints: []CapabilityConstraint{
			{Capability: "sdk::Codec", Arguments: []TypeExpression{ParameterType("K")}},
			{Capability: "sdk::Codec", Arguments: []TypeExpression{ParameterType("V")}},
		},
		Dependencies: []Dependency{
			{Declaration: "sdk::Encode", Arguments: []TypeExpression{ParameterType("K")}},
			{Declaration: "sdk::Encode", Arguments: []TypeExpression{ParameterType("V")}},
		},
		Definition: DigestText("declaration", "StorageWrite<K,V>/v1"),
	})
	catalog, _ := builder.Build()
	plan, _ := catalog.Plan([]Request{{
		Declaration: "sdk::StorageWrite",
		Arguments:   []TypeID{"sdk::Address", "sdk::U64"},
	}}, DefaultPlanLimits())

	for _, specialization := range plan.Specializations {
		fmt.Printf("%s", specialization.Declaration)
		for _, binding := range specialization.Bindings {
			fmt.Printf(" %s=%s", binding.Parameter, binding.Type)
		}
		for _, constraint := range specialization.Constraints {
			fmt.Printf(" via=%s", constraint.Provider)
		}
		fmt.Println()
	}

	// Output:
	// sdk::Encode T=sdk::Address via=sdk::encode_address
	// sdk::Encode T=sdk::U64 via=sdk::encode_u64
	// sdk::StorageWrite K=sdk::Address V=sdk::U64 via=sdk::encode_address via=sdk::encode_u64
}
