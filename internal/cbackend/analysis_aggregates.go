package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/mir"
	"strconv"
)

func (e *analysisBuilder) inspectStructOperation(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	switch operation.Kind {
	case mir.OperationStructConstruct:
		index, found := e.structByType[operation.Type]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "struct construction type " + strconv.Quote(operation.Type.String())}
		}
		definition := &e.structTypes[index]
		if len(operation.Fields) != len(definition.definition.Fields) || len(operation.Operands) != len(definition.definition.Fields) {
			return fmt.Errorf("struct construction in %s does not match type %q", function.Name, operation.Type)
		}
		for fieldIndex, field := range definition.definition.Fields {
			if operation.Fields[fieldIndex] != field.Name {
				return fmt.Errorf("struct construction in %s has field %q at index %d, want %q", function.Name, operation.Fields[fieldIndex], fieldIndex, field.Name)
			}
			operandType, err := e.operandType(functionIndex, operation.Operands[fieldIndex])
			if err != nil {
				return fmt.Errorf("struct construction field %d in %s: %w", fieldIndex, function.Name, err)
			}
			if canonicalCValueType(operandType) != canonicalCValueType(field.Type) {
				return fmt.Errorf("struct construction field %d in %s has type %q, want %q", fieldIndex, function.Name, operandType, field.Type)
			}
		}
	case mir.OperationFieldRead:
		index, found := e.structByType[fieldAggregateType(operation.OwnerType)]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "field-read owner type " + strconv.Quote(operation.OwnerType.String())}
		}
		definition := &e.structTypes[index]
		fieldIndex, found := definition.fieldIndex[operation.Field]
		if !found || canonicalCValueType(definition.definition.Fields[fieldIndex].Type) != canonicalCValueType(operation.Type) {
			return fmt.Errorf("struct field read in %s does not match field %q", function.Name, operation.Field)
		}
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("struct field read in %s: %w", function.Name, err)
		}
		receiverType, err := e.operandType(functionIndex, receiver)
		if err != nil {
			return fmt.Errorf("struct field read in %s: %w", function.Name, err)
		}
		if canonicalCValueType(receiverType) != canonicalCValueType(operation.OwnerType) {
			return fmt.Errorf("struct field read in %s has receiver type %q, want %q", function.Name, receiverType, operation.OwnerType)
		}
	}
	return nil
}

func (e *analysisBuilder) inspectEnumOperation(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	ownerType := operation.Type
	if operation.Kind == mir.OperationEnumPayload {
		ownerType = operation.OwnerType
	}
	index, found := e.enumByType[ownerType]
	if !found {
		return CapabilityError{Function: function.Name, Feature: "enum operation type " + strconv.Quote(ownerType.String())}
	}
	definition := &e.enumTypes[index]
	variantIndex, found := definition.variantIndex[operation.Variant]
	if !found {
		return fmt.Errorf("enum operation in %s names missing variant %q", function.Name, operation.Variant)
	}
	variant := definition.definition.Variants[variantIndex]
	switch operation.Kind {
	case mir.OperationEnumConstruct:
		if len(operation.Fields) != len(variant.Fields) || len(operation.Operands) != len(variant.Fields) {
			return fmt.Errorf("enum construction in %s does not match variant %q", function.Name, operation.Variant)
		}
		seen := make(map[string]struct{}, len(operation.Fields))
		for operandIndex, fieldName := range operation.Fields {
			fieldIndex, declared := definition.fieldIndex[variantIndex][fieldName]
			if !declared {
				return fmt.Errorf("enum construction in %s names missing field %q", function.Name, fieldName)
			}
			if _, duplicate := seen[fieldName]; duplicate {
				return fmt.Errorf("enum construction in %s repeats field %q", function.Name, fieldName)
			}
			seen[fieldName] = struct{}{}
			operandType, err := e.operandType(functionIndex, operation.Operands[operandIndex])
			if err != nil {
				return fmt.Errorf("enum construction field %q in %s: %w", fieldName, function.Name, err)
			}
			if canonicalCValueType(operandType) != canonicalCValueType(variant.Fields[fieldIndex].Type) {
				return fmt.Errorf("enum construction field %q in %s has type %q, want %q", fieldName, function.Name, operandType, variant.Fields[fieldIndex].Type)
			}
		}
	case mir.OperationEnumPayload:
		fieldIndex, declared := definition.fieldIndex[variantIndex][operation.Field]
		if !declared || canonicalCValueType(variant.Fields[fieldIndex].Type) != canonicalCValueType(operation.Type) {
			return fmt.Errorf("enum payload in %s does not match field %q", function.Name, operation.Field)
		}
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("enum payload in %s: %w", function.Name, err)
		}
		receiverType, err := e.operandType(functionIndex, receiver)
		if err != nil {
			return fmt.Errorf("enum payload in %s: %w", function.Name, err)
		}
		if canonicalCValueType(receiverType) != canonicalCValueType(operation.OwnerType) {
			return fmt.Errorf("enum payload in %s has receiver type %q, want %q", function.Name, receiverType, operation.OwnerType)
		}
	}
	return nil
}
