package genericplan

import (
	"cmp"
	"slices"
)

// Ability is one compiler-known static property usable by the first generic
// constraint kernel.
type Ability string

const (
	AbilityCopy     Ability = "copy"
	AbilityDrop     Ability = "drop"
	AbilityStore    Ability = "store"
	AbilityKey      Ability = "key"
	AbilityInert    Ability = "inert"
	AbilityScalar   Ability = "scalar"
	AbilityResource Ability = "resource"
)

var knownAbilities = []Ability{
	AbilityCopy,
	AbilityDrop,
	AbilityInert,
	AbilityKey,
	AbilityResource,
	AbilityScalar,
	AbilityStore,
}

// KnownAbilities returns the closed vocabulary in canonical lexical order.
func KnownAbilities() []Ability {
	return slices.Clone(knownAbilities)
}

func canonicalAbilities(values []Ability) ([]Ability, error) {
	set := make(map[Ability]struct{}, len(values)+4)
	for _, value := range values {
		if !slices.Contains(knownAbilities, value) {
			return nil, fail(CodeInvalidAbilitySet, "unknown ability %q", value)
		}
		if _, duplicate := set[value]; duplicate {
			return nil, fail(CodeInvalidAbilitySet, "ability %q is declared more than once", value)
		}
		set[value] = struct{}{}
	}

	// Compiler-known closure keeps requirements simple without making these
	// implications source syntax. Scalars are inert copy values. Keys are stored.
	if _, found := set[AbilityScalar]; found {
		set[AbilityCopy] = struct{}{}
		set[AbilityDrop] = struct{}{}
		set[AbilityInert] = struct{}{}
		set[AbilityStore] = struct{}{}
	}
	if _, found := set[AbilityInert]; found {
		set[AbilityDrop] = struct{}{}
	}
	if _, found := set[AbilityKey]; found {
		set[AbilityStore] = struct{}{}
	}
	if _, resource := set[AbilityResource]; resource {
		if _, copyable := set[AbilityCopy]; copyable {
			return nil, fail(CodeInvalidAbilitySet, "resource and copy abilities are mutually exclusive")
		}
		if _, inert := set[AbilityInert]; inert {
			return nil, fail(CodeInvalidAbilitySet, "resource and inert abilities are mutually exclusive")
		}
		if _, scalar := set[AbilityScalar]; scalar {
			return nil, fail(CodeInvalidAbilitySet, "resource and scalar abilities are mutually exclusive")
		}
	}

	result := make([]Ability, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	slices.SortFunc(result, func(left, right Ability) int {
		return cmp.Compare(string(left), string(right))
	})
	return result, nil
}

func hasAbility(values []Ability, required Ability) bool {
	return slices.Contains(values, required)
}
