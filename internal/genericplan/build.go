package genericplan

import (
	"cmp"
	"slices"
)

// Build validates exact implementations, coherence, type expressions, and
// dependency arities before publishing immutable lookup tables.
func (b *Builder) Build() (*Catalog, error) {
	if b == nil {
		return nil, fail(CodeInvalidDeclaration, "catalog builder is nil")
	}

	orderedImplementations := slices.Clone(b.implementations)
	slices.SortFunc(orderedImplementations, func(left, right Implementation) int {
		leftKey := implementationKey(left.Capability, left.Arguments) + "=" + string(left.Provider)
		rightKey := implementationKey(right.Capability, right.Arguments) + "=" + string(right.Provider)
		return cmp.Compare(leftKey, rightKey)
	})
	implementationMap := make(map[string]Implementation, len(orderedImplementations))
	for _, implementation := range orderedImplementations {
		capability, found := b.capabilities[implementation.Capability]
		if !found {
			return nil, fail(CodeUnknownCatalogItem, "implementation references unknown capability %q", implementation.Capability)
		}
		if len(implementation.Arguments) != capability.Arity {
			return nil, fail(CodeTypeArgumentArity, "implementation for %q has %d arguments, expected %d", implementation.Capability, len(implementation.Arguments), capability.Arity)
		}
		ownsArgument := false
		for _, argument := range implementation.Arguments {
			typeValue, found := b.types[argument]
			if !found {
				return nil, fail(CodeUnknownCatalogItem, "implementation for %q references unknown type %q", implementation.Capability, argument)
			}
			ownsArgument = ownsArgument || typeValue.Owner == implementation.Owner
		}
		if capability.Owner != implementation.Owner && !ownsArgument {
			return nil, fail(CodeCoherenceViolation, "implementation owner %q owns neither capability %q nor an argument type", implementation.Owner, implementation.Capability)
		}
		key := implementationKey(implementation.Capability, implementation.Arguments)
		if previous, duplicate := implementationMap[key]; duplicate {
			return nil, fail(CodeDuplicateCatalogItem, "implementations %q and %q overlap for %s", previous.Provider, implementation.Provider, key)
		}
		implementationMap[key] = cloneImplementation(implementation)
	}

	declarationIDs := make([]DeclarationID, 0, len(b.declarations))
	for id := range b.declarations {
		declarationIDs = append(declarationIDs, id)
	}
	slices.SortFunc(declarationIDs, func(left, right DeclarationID) int {
		return cmp.Compare(string(left), string(right))
	})
	for _, declarationID := range declarationIDs {
		declaration := b.declarations[declarationID]
		parameterNames := make(map[string]struct{}, len(declaration.Parameters))
		for _, parameter := range declaration.Parameters {
			parameterNames[parameter.Name] = struct{}{}
		}
		seenConstraints := make(map[string]struct{}, len(declaration.Constraints))
		for _, constraint := range declaration.Constraints {
			constraintKey := expressionListKey(string(constraint.Capability), constraint.Arguments)
			if _, duplicate := seenConstraints[constraintKey]; duplicate {
				return nil, fail(CodeInvalidDeclaration, "declaration %q repeats constraint %s", declaration.ID, constraintKey)
			}
			seenConstraints[constraintKey] = struct{}{}
			capability, found := b.capabilities[constraint.Capability]
			if !found {
				return nil, fail(CodeUnknownCatalogItem, "declaration %q references unknown capability %q", declaration.ID, constraint.Capability)
			}
			if len(constraint.Arguments) != capability.Arity {
				return nil, fail(CodeTypeArgumentArity, "constraint %q in %q has %d arguments, expected %d", constraint.Capability, declaration.ID, len(constraint.Arguments), capability.Arity)
			}
			if err := validateExpressions(constraint.Arguments, parameterNames, b.types); err != nil {
				return nil, err
			}
		}
		seenDependencies := make(map[string]struct{}, len(declaration.Dependencies))
		for _, dependency := range declaration.Dependencies {
			dependencyKey := expressionListKey(string(dependency.Declaration), dependency.Arguments)
			if _, duplicate := seenDependencies[dependencyKey]; duplicate {
				return nil, fail(CodeInvalidDeclaration, "declaration %q repeats dependency %s", declaration.ID, dependencyKey)
			}
			seenDependencies[dependencyKey] = struct{}{}
			target, found := b.declarations[dependency.Declaration]
			if !found {
				return nil, fail(CodeUnknownCatalogItem, "declaration %q depends on unknown declaration %q", declaration.ID, dependency.Declaration)
			}
			if len(dependency.Arguments) != len(target.Parameters) {
				return nil, fail(CodeTypeArgumentArity, "dependency %q in %q has %d arguments, expected %d", dependency.Declaration, declaration.ID, len(dependency.Arguments), len(target.Parameters))
			}
			if err := validateExpressions(dependency.Arguments, parameterNames, b.types); err != nil {
				return nil, err
			}
		}
	}

	catalog := &Catalog{
		types:           cloneTypeMap(b.types),
		capabilities:    cloneCapabilityMap(b.capabilities),
		implementations: implementationMap,
		declarations:    cloneDeclarationMap(b.declarations),
	}
	catalog.digest = catalogDigest(catalog)
	return catalog, nil
}

func validateExpressions(expressions []TypeExpression, parameters map[string]struct{}, types map[TypeID]TypeDescriptor) error {
	for _, expression := range expressions {
		if (expression.Parameter == "") == (expression.Concrete == "") {
			return fail(CodeInvalidDeclaration, "type expression must contain exactly one parameter or concrete type")
		}
		if expression.Parameter != "" {
			if _, found := parameters[expression.Parameter]; !found {
				return fail(CodeInvalidDeclaration, "type expression references unknown parameter %q", expression.Parameter)
			}
			continue
		}
		if _, found := types[expression.Concrete]; !found {
			return fail(CodeUnknownCatalogItem, "type expression references unknown type %q", expression.Concrete)
		}
	}
	return nil
}
