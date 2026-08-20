package genericplan

import (
	"cmp"
	"crypto/sha256"
	"hash"
	"slices"
	"strings"
)

func catalogDigest(catalog *Catalog) Digest {
	value := sha256.New()
	writeString(value, "fairyc.genericplan.catalog.v0")

	typeIDs := make([]TypeID, 0, len(catalog.types))
	for id := range catalog.types {
		typeIDs = append(typeIDs, id)
	}
	slices.SortFunc(typeIDs, func(left, right TypeID) int { return cmp.Compare(string(left), string(right)) })
	writeUint64(value, uint64(len(typeIDs)))
	for _, id := range typeIDs {
		typeValue := catalog.types[id]
		writeString(value, "type")
		writeString(value, string(typeValue.ID))
		writeString(value, string(typeValue.Owner))
		writeUint64(value, uint64(len(typeValue.Abilities)))
		for _, ability := range typeValue.Abilities {
			writeString(value, string(ability))
		}
		writeDigest(value, typeValue.Definition)
	}

	capabilityIDs := make([]CapabilityID, 0, len(catalog.capabilities))
	for id := range catalog.capabilities {
		capabilityIDs = append(capabilityIDs, id)
	}
	slices.SortFunc(capabilityIDs, func(left, right CapabilityID) int { return cmp.Compare(string(left), string(right)) })
	writeUint64(value, uint64(len(capabilityIDs)))
	for _, id := range capabilityIDs {
		capability := catalog.capabilities[id]
		writeString(value, "capability")
		writeString(value, string(capability.ID))
		writeString(value, string(capability.Owner))
		writeUint64(value, uint64(capability.Arity))
		writeDigest(value, capability.Definition)
	}

	implementationKeys := make([]string, 0, len(catalog.implementations))
	for key := range catalog.implementations {
		implementationKeys = append(implementationKeys, key)
	}
	slices.Sort(implementationKeys)
	writeUint64(value, uint64(len(implementationKeys)))
	for _, key := range implementationKeys {
		implementation := catalog.implementations[key]
		writeString(value, "implementation")
		writeString(value, key)
		writeString(value, string(implementation.Owner))
		writeString(value, string(implementation.Provider))
		writeDigest(value, implementation.ProviderDigest)
	}

	declarationIDs := make([]DeclarationID, 0, len(catalog.declarations))
	for id := range catalog.declarations {
		declarationIDs = append(declarationIDs, id)
	}
	slices.SortFunc(declarationIDs, func(left, right DeclarationID) int { return cmp.Compare(string(left), string(right)) })
	writeUint64(value, uint64(len(declarationIDs)))
	for _, id := range declarationIDs {
		declaration := catalog.declarations[id]
		writeString(value, "declaration")
		writeString(value, string(declaration.ID))
		writeString(value, string(declaration.Owner))
		writeDigest(value, declaration.Definition)
		writeUint64(value, uint64(len(declaration.Parameters)))
		for _, parameter := range declaration.Parameters {
			writeString(value, parameter.Name)
			writeUint64(value, uint64(len(parameter.Abilities)))
			for _, ability := range parameter.Abilities {
				writeString(value, string(ability))
			}
		}
		writeUint64(value, uint64(len(declaration.Constraints)))
		for _, constraint := range declaration.Constraints {
			writeString(value, string(constraint.Capability))
			writeExpressions(value, constraint.Arguments)
		}
		writeUint64(value, uint64(len(declaration.Dependencies)))
		for _, dependency := range declaration.Dependencies {
			writeString(value, string(dependency.Declaration))
			writeExpressions(value, dependency.Arguments)
		}
	}

	var result Digest
	copy(result[:], value.Sum(nil))
	return result
}

func writeExpressions(target hash.Hash, expressions []TypeExpression) {
	writeUint64(target, uint64(len(expressions)))
	for _, expression := range expressions {
		if expression.Parameter != "" {
			writeString(target, "parameter")
			writeString(target, expression.Parameter)
		} else {
			writeString(target, "concrete")
			writeString(target, string(expression.Concrete))
		}
	}
}

func expressionListKey(owner string, expressions []TypeExpression) string {
	parts := make([]string, len(expressions))
	for index, expression := range expressions {
		if expression.Parameter != "" {
			parts[index] = "$" + expression.Parameter
		} else {
			parts[index] = string(expression.Concrete)
		}
	}
	return owner + "<" + strings.Join(parts, ",") + ">"
}
