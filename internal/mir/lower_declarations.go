package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"slices"
	"strings"
)

func lowerEnumTypes(input hir.Package) ([]EnumType, error) {
	var result []EnumType
	seen := make(map[hir.Type]struct{})
	for _, module := range input.Modules {
		for _, enumeration := range module.Enums {
			if enumeration.Type == hir.ParseType("") || enumeration.Type == hir.TypeInvalid || enumeration.Name == "" {
				return nil, fmt.Errorf("enum %s::%s has no concrete identity", module.Path, enumeration.Name)
			}
			if _, duplicate := seen[enumeration.Type]; duplicate {
				return nil, fmt.Errorf("duplicate enum type %q", enumeration.Type)
			}
			seen[enumeration.Type] = struct{}{}
			definition := EnumType{
				CBase: enumeration.CBase,
				Type:  enumeration.Type, Name: enumeration.Name, Span: enumeration.Span,
				Entity: enumTypeEntity(enumeration.Type), Variants: make([]EnumVariant, 0, len(enumeration.Variants)),
			}
			variantNames := make(map[string]struct{}, len(enumeration.Variants))
			for _, variant := range enumeration.Variants {
				if variant.Name == "" {
					return nil, fmt.Errorf("enum %q has an empty variant name", enumeration.Type)
				}
				if _, duplicate := variantNames[variant.Name]; duplicate {
					return nil, fmt.Errorf("enum %q has duplicate variant %q", enumeration.Type, variant.Name)
				}
				variantNames[variant.Name] = struct{}{}
				lowered := EnumVariant{Name: variant.Name, Span: variant.Span, Fields: make([]EnumField, 0, len(variant.Fields))}
				fieldNames := make(map[string]struct{}, len(variant.Fields))
				for _, field := range variant.Fields {
					if field.Name == "" || field.Type == hir.ParseType("") || field.Type == hir.TypeInvalid {
						return nil, fmt.Errorf("enum %q variant %q has an invalid field", enumeration.Type, variant.Name)
					}
					if _, duplicate := fieldNames[field.Name]; duplicate {
						return nil, fmt.Errorf("enum %q variant %q has duplicate field %q", enumeration.Type, variant.Name, field.Name)
					}
					fieldNames[field.Name] = struct{}{}
					lowered.Fields = append(lowered.Fields, EnumField{Name: field.Name, Type: field.Type, Span: field.Span})
				}
				definition.Variants = append(definition.Variants, lowered)
			}
			result = append(result, definition)
		}
	}
	slices.SortFunc(result, func(left, right EnumType) int {
		return strings.Compare(left.Type.String(), right.Type.String())
	})
	return result, nil
}

func lowerStructTypes(input hir.Package) ([]StructType, error) {
	var result []StructType
	seen := make(map[hir.Type]struct{})
	for _, module := range input.Modules {
		for _, structure := range module.Structs {
			if structure.Type == hir.ParseType("") || structure.Type == hir.TypeInvalid || structure.Name == "" {
				return nil, fmt.Errorf("struct %s::%s has no concrete identity", module.Path, structure.Name)
			}
			if _, duplicate := seen[structure.Type]; duplicate {
				return nil, fmt.Errorf("duplicate struct type %q", structure.Type)
			}
			seen[structure.Type] = struct{}{}
			definition := StructType{
				Type: structure.Type, Name: structure.Name, Span: structure.Span,
				Entity: structTypeEntity(structure.Type), ReprC: structure.ReprC,
				Fields: make([]StructField, 0, len(structure.Fields)),
			}
			fieldNames := make(map[string]struct{}, len(structure.Fields))
			for _, field := range structure.Fields {
				if field.Name == "" || field.Type == hir.ParseType("") || field.Type == hir.TypeInvalid {
					return nil, fmt.Errorf("struct %q has an invalid field definition", structure.Type)
				}
				if _, duplicate := fieldNames[field.Name]; duplicate {
					return nil, fmt.Errorf("struct %q has duplicate field %q", structure.Type, field.Name)
				}
				fieldNames[field.Name] = struct{}{}
				definition.Fields = append(definition.Fields, StructField{Name: field.Name, Type: field.Type, Span: field.Span})
			}
			result = append(result, definition)
		}
	}
	slices.SortFunc(result, func(left, right StructType) int {
		return strings.Compare(left.Type.String(), right.Type.String())
	})
	return result, nil
}

