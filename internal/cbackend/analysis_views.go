package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"slices"
	"strconv"
)

func validateOperationShape(function string, operation mir.Operation) error {
	switch operation.Kind {
	case mir.OperationAssign, mir.OperationInitialize, mir.OperationUnary, mir.OperationBinary,
		mir.OperationRawAddress, mir.OperationRawBorrowPointer, mir.OperationStrFromRawParts, mir.OperationCStrFromRawParts,
		mir.OperationByteSliceFromRawParts, mir.OperationMutByteSliceFromRawParts,
		mir.OperationStrDataAddress, mir.OperationByteSliceDataAddress, mir.OperationCStrDataAddress,
		mir.OperationStructConstruct, mir.OperationEnumConstruct, mir.OperationEnumPayload,
		mir.OperationResourceConstruct, mir.OperationFieldRead, mir.OperationFieldBorrow, mir.OperationViewLength, mir.OperationViewEmpty, mir.OperationViewIndex, mir.OperationViewElementBorrow, mir.OperationViewSlice, mir.OperationScalarCast, mir.OperationSharedReborrow,
		mir.OperationCCallback, mir.OperationStorageLayout, mir.OperationStorageTake, mir.OperationStorageBorrow, mir.OperationStorageBorrowMut, mir.OperationStorageElementAddress, mir.OperationArrayConstruct, mir.OperationArrayBorrowView, mir.OperationArrayLength, mir.OperationArrayIndex,
		mir.OperationRegionPlace:
		if operation.DestinationLocal < 0 {
			return fmt.Errorf("%s operation in %s has no destination local", operation.Kind, function)
		}
	case mir.OperationMove, mir.OperationDrop, mir.OperationBurn:
		if operation.Symbol == hir.NoSymbol {
			return fmt.Errorf("%s operation in %s has no resource symbol", operation.Kind, function)
		}
	case mir.OperationRegionEnter, mir.OperationRegionReset:
		if operation.Symbol == hir.NoSymbol || operation.DestinationLocal != mir.NoLocal {
			return fmt.Errorf("%s operation in %s has invalid Region identity", operation.Kind, function)
		}
	}
	return nil
}

func (e *analysisBuilder) inspectViewLength(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	receiver, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("borrowed-view length in %s: %w", function.Name, err)
	}
	element, _, borrow := hir.BorrowElement(operation.OwnerType)
	if !borrow || element != hir.TypeStr && element != hir.TypeByteSlice && !hir.NonByteIndexedView(operation.OwnerType) || operation.Type != hir.TypeUSize || operation.Field != "len" {
		return fmt.Errorf("borrowed-view length in %s has incompatible owner, result, or field", function.Name)
	}
	receiverType, err := e.operandType(functionIndex, receiver)
	if err != nil {
		return fmt.Errorf("borrowed-view length in %s: %w", function.Name, err)
	}
	if receiverType != operation.OwnerType {
		return fmt.Errorf("borrowed-view length in %s has receiver type %q, want %q", function.Name, receiverType, operation.OwnerType)
	}
	return nil
}

func (e *analysisBuilder) inspectArrayConstruction(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	elementType, length, array := hir.FixedArrayParts(operation.Type)
	if !array || operation.OwnerType != elementType || operation.Operator != "list" ||
		operation.DestinationLocal == mir.NoLocal || uint64(len(operation.Operands)) != length {
		return fmt.Errorf("fixed-array construction in %s has incompatible type or length", function.Name)
	}
	if _, found := e.arrayByType[operation.Type]; !found {
		return CapabilityError{Function: function.Name, Feature: "fixed-array type " + strconv.Quote(operation.Type.String())}
	}
	for index, operand := range operation.Operands {
		operandType, err := e.operandType(functionIndex, operand)
		if err != nil || operandType != elementType {
			return fmt.Errorf("fixed-array construction element %d in %s has incompatible type", index, function.Name)
		}
	}
	return nil
}

func (e *analysisBuilder) inspectArrayBorrowView(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	receiver, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("fixed-array byte-view borrow in %s: %w", function.Name, err)
	}
	if !hir.ArrayViewBorrow(operation.OwnerType, operation.Type, operation.Operator) {
		return fmt.Errorf("fixed-array byte-view borrow in %s has incompatible owner, result, or operator", function.Name)
	}
	receiverType, err := e.operandType(functionIndex, receiver)
	if err != nil {
		return fmt.Errorf("fixed-array byte-view borrow in %s: %w", function.Name, err)
	}
	if receiver.Kind != mir.OperandLocal || receiverType != operation.OwnerType {
		return fmt.Errorf("fixed-array byte-view borrow in %s has receiver type %q, want local %q", function.Name, receiverType, operation.OwnerType)
	}
	return nil
}

