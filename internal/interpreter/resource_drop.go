package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func (r *executionState) dropValue(value runtimeValue, binding hir.Span, depth int) (string, error) {
	if err := r.checkCancellation(); err != nil {
		return "", err
	}
	switch value.kind {
	case valueI32, valueI64, valueU8, valueU32, valueUSize, valueNull:
		return "", nil
	case valueStruct:
		if !inertDropValue(value, 0) {
			return "", capabilityError("struct field is not an inert copy value")
		}
		return "", nil
	case valueResult, valueOption:
		if value.tagged() == nil {
			return "", fmt.Errorf("tagged value has no state")
		}
		if value.tagged().dropped {
			return "", fmt.Errorf("%s::%s was dropped twice", taggedFamily(value.kind), value.tagged().variant)
		}
		value.tagged().dropped = true
		if value.tagged().payload == nil {
			return "", nil
		}
		return r.dropValue(*value.tagged().payload, binding, depth)
	case valueResource:
		if value.resource() == nil {
			return "", fmt.Errorf("resource value has no state")
		}
		if value.resource().definition == nil ||
			value.resource().definition.DropKind != hir.ResourceDropEmpty && value.resource().definition.DropKind != hir.ResourceDropNonEmpty {
			return "", fmt.Errorf("resource value has no executable drop definition")
		}
		if value.resource().dropped || value.resource().dropping {
			return "", fmt.Errorf("resource %q was dropped twice", value.resource().definition.Type)
		}
		value.resource().dropping = true
		r.dropTrace = append(r.dropTrace, DropEvent{ResourceType: value.resource().definition.Type, Binding: binding})
		if value.resource().definition.DropKind == hir.ResourceDropNonEmpty && !r.abstractArenas[value.resource()] {
			glue := r.bySymbol[value.resource().definition.DropSymbol]
			if glue == nil || glue.DropGlueFor != value.resource().definition.Type {
				return "", fmt.Errorf("resource %q has no matching drop glue", value.resource().definition.Type)
			}
			completion, err := r.call(glue, []runtimeValue{value}, depth+1)
			if err != nil {
				return "", err
			}
			if completion.trapCategory != "" {
				return completion.trapCategory, nil
			}
		}
		value.resource().dropping = false
		value.resource().dropped = true
		for index := len(value.resource().definition.Fields) - 1; index >= 0; index-- {
			trap, err := r.dropValue(value.resource().fields[index], binding, depth)
			if err != nil || trap != "" {
				return trap, err
			}
		}
		return "", nil
	default:
		return "", capabilityError("value has no executable drop behavior")
	}
}

func taggedFamily(kind valueKind) string {
	if kind == valueResult {
		return "Result"
	}
	return "Option"
}
