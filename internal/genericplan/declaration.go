package genericplan

import (
	"cmp"
	"slices"
)

// canonicalDeclaration validates in the same order as AddDeclaration and
// returns detached data. Builder state is published only after it succeeds.
func canonicalDeclaration(value Declaration, limits CatalogLimits) (Declaration, error) {
	if value.Definition.IsZero() {
		return Declaration{}, fail(CodeInvalidDeclaration, "declaration %q has no definition digest", value.ID)
	}
	if len(value.Parameters) == 0 || len(value.Parameters) > limits.MaxTypeParameters {
		return Declaration{}, fail(CodeInvalidDeclaration, "declaration %q parameter count is outside 1..%d", value.ID, limits.MaxTypeParameters)
	}
	if len(value.Constraints) > limits.MaxConstraints {
		return Declaration{}, fail(CodeCatalogBudgetExceeded, "declaration %q constraint count exceeds %d", value.ID, limits.MaxConstraints)
	}
	if len(value.Dependencies) > limits.MaxDependencies {
		return Declaration{}, fail(CodeCatalogBudgetExceeded, "declaration %q dependency count exceeds %d", value.ID, limits.MaxDependencies)
	}
	parameters, err := canonicalDeclarationParameters(value.ID, value.Parameters)
	if err != nil {
		return Declaration{}, err
	}
	value.Parameters = parameters
	value.Constraints = cloneConstraints(value.Constraints)
	sortDeclarationEntries(value.Constraints, func(item CapabilityConstraint) string {
		return expressionListKey(string(item.Capability), item.Arguments)
	})
	value.Dependencies = cloneDependencies(value.Dependencies)
	sortDeclarationEntries(value.Dependencies, func(item Dependency) string {
		return expressionListKey(string(item.Declaration), item.Arguments)
	})
	return value, nil
}

func canonicalDeclarationParameters(id DeclarationID, values []TypeParameter) ([]TypeParameter, error) {
	parameters := make([]TypeParameter, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, parameter := range values {
		if !validIdentifier(parameter.Name) {
			return nil, fail(CodeInvalidIdentifier, "declaration %q parameter %q is invalid", id, parameter.Name)
		}
		if _, duplicate := seen[parameter.Name]; duplicate {
			return nil, fail(CodeInvalidDeclaration, "declaration %q parameter %q is duplicated", id, parameter.Name)
		}
		seen[parameter.Name] = struct{}{}
		abilities, err := canonicalAbilities(parameter.Abilities)
		if err != nil {
			return nil, err
		}
		parameters[index] = TypeParameter{Name: parameter.Name, Abilities: abilities}
	}
	return parameters, nil
}

// sortDeclarationEntries mutates only the detached working slices above. Keep
// the exact encoded-key order, computing each key once instead of per comparison.
func sortDeclarationEntries[T any](values []T, key func(T) string) {
	if len(values) < 2 {
		return
	}
	type entry struct {
		key   string
		value T
	}
	entries := make([]entry, len(values))
	for index, value := range values {
		entries[index] = entry{key: key(value), value: value}
	}
	slices.SortFunc(entries, func(left, right entry) int { return cmp.Compare(left.key, right.key) })
	for index, entry := range entries {
		values[index] = entry.value
	}
}
