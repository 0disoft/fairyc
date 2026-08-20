package genericplan

import (
	"slices"
	"strings"
)

func validIdentifier(value string) bool {
	if value == "" || len(value) > 128 || !identifierStart(value[0]) {
		return false
	}
	for index := 1; index < len(value); index++ {
		if !identifierContinue(value[index]) {
			return false
		}
	}
	return true
}

func validPath(value string) bool {
	if value == "" || len(value) > 512 {
		return false
	}
	segments := strings.Split(value, "::")
	for _, segment := range segments {
		if !validIdentifier(segment) {
			return false
		}
	}
	return true
}

func identifierStart(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func identifierContinue(value byte) bool {
	return identifierStart(value) || value >= '0' && value <= '9'
}

func implementationKey(capability CapabilityID, arguments []TypeID) string {
	parts := make([]string, len(arguments))
	for index, argument := range arguments {
		parts[index] = string(argument)
	}
	return string(capability) + "<" + strings.Join(parts, ",") + ">"
}

func cloneConstraints(values []CapabilityConstraint) []CapabilityConstraint {
	result := make([]CapabilityConstraint, len(values))
	for index, value := range values {
		result[index] = CapabilityConstraint{Capability: value.Capability, Arguments: slices.Clone(value.Arguments)}
	}
	return result
}

func cloneDependencies(values []Dependency) []Dependency {
	result := make([]Dependency, len(values))
	for index, value := range values {
		result[index] = Dependency{Declaration: value.Declaration, Arguments: slices.Clone(value.Arguments)}
	}
	return result
}

func cloneImplementation(value Implementation) Implementation {
	value.Arguments = slices.Clone(value.Arguments)
	return value
}

func cloneTypeMap(values map[TypeID]TypeDescriptor) map[TypeID]TypeDescriptor {
	result := make(map[TypeID]TypeDescriptor, len(values))
	for key, value := range values {
		value.Abilities = slices.Clone(value.Abilities)
		result[key] = value
	}
	return result
}

func cloneCapabilityMap(values map[CapabilityID]CapabilityDescriptor) map[CapabilityID]CapabilityDescriptor {
	result := make(map[CapabilityID]CapabilityDescriptor, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func cloneDeclarationMap(values map[DeclarationID]Declaration) map[DeclarationID]Declaration {
	result := make(map[DeclarationID]Declaration, len(values))
	for key, value := range values {
		parameters := make([]TypeParameter, len(value.Parameters))
		for index, parameter := range value.Parameters {
			parameters[index] = TypeParameter{Name: parameter.Name, Abilities: slices.Clone(parameter.Abilities)}
		}
		value.Parameters = parameters
		value.Constraints = cloneConstraints(value.Constraints)
		value.Dependencies = cloneDependencies(value.Dependencies)
		result[key] = value
	}
	return result
}
