package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/mir"
	"strconv"
)

func (e *analysisBuilder) inspectAssignment(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("assign in %s: %w", function.Name, err)
	}
	sourceType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("assign in %s: %w", function.Name, err)
	}
	destinationType, found := e.localTypes[functionIndex][operation.DestinationLocal]
	if !found {
		return fmt.Errorf("assign in %s has no type for destination local %d", function.Name, operation.DestinationLocal)
	}
	if canonicalCValueType(sourceType) != canonicalCValueType(destinationType) {
		return fmt.Errorf("assign in %s has source type %q and destination type %q", function.Name, sourceType, destinationType)
	}
	return nil
}

func (e *analysisBuilder) inspectResourceInitialize(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	resource, found := e.resourceLocals[functionIndex][operation.Symbol]
	if !found || operation.DestinationLocal != mir.LocalID(operation.Symbol) {
		return fmt.Errorf("resource initialization in %s does not name its declared local", function.Name)
	}
	if !e.supportedNeedsDropType(resource.Type) {
		return CapabilityError{Function: function.Name, Feature: "resource initialization type " + strconv.Quote(resource.Type.String())}
	}
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("resource initialization in %s: %w", function.Name, err)
	}
	valueType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("resource initialization in %s: %w", function.Name, err)
	}
	if canonicalCValueType(valueType) != canonicalCValueType(resource.Type) {
		return fmt.Errorf("resource initialization in %s has value type %q, want %q", function.Name, valueType, resource.Type)
	}
	return nil
}

func (e *analysisBuilder) inspectResourceDischarge(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	resource, found := e.resourceLocals[functionIndex][operation.Symbol]
	if !found {
		return fmt.Errorf("%s in %s names undeclared resource local %d", operation.Kind, function.Name, operation.Symbol)
	}
	if !e.supportedNeedsDropType(resource.Type) {
		return CapabilityError{Function: function.Name, Feature: "resource discharge type " + strconv.Quote(resource.Type.String())}
	}
	if operation.Kind == mir.OperationMove && operation.Destination == "" {
		return fmt.Errorf("resource move in %s has no destination class", function.Name)
	}
	if operation.Kind == mir.OperationDrop && (operation.DropFlagEntity != "" || !operation.FlagClearedFirst) {
		return CapabilityError{Function: function.Name, Feature: "conditional resource drop"}
	}
	if operation.Kind == mir.OperationBurn && (operation.Type != resource.Type || operation.Destination != "" || operation.DropFlagEntity != "" || operation.FlagClearedFirst) {
		return fmt.Errorf("resource burn in %s has invalid direct-discharge metadata", function.Name)
	}
	return nil
}

func (e *analysisBuilder) inspectResourceOperation(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	switch operation.Kind {
	case mir.OperationResourceConstruct:
		index, found := e.resourceByType[operation.Type]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "resource construction type " + strconv.Quote(operation.Type.String())}
		}
		definition := &e.resourceTypes[index]
		if len(operation.Fields) != len(definition.definition.Fields) || len(operation.Operands) != len(definition.definition.Fields) {
			return fmt.Errorf("resource construction in %s does not match type %q", function.Name, operation.Type)
		}
		for fieldIndex, field := range definition.definition.Fields {
			if operation.Fields[fieldIndex] != field.Name {
				return fmt.Errorf("resource construction in %s has field %q at index %d, want %q", function.Name, operation.Fields[fieldIndex], fieldIndex, field.Name)
			}
			operandType, err := e.operandType(functionIndex, operation.Operands[fieldIndex])
			if err != nil {
				return fmt.Errorf("resource construction field %d in %s: %w", fieldIndex, function.Name, err)
			}
			if operandType != field.Type {
				return fmt.Errorf(
					"resource construction field %d in %s has type %q, want %q",
					fieldIndex, function.Name, operandType, field.Type,
				)
			}
		}
	case mir.OperationFieldRead:
		index, found := e.resourceByType[fieldAggregateType(operation.OwnerType)]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "field-read owner type " + strconv.Quote(operation.OwnerType.String())}
		}
		definition := &e.resourceTypes[index]
		fieldIndex, found := definition.fieldIndex[operation.Field]
		if !found {
			return fmt.Errorf("resource field read in %s names missing field %q", function.Name, operation.Field)
		}
		if operation.Type != definition.definition.Fields[fieldIndex].Type {
			return fmt.Errorf(
				"resource field read in %s has type %q, want %q",
				function.Name, operation.Type, definition.definition.Fields[fieldIndex].Type,
			)
		}
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("resource field read in %s: %w", function.Name, err)
		}
		receiverType, err := e.operandType(functionIndex, receiver)
		if err != nil {
			return fmt.Errorf("resource field read in %s: %w", function.Name, err)
		}
		if receiverType != operation.OwnerType {
			return fmt.Errorf("resource field read in %s has receiver type %q, want %q", function.Name, receiverType, operation.OwnerType)
		}
	}
	return nil
}