func (e *analysisBuilder) inspectArrayLength(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	receiver, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("fixed-array length in %s: %w", function.Name, err)
	}
	elementType, length, array := hir.FixedArrayParts(operation.OwnerType)
	if !array || !hir.ExecutableFixedArrayShape(elementType, length) || operation.Type != hir.TypeUSize || operation.Field != "len" {
		return fmt.Errorf("fixed-array length in %s has incompatible owner, result, or field", function.Name)
	}
	receiverType, err := e.operandType(functionIndex, receiver)
	if err != nil {
		return fmt.Errorf("fixed-array length in %s: %w", function.Name, err)
	}
	if receiverType != operation.OwnerType {
		return fmt.Errorf("fixed-array length in %s has receiver type %q, want %q", function.Name, receiverType, operation.OwnerType)
	}
	return nil
}

func (e *analysisBuilder) inspectArrayIndex(functionIndex int, operation mir.Operation, destination bool) error {
	function := &e.functions[functionIndex]
	receiver, index, err := twoOperands(operation.Operands)
	if err != nil {
		return fmt.Errorf("fixed-array index in %s: %w", function.Name, err)
	}
	elementType, _, array := hir.FixedArrayParts(operation.OwnerType)
	if !array || operation.Type != elementType || operation.IndexType != hir.TypeUSize || operation.Operator != "[]" ||
		destination != (operation.DestinationLocal != mir.NoLocal) {
		return fmt.Errorf("fixed-array index in %s has incompatible owner, result, operator, or destination", function.Name)
	}
	receiverType, receiverErr := e.operandType(functionIndex, receiver)
	indexType, indexErr := e.operandType(functionIndex, index)
	if receiverErr != nil || indexErr != nil || receiverType != operation.OwnerType || indexType != hir.TypeUSize {
		return fmt.Errorf("fixed-array index in %s has incompatible operand types", function.Name)
	}
	if destination && len(operation.TrapCategories) != 0 || !destination && !slices.Equal(operation.TrapCategories, []string{"bounds"}) {
		return fmt.Errorf("fixed-array index in %s has an invalid trap marker", function.Name)
	}
	return nil
}

func (e *analysisBuilder) inspectArrayIndexStore(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	receiver, index, value, err := threeOperands(operation.Operands)
	if err != nil {
		return fmt.Errorf("fixed-array index store in %s: %w", function.Name, err)
	}
	elementType, _, array := hir.FixedArrayParts(operation.OwnerType)
	if !array || operation.Type != elementType || operation.IndexType != hir.TypeUSize || operation.Operator != "[]=" || operation.DestinationLocal != mir.NoLocal {
		return fmt.Errorf("fixed-array index store in %s has incompatible owner, value, operator, or destination", function.Name)
	}
	receiverType, receiverErr := e.operandType(functionIndex, receiver)
	indexType, indexErr := e.operandType(functionIndex, index)
	valueType, valueErr := e.operandType(functionIndex, value)
	if receiverErr != nil || indexErr != nil || valueErr != nil || receiverType != operation.OwnerType || indexType != hir.TypeUSize || valueType != elementType {
		return fmt.Errorf("fixed-array index store in %s has incompatible operand types", function.Name)
	}
	if len(operation.TrapCategories) != 0 {
		return fmt.Errorf("fixed-array index store in %s has an invalid trap marker", function.Name)
	}
	return nil
}

func (e *analysisBuilder) inspectViewEmpty(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	receiver, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("borrowed-view emptiness in %s: %w", function.Name, err)
	}
	element, _, borrow := hir.BorrowElement(operation.OwnerType)
	if !borrow || element != hir.TypeStr && element != hir.TypeByteSlice && !hir.NonByteIndexedView(operation.OwnerType) || operation.Type != hir.TypeBool || operation.Field != "is_empty" {
		return fmt.Errorf("borrowed-view emptiness in %s has incompatible owner, result, or field", function.Name)
	}
	receiverType, err := e.operandType(functionIndex, receiver)
	if err != nil {
		return fmt.Errorf("borrowed-view emptiness in %s: %w", function.Name, err)
	}
	if receiverType != operation.OwnerType {
		return fmt.Errorf("borrowed-view emptiness in %s has receiver type %q, want %q", function.Name, receiverType, operation.OwnerType)
	}
	return nil
}

