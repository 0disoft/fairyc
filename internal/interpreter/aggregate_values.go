package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func enumValue(definition *mir.EnumType, variantName string, fieldNames []string, values []runtimeValue) (runtimeValue, error) {
	if definition == nil || definition.Type == hir.TypeInvalid || len(fieldNames) != len(values) {
		return runtimeValue{}, fmt.Errorf("invalid enum construction definition")
	}
	var variant *mir.EnumVariant
	for index := range definition.Variants {
		if definition.Variants[index].Name == variantName {
			variant = &definition.Variants[index]
			break
		}
	}
	if variant == nil || len(variant.Fields) != len(fieldNames) {
		return runtimeValue{}, fmt.Errorf("enum %q has no matching variant %q", definition.Type, variantName)
	}
	state := &enumState{definition: definition, variant: variantName, fields: make([]runtimeValue, len(values))}
	for index, field := range variant.Fields {
		// MIR verification already fixes declaration order; defend this private boundary too.
		if field.Name == "" || fieldNames[index] != field.Name {
			return runtimeValue{}, fmt.Errorf("enum %q variant %q has mismatched field order", definition.Type, variantName)
		}
		state.fields[index] = values[index].cloneForValueCopy()
	}
	return runtimeValue{kind: valueEnum, typ: definition.Type, enum: state}, nil
}

func resourceValue(definition *mir.ResourceType, values []runtimeValue) (runtimeValue, error) {
	if definition == nil || definition.Type == hir.TypeInvalid || len(definition.Fields) == 0 || len(definition.Fields) != len(values) || definition.DropKind != hir.ResourceDropEmpty && definition.DropKind != hir.ResourceDropNonEmpty {
		return runtimeValue{}, fmt.Errorf("invalid resource construction definition")
	}
	state := &resourceState{definition: definition, fields: copyValueFields(values)}
	return runtimeValue{kind: valueResource, typ: definition.Type, resource: state}, nil
}

func structValue(definition *mir.StructType, values []runtimeValue) (runtimeValue, error) {
	if definition == nil || definition.Type == hir.TypeInvalid || len(definition.Fields) != len(values) {
		return runtimeValue{}, fmt.Errorf("invalid struct construction definition")
	}
	state := &structState{definition: definition, fields: copyValueFields(values)}
	return runtimeValue{kind: valueStruct, typ: definition.Type, structure: state}, nil
}

func structFieldIndex(definition *mir.StructType, name string) int {
	if definition != nil {
		for index, field := range definition.Fields {
			if field.Name == name {
				return index
			}
		}
	}
	return -1
}

func resourceFieldIndex(definition *mir.ResourceType, name string) int {
	if definition != nil {
		for index, field := range definition.Fields {
			if field.Name == name {
				return index
			}
		}
	}
	return -1
}

func indexedField(fields []runtimeValue, index int) (runtimeValue, bool) {
	if index < 0 || index >= len(fields) {
		return runtimeValue{}, false
	}
	return fields[index], true
}

func (v runtimeValue) field(definition *mir.ResourceType, name string) (runtimeValue, error) {
	if v.kind != valueResource || v.resource == nil {
		return runtimeValue{}, capabilityError("field read received a non-resource value")
	}
	if definition == nil || v.resource.definition == nil || v.resource.definition.Type != definition.Type {
		return runtimeValue{}, fmt.Errorf("field read resource definition does not match the runtime value")
	}
	if v.resource.dropped {
		return runtimeValue{}, fmt.Errorf("resource %q was read after drop", v.resource.definition.Type)
	}
	value, found := indexedField(v.resource.fields, resourceFieldIndex(definition, name))
	if !found {
		return runtimeValue{}, fmt.Errorf("resource %q has no field %q", definition.Type, name)
	}
	return value, nil
}

func (v runtimeValue) structField(definition *mir.StructType, name string) (runtimeValue, error) {
	if v.kind != valueStruct || v.structure == nil {
		return runtimeValue{}, capabilityError("field read received a non-struct value")
	}
	if definition == nil || v.structure.definition == nil || v.structure.definition.Type != definition.Type {
		return runtimeValue{}, fmt.Errorf("field read struct definition does not match the runtime value")
	}
	value, found := indexedField(v.structure.fields, structFieldIndex(definition, name))
	if !found {
		return runtimeValue{}, fmt.Errorf("struct %q has no field %q", definition.Type, name)
	}
	return value, nil
}

func (v runtimeValue) enumPayload(definition *mir.EnumType, variant, field string) (runtimeValue, error) {
	if v.kind != valueEnum || v.enum == nil || v.enum.definition == nil {
		return runtimeValue{}, capabilityError("enum payload received a non-enum value")
	}
	if definition == nil || v.enum.definition.Type != definition.Type || v.enum.variant != variant {
		return runtimeValue{}, fmt.Errorf("enum payload definition or active variant does not match")
	}
	for _, declared := range definition.Variants {
		if declared.Name != variant {
			continue
		}
		for index, item := range declared.Fields {
			if item.Name == field {
				if value, found := indexedField(v.enum.fields, index); found {
					return value, nil
				}
			}
		}
	}
	return runtimeValue{}, fmt.Errorf("enum %q variant %q has no active field %q", definition.Type, variant, field)
}