func (e *analysisBuilder) inspectTaggedOperation(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	switch operation.Kind {
	case mir.OperationResultConstruct:
		successType, failureType, feature, supported := e.resultCSpec(operation.Type)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Result construction type " + strconv.Quote(operation.Type.String())}
		}
		expectedType := successType
		if operation.Variant == "err" {
			expectedType = failureType
		} else if operation.Variant != "ok" {
			return fmt.Errorf("Result construction in %s has invalid variant %q", function.Name, operation.Variant)
		}
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Result construction in %s: %w", function.Name, err)
		}
		payloadType, err := e.operandType(functionIndex, operand)
		if err != nil || canonicalCValueType(payloadType) != canonicalCValueType(expectedType) {
			return CapabilityError{Function: function.Name, Feature: "Result payload type " + strconv.Quote(payloadType.String())}
		}
		e.features[feature] = struct{}{}
	case mir.OperationResultPayload:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Result payload in %s: %w", function.Name, err)
		}
		containerType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return CapabilityError{Function: function.Name, Feature: "Result payload source type " + strconv.Quote(containerType.String())}
		}
		successType, failureType, feature, supported := e.resultCSpec(containerType)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Result payload source type " + strconv.Quote(containerType.String())}
		}
		expectedType := successType
		if operation.Variant == "err" {
			expectedType = failureType
		} else if operation.Variant != "ok" {
			return fmt.Errorf("Result payload in %s has invalid variant %q", function.Name, operation.Variant)
		}
		if canonicalCValueType(operation.Type) != canonicalCValueType(expectedType) || operation.MovePayload != e.supportedNeedsDropType(expectedType) {
			return CapabilityError{Function: function.Name, Feature: "Result payload extraction type " + strconv.Quote(operation.Type.String())}
		}
		e.features[feature] = struct{}{}
	case mir.OperationOptionConstruct:
		elementType, feature, supported := e.optionCSpec(operation.Type)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Option construction type " + strconv.Quote(operation.Type.String())}
		}
		if operation.Variant == "some" {
			operand, err := oneOperand(operation.Operands)
			if err != nil {
				return fmt.Errorf("Option::Some construction in %s: %w", function.Name, err)
			}
			payloadType, err := e.operandType(functionIndex, operand)
			if err != nil || canonicalCValueType(payloadType) != canonicalCValueType(elementType) {
				return CapabilityError{Function: function.Name, Feature: "Option payload type " + strconv.Quote(payloadType.String())}
			}
		} else if operation.Variant != "none" || len(operation.Operands) != 0 {
			return fmt.Errorf("Option construction in %s has invalid variant %q", function.Name, operation.Variant)
		}
		e.features[feature] = struct{}{}
	case mir.OperationOptionPayload:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Option payload in %s: %w", function.Name, err)
		}
		containerType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return CapabilityError{Function: function.Name, Feature: "Option payload source type " + strconv.Quote(containerType.String())}
		}
		elementType, feature, supported := e.optionCSpec(containerType)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Option payload source type " + strconv.Quote(containerType.String())}
		}
		if operation.Variant != "some" || canonicalCValueType(operation.Type) != canonicalCValueType(elementType) || operation.MovePayload != e.supportedNeedsDropType(elementType) {
			return CapabilityError{Function: function.Name, Feature: "Option payload extraction type " + strconv.Quote(operation.Type.String())}
		}
		e.features[feature] = struct{}{}
	case mir.OperationResultSwitch:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Result switch marker in %s: %w", function.Name, err)
		}
		containerType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return CapabilityError{Function: function.Name, Feature: "Result switch source type " + strconv.Quote(containerType.String())}
		}
		_, _, feature, supported := e.resultCSpec(containerType)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Result switch source type " + strconv.Quote(containerType.String())}
		}
		e.features[feature] = struct{}{}
	}
	return nil
}