func lowerForeignFunctions(input hir.Package) ([]ForeignFunction, error) {
	var result []ForeignFunction
	seen := make(map[hir.SymbolID]struct{})
	for _, module := range input.Modules {
		moduleIdentity := hir.QualifiedModulePath(module.Package, module.Path)
		for _, function := range module.ForeignFunctions {
			if function.Symbol == hir.NoSymbol || function.Name == "" || function.LinkName == "" || function.ABI == "" {
				return nil, fmt.Errorf("foreign function %s::%s has an incomplete declaration", module.Path, function.Name)
			}
			if _, duplicate := seen[function.Symbol]; duplicate {
				return nil, fmt.Errorf("duplicate foreign function symbol %d", function.Symbol)
			}
			seen[function.Symbol] = struct{}{}
			lowered := ForeignFunction{
				Symbol: function.Symbol, Name: moduleIdentity + "::" + function.Name,
				LinkName: function.LinkName, ABI: function.ABI, Span: function.Span,
				ResultType: function.ResultType,
				Entity:     sourceEntity("foreign", function.Span),
			}
			for _, parameter := range function.Parameters {
				lowered.ParameterTypes = append(lowered.ParameterTypes, parameter.Type)
			}
			result = append(result, lowered)
		}
	}
	slices.SortFunc(result, func(left, right ForeignFunction) int {
		if value := strings.Compare(left.LinkName, right.LinkName); value != 0 {
			return value
		}
		return int(left.Symbol - right.Symbol)
	})
	return result, nil
}

func lowerResourceTypes(input hir.Package) ([]ResourceType, error) {
	var result []ResourceType
	seen := make(map[hir.Type]struct{})
	for _, module := range input.Modules {
		for _, resource := range module.Resources {
			if resource.Type == hir.ParseType("") || resource.Type == hir.TypeInvalid {
				return nil, fmt.Errorf("resource %s::%s has no concrete type", module.Path, resource.Name)
			}
			if _, duplicate := seen[resource.Type]; duplicate {
				return nil, fmt.Errorf("duplicate resource type %q", resource.Type)
			}
			seen[resource.Type] = struct{}{}
			if resource.DropKind != hir.ResourceDropEmpty && resource.DropKind != hir.ResourceDropNonEmpty {
				return nil, fmt.Errorf("resource %q has unsupported drop kind %q", resource.Type, resource.DropKind)
			}
			if resource.DropKind == hir.ResourceDropEmpty && resource.DropSymbol != hir.NoSymbol ||
				resource.DropKind == hir.ResourceDropNonEmpty && resource.DropSymbol == hir.NoSymbol {
				return nil, fmt.Errorf("resource %q has inconsistent drop symbol %d for kind %q", resource.Type, resource.DropSymbol, resource.DropKind)
			}
			definition := ResourceType{
				Type: resource.Type, Span: resource.Span, Entity: resourceTypeEntity(resource.Type),
				DropKind: resource.DropKind, DropSpan: resource.DropSpan, DropSymbol: resource.DropSymbol,
				DropEffects: slices.Clone(resource.DropEffects),
				Fields:      make([]ResourceField, 0, len(resource.Fields)),
			}
			fieldNames := make(map[string]struct{}, len(resource.Fields))
			for _, field := range resource.Fields {
				if field.Name == "" || field.Type == hir.ParseType("") || field.Type == hir.TypeInvalid {
					return nil, fmt.Errorf("resource %q has an invalid field definition", resource.Type)
				}
				if _, duplicate := fieldNames[field.Name]; duplicate {
					return nil, fmt.Errorf("resource %q has duplicate field %q", resource.Type, field.Name)
				}
				fieldNames[field.Name] = struct{}{}
				definition.Fields = append(definition.Fields, ResourceField{Name: field.Name, Type: field.Type, Span: field.Span})
			}
			result = append(result, definition)
		}
	}
	slices.SortFunc(result, func(left, right ResourceType) int {
		return strings.Compare(left.Type.String(), right.Type.String())
	})
	return result, nil
}
