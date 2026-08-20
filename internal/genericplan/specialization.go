package genericplan

import (
	"cmp"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
)

func (p *planner) resolveExpressions(expressions []TypeExpression, bindings map[string]TypeDescriptor) ([]TypeID, error) {
	result := make([]TypeID, len(expressions))
	for index, expression := range expressions {
		if expression.Parameter != "" {
			value, found := bindings[expression.Parameter]
			if !found {
				return nil, fail(CodeInvalidDeclaration, "unknown substituted parameter %q", expression.Parameter)
			}
			result[index] = value.ID
			continue
		}
		if _, found := p.catalog.types[expression.Concrete]; !found {
			return nil, fail(CodeUnknownCatalogItem, "unknown substituted concrete type %q", expression.Concrete)
		}
		result[index] = expression.Concrete
	}
	return result, nil
}

func (p *planner) specializationID(declaration Declaration, arguments []TypeID, constraints []ResolvedConstraint, dependencies []SpecializationID) SpecializationID {
	value := sha256.New()
	writeString(value, "fairyc.genericplan.specialization.v0")
	writeString(value, string(declaration.ID))
	writeString(value, string(declaration.Owner))
	writeDigest(value, declaration.Definition)
	writeUint64(value, uint64(len(arguments)))
	for _, argument := range arguments {
		typeValue := p.catalog.types[argument]
		writeString(value, string(typeValue.ID))
		writeString(value, string(typeValue.Owner))
		writeUint64(value, uint64(len(typeValue.Abilities)))
		for _, ability := range typeValue.Abilities {
			writeString(value, string(ability))
		}
		writeDigest(value, typeValue.Definition)
	}
	writeUint64(value, uint64(len(constraints)))
	for _, constraint := range constraints {
		capability := p.catalog.capabilities[constraint.Capability]
		writeString(value, string(constraint.Capability))
		writeDigest(value, capability.Definition)
		writeUint64(value, uint64(len(constraint.Arguments)))
		for _, argument := range constraint.Arguments {
			writeString(value, string(argument))
			writeDigest(value, p.catalog.types[argument].Definition)
		}
		writeString(value, string(constraint.Provider))
		writeDigest(value, constraint.ProviderDigest)
	}
	writeUint64(value, uint64(len(dependencies)))
	for _, dependency := range dependencies {
		writeString(value, string(dependency))
	}
	return SpecializationID("g0_" + fmt.Sprintf("%x", value.Sum(nil)))
}

func requestKey(value Request) string {
	arguments := make([]string, len(value.Arguments))
	for index, argument := range value.Arguments {
		arguments[index] = string(argument)
	}
	return string(value.Declaration) + "<" + strings.Join(arguments, ",") + ">"
}

func compareRequest(left, right Request) int {
	return cmp.Compare(requestKey(left), requestKey(right))
}

func compareResolvedConstraint(left, right ResolvedConstraint) int {
	leftKey := implementationKey(left.Capability, left.Arguments) + "=" + string(left.Provider)
	rightKey := implementationKey(right.Capability, right.Arguments) + "=" + string(right.Provider)
	return cmp.Compare(leftKey, rightKey)
}

func cloneRequests(values []Request) []Request {
	result := make([]Request, len(values))
	for index, value := range values {
		result[index] = Request{Declaration: value.Declaration, Arguments: slices.Clone(value.Arguments)}
	}
	return result
}

func cloneSpecializations(values []Specialization) []Specialization {
	result := make([]Specialization, len(values))
	for index, value := range values {
		constraints := make([]ResolvedConstraint, len(value.Constraints))
		for constraintIndex, constraint := range value.Constraints {
			constraint.Arguments = slices.Clone(constraint.Arguments)
			constraints[constraintIndex] = constraint
		}
		value.Bindings = slices.Clone(value.Bindings)
		value.Constraints = constraints
		value.Dependencies = slices.Clone(value.Dependencies)
		result[index] = value
	}
	return result
}
