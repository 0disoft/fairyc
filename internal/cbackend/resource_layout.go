package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
	"strconv"
)

func prepareResourceTypes(definitions []mir.ResourceType, engine *layout.Engine, structures []structCType, structByType map[hir.Type]int, enumerations []enumCType, enumByType map[hir.Type]int) ([]resourceCType, map[hir.Type]int, error) {
	ordered, err := orderResourceDefinitions(definitions)
	if err != nil {
		return nil, nil, err
	}
	result := make([]resourceCType, 0, len(ordered))
	byType := make(map[hir.Type]int, len(ordered))
	for _, definition := range ordered {
		if definition.DropKind != hir.ResourceDropEmpty && definition.DropKind != hir.ResourceDropNonEmpty {
			return nil, nil, CapabilityError{Feature: "resource type " + strconv.Quote(definition.Type.String()) + " with unsupported drop behavior"}
		}
		if len(definition.Fields) == 0 {
			return nil, nil, CapabilityError{Feature: "empty resource type " + strconv.Quote(definition.Type.String())}
		}
		fields := make([]layout.Field, 0, len(definition.Fields))
		fieldIndex := make(map[string]int, len(definition.Fields))
		allI32 := true
		nestedCopy := false
		nestedResource := false
		for index, field := range definition.Fields {
			fieldType, ok := resourceFieldLayoutType(field.Type)
			if structure, found := structByType[field.Type]; !ok && found {
				fieldType, ok = structures[structure].layoutType, true
				nestedCopy = true
			}
			if enumeration, found := enumByType[field.Type]; !ok && found {
				fieldType, ok = enumerations[enumeration].layoutType, true
				nestedCopy = true
			}
			if resource, found := byType[field.Type]; !ok && found {
				fieldType, ok = result[resource].layoutType, true
				nestedResource = true
			}
			if !ok {
				return nil, nil, CapabilityError{Feature: "resource type " + strconv.Quote(definition.Type.String()) + " field " + strconv.Quote(field.Name) + " of type " + strconv.Quote(field.Type.String())}
			}
			allI32 = allI32 && field.Type == hir.TypeI32
			fieldIndex[field.Name] = index
			fields = append(fields, layout.Field{
				Name: fmt.Sprintf("f_%d", index),
				Type: fieldType,
			})
		}
		aggregateType := layout.Type{Kind: layout.KindStruct, Fields: fields}
		computed, err := engine.Of(aggregateType)
		if err != nil {
			return nil, nil, fmt.Errorf("compute C transport layout for resource %q: %w", definition.Type, err)
		}
		if len(computed.Fields) != len(definition.Fields) {
			return nil, nil, fmt.Errorf("resource %q layout returned %d fields for %d definitions", definition.Type, len(computed.Fields), len(definition.Fields))
		}
		if _, duplicate := byType[definition.Type]; duplicate {
			return nil, nil, fmt.Errorf("duplicate C resource type %q", definition.Type)
		}
		byType[definition.Type] = len(result)
		feature := "flat-copy-resource"
		if allI32 {
			feature = "flat-i32-resource"
		}
		if nestedCopy {
			feature = "nested-copy-resource"
		}
		if nestedResource {
			feature = "nested-resource"
		}
		result = append(result, resourceCType{
			definition: definition, layout: computed, layoutType: aggregateType,
			fieldIndex: fieldIndex, feature: feature,
		})
	}
	return result, byType, nil
}

func resourceFieldLayoutType(value hir.Type) (layout.Type, bool) {
	canonical := canonicalCValueType(value)
	if canonical == hir.TypeU8 || canonical == hir.TypeI32 || canonical == hir.TypeI64 || canonical == hir.TypeU32 || canonical == hir.TypeUSize {
		return layout.Type{Kind: layout.KindLanguageScalar, Name: canonical.String()}, true
	}
	if _, _, pointer := hir.RawPointerElement(value); pointer {
		return layout.Type{Kind: layout.KindPointer}, true
	}
	return layout.Type{}, false
}