func (e *analysisBuilder) inspectViewIndex(functionIndex int, operation mir.Operation, destination bool) error {
	function := &e.functions[functionIndex]
	receiver, index, err := twoOperands(operation.Operands)
	if err != nil {
		return fmt.Errorf("byte-view index in %s: %w", function.Name, err)
	}
	element, indexed := hir.IndexedViewElement(operation.OwnerType)
	if !indexed || operation.IndexType != hir.TypeUSize || operation.Type != element || operation.Operator != "[]" ||
		destination != (operation.DestinationLocal != mir.NoLocal) {
		return fmt.Errorf("byte-view index in %s has incompatible owner, result, operator, or destination", function.Name)
	}
	receiverType, err := e.operandType(functionIndex, receiver)
	if err != nil {
		return fmt.Errorf("byte-view index in %s: %w", function.Name, err)
	}
	indexType, err := e.operandType(functionIndex, index)
	if err != nil {
		return fmt.Errorf("byte-view index in %s: %w", function.Name, err)
	}
	if receiverType != operation.OwnerType || indexType != operation.IndexType {
		return fmt.Errorf("byte-view index in %s has receiver type %q and index type %q", function.Name, receiverType, indexType)
	}
	if destination && len(operation.TrapCategories) != 0 || !destination && !slices.Equal(operation.TrapCategories, []string{"bounds"}) {
		return fmt.Errorf("byte-view index in %s has an invalid trap marker", function.Name)
	}
	return nil
}

func (e *analysisBuilder) inspectViewIndexStore(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	receiver, index, value, err := threeOperands(operation.Operands)
	if err != nil {
		return fmt.Errorf("mutable byte-view index store in %s: %w", function.Name, err)
	}
	element, indexed := hir.IndexedViewElement(operation.OwnerType)
	if !indexed || !hir.MutableIndexedView(operation.OwnerType) || operation.IndexType != hir.TypeUSize || operation.Type != element ||
		operation.Operator != "[]=" || operation.DestinationLocal != mir.NoLocal {
		return fmt.Errorf("mutable byte-view index store in %s has incompatible owner, value, operator, or destination", function.Name)
	}
	receiverType, receiverErr := e.operandType(functionIndex, receiver)
	indexType, indexErr := e.operandType(functionIndex, index)
	valueType, valueErr := e.operandType(functionIndex, value)
	if receiverErr != nil || indexErr != nil || valueErr != nil || receiverType != operation.OwnerType || indexType != hir.TypeUSize || valueType != element {
		return fmt.Errorf("mutable byte-view index store in %s has incompatible operand types", function.Name)
	}
	if len(operation.TrapCategories) != 0 {
		return fmt.Errorf("mutable byte-view index store in %s has an invalid trap marker", function.Name)
	}
	return nil
}

func isCIndexedViewType(value hir.Type) bool {
	_, indexed := hir.IndexedViewElement(value)
	return indexed
}

func (e *analysisBuilder) inspectViewSlice(functionIndex int, operation mir.Operation, destination bool) error {
	function := &e.functions[functionIndex]
	receiver, start, end, err := threeOperands(operation.Operands)
	if err != nil {
		return fmt.Errorf("immutable byte-view slice in %s: %w", function.Name, err)
	}
	wantView := operation.OwnerType
	if !isCIndexedViewType(wantView) || operation.IndexType != hir.TypeUSize || operation.Type != wantView || operation.Operator != ".." ||
		destination != (operation.DestinationLocal != mir.NoLocal) {
		return fmt.Errorf("immutable byte-view slice in %s has incompatible owner, result, operator, or destination", function.Name)
	}
	receiverType, receiverErr := e.operandType(functionIndex, receiver)
	startType, startErr := e.operandType(functionIndex, start)
	endType, endErr := e.operandType(functionIndex, end)
	if receiverErr != nil || startErr != nil || endErr != nil || receiverType != wantView || startType != hir.TypeUSize || endType != hir.TypeUSize {
		return fmt.Errorf("immutable byte-view slice in %s has incompatible operand types", function.Name)
	}
	if destination && len(operation.TrapCategories) != 0 || !destination && !slices.Equal(operation.TrapCategories, []string{"bounds"}) {
		return fmt.Errorf("immutable byte-view slice in %s has an invalid trap marker", function.Name)
	}
	return nil
}
