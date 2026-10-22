package mir

import (
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

func verifyEnumTypes(definitions []EnumType) (map[hir.Type]*EnumType, []VerificationError) {
	enums := make(map[hir.Type]*EnumType, len(definitions))
	var failures []VerificationError
	for index := range definitions {
		definition := &definitions[index]
		entity := fallbackEntity(definition.Entity, enumTypeEntity(definition.Type))
		if definition.Type == hir.ParseType("") || definition.Type == hir.TypeInvalid || definition.Name == "" {
			failures = append(failures, VerificationError{Code: CodeAggregateIdentity, Entity: entity, Detail: "enum definition has no concrete identity"})
			continue
		}
		if _, duplicate := enums[definition.Type]; duplicate {
			failures = append(failures, VerificationError{Code: CodeAggregateIdentity, Entity: entity, Detail: "enum definition is duplicated"})
			continue
		}
		enums[definition.Type] = definition
		variantNames := make(map[string]struct{}, len(definition.Variants))
		for _, variant := range definition.Variants {
			if variant.Name == "" {
				failures = append(failures, VerificationError{Code: CodeAggregateFields, Entity: entity, Detail: "enum definition has an empty variant"})
				continue
			}
			if _, duplicate := variantNames[variant.Name]; duplicate {
				failures = append(failures, VerificationError{Code: CodeAggregateFields, Entity: entity, Detail: "enum definition has a duplicate variant"})
				continue
			}
			variantNames[variant.Name] = struct{}{}
			fieldNames := make(map[string]struct{}, len(variant.Fields))
			for _, field := range variant.Fields {
				if field.Name == "" || field.Type == hir.ParseType("") || field.Type == hir.TypeInvalid {
					failures = append(failures, VerificationError{Code: CodeAggregateFields, Entity: entity, Detail: "enum definition has an invalid payload field"})
					continue
				}
				if _, duplicate := fieldNames[field.Name]; duplicate {
					failures = append(failures, VerificationError{Code: CodeAggregateFields, Entity: entity, Detail: "enum definition has a duplicate payload field"})
					continue
				}
				fieldNames[field.Name] = struct{}{}
			}
		}
	}
	return enums, failures
}

func verifyStructTypes(definitions []StructType) (map[hir.Type]*StructType, []VerificationError) {
	structs := make(map[hir.Type]*StructType, len(definitions))
	var failures []VerificationError
	for index := range definitions {
		definition := &definitions[index]
		entity := fallbackEntity(definition.Entity, structTypeEntity(definition.Type))
		if definition.Type == hir.ParseType("") || definition.Type == hir.TypeInvalid || definition.Name == "" {
			failures = append(failures, VerificationError{Code: CodeAggregateIdentity, Entity: entity, Detail: "struct definition has no concrete identity"})
			continue
		}
		if _, duplicate := structs[definition.Type]; duplicate {
			failures = append(failures, VerificationError{Code: CodeAggregateIdentity, Entity: entity, Detail: "struct definition is duplicated"})
			continue
		}
		structs[definition.Type] = definition
		if definition.ReprC && len(definition.Fields) == 0 {
			failures = append(failures, VerificationError{Code: CodeAggregateFields, Entity: entity, Detail: "repr(C) struct definition is empty"})
		}
		fieldNames := make(map[string]struct{}, len(definition.Fields))
		for _, field := range definition.Fields {
			if field.Name == "" || field.Type == hir.ParseType("") || field.Type == hir.TypeInvalid {
				failures = append(failures, VerificationError{Code: CodeAggregateFields, Entity: entity, Detail: "struct definition has an invalid field"})
				continue
			}
			if _, duplicate := fieldNames[field.Name]; duplicate {
				failures = append(failures, VerificationError{Code: CodeAggregateFields, Entity: entity, Detail: "struct definition has a duplicate field"})
				continue
			}
			fieldNames[field.Name] = struct{}{}
		}
	}
	properties := typeProperties{structs: structs}
	for _, definition := range definitions {
		for _, field := range definition.Fields {
			allowed := properties.inlineArrayField(field.Type)
			if definition.ReprC {
				allowed = hir.ReprCArrayField(field.Type)
			}
			if _, _, array := hir.FixedArrayParts(field.Type); array && !allowed {
				failures = append(failures, VerificationError{Code: CodeAggregateFields, Entity: fallbackEntity(definition.Entity, definition.Type.String()), Detail: "invalid inline array struct field"})
			}
		}
	}
	return structs, failures
}

func verifyResourceTypes(definitions []ResourceType) (map[hir.Type]*ResourceType, []VerificationError) {
	resources := make(map[hir.Type]*ResourceType, len(definitions))
	var failures []VerificationError
	for index := range definitions {
		definition := &definitions[index]
		entity := fallbackEntity(definition.Entity, resourceTypeEntity(definition.Type))
		if definition.Type == hir.ParseType("") || definition.Type == hir.TypeInvalid {
			failures = append(failures, VerificationError{Code: CodeAggregateIdentity, Entity: entity, Detail: "resource definition has no concrete type"})
			continue
		}
		if !slices.Equal(definition.DropEffects, hir.CanonicalEffects(definition.DropEffects)) {
			failures = append(failures, VerificationError{Code: CodeDropGlue, Entity: entity, Detail: "drop effects are not canonical"})
		}
		for _, effect := range definition.DropEffects {
			if _, valid := hir.ParseEffect(string(effect)); !valid {
				failures = append(failures, VerificationError{Code: CodeDropGlue, Entity: entity, Detail: "drop effect is outside the Stage 0 vocabulary"})
				break
			}
		}
		if _, duplicate := resources[definition.Type]; duplicate {
			failures = append(failures, VerificationError{Code: CodeAggregateIdentity, Entity: entity, Detail: "resource definition is duplicated"})
			continue
		}
		resources[definition.Type] = definition
		if definition.DropKind != hir.ResourceDropEmpty && definition.DropKind != hir.ResourceDropNonEmpty {
			failures = append(failures, VerificationError{Code: CodeDropGlue, Entity: entity, Detail: "resource definition has unsupported drop behavior"})
		}
		fieldNames := make(map[string]struct{}, len(definition.Fields))
		for _, field := range definition.Fields {
			if field.Name == "" || field.Type == hir.ParseType("") || field.Type == hir.TypeInvalid {
				failures = append(failures, VerificationError{Code: CodeAggregateFields, Entity: entity, Detail: "resource definition has an invalid field"})
				continue
			}
			if _, duplicate := fieldNames[field.Name]; duplicate {
				failures = append(failures, VerificationError{Code: CodeAggregateFields, Entity: entity, Detail: "resource definition has a duplicate field"})
				continue
			}
			fieldNames[field.Name] = struct{}{}
		}
	}
	return resources, failures
}

func enumVariantNames(variants []EnumVariant) []string {
	result := make([]string, len(variants))
	for index := range variants {
		result[index] = variants[index].Name
	}
	return result
}

func uniqueNonEmpty(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func containsDefinedResource(value hir.Type, definitions map[hir.Type]*ResourceType) bool {
	if definitions[value] != nil {
		return true
	}
	if element, option := hir.OptionElement(value); option {
		return containsDefinedResource(element, definitions)
	}
	success, failure, result := hir.ResultElements(value)
	return result && (containsDefinedResource(success, definitions) || containsDefinedResource(failure, definitions))
}

func resourceFieldNames(fields []ResourceField) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}

func structFieldNames(fields []StructField) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}

func resourceField(fields []ResourceField, name string) (ResourceField, bool) {
	for _, field := range fields {
		if field.Name == name {
			return field, true
		}
	}
	return ResourceField{}, false
}

func structField(fields []StructField, name string) (StructField, bool) {
	for _, field := range fields {
		if field.Name == name {
			return field, true
		}
	}
	return StructField{}, false
}

func fieldAggregateType(value hir.Type) hir.Type {
	if element, _, borrow := hir.BorrowElement(value); borrow {
		return element
	}
	return value
}
