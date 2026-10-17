package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"strconv"
)

func (e *analysisBuilder) inspectRawAddress(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("raw address in %s: %w", function.Name, err)
	}
	if operand.Kind != mir.OperandLocal || operation.Entity == "" || operation.UnsafeScope.Source == "" {
		return fmt.Errorf("raw address in %s has incomplete place or unsafe provenance", function.Name)
	}
	element, mutable, pointer := hir.RawPointerElement(operation.Type)
	if !pointer || element != operation.OwnerType && element != hir.TypeVoid || operation.Operator != "&" && operation.Operator != "&mut" || operation.Operator == "&" && mutable {
		return fmt.Errorf("raw address in %s has incompatible source and pointer types", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("raw address in %s: %w", function.Name, err)
	}
	if canonicalCValueType(operandType) != canonicalCValueType(operation.OwnerType) {
		return fmt.Errorf("raw address in %s has place type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return CapabilityError{Function: function.Name, Feature: "raw address type " + strconv.Quote(operation.Type.String())}
	}
	return nil
}

func (e *analysisBuilder) inspectRawBorrowPointer(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("raw borrow pointer in %s: %w", function.Name, err)
	}
	if operand.Kind != mir.OperandLocal || operation.DestinationLocal == mir.NoLocal ||
		operation.Entity == "" || operation.UnsafeScope.Source == "" || operation.Operator != "as" {
		return fmt.Errorf("raw borrow pointer in %s has incomplete operand or unsafe provenance", function.Name)
	}
	sourceElement, sourceMutable, sourceBorrow := hir.BorrowElement(operation.OwnerType)
	targetElement, targetMutable, targetPointer := hir.RawPointerElement(operation.Type)
	if !sourceBorrow || !targetPointer || (targetElement != sourceElement && targetElement != hir.TypeVoid) ||
		(!sourceMutable && targetMutable) {
		return fmt.Errorf("raw borrow pointer in %s has incompatible source and pointer types", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("raw borrow pointer in %s: %w", function.Name, err)
	}
	if operandType != operation.OwnerType {
		return fmt.Errorf("raw borrow pointer in %s has operand type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	if _, ok := e.safeBorrowCType(operation.OwnerType); !ok {
		return CapabilityError{Function: function.Name, Feature: "raw borrow owner type " + strconv.Quote(operation.OwnerType.String())}
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return CapabilityError{Function: function.Name, Feature: "raw borrow pointer type " + strconv.Quote(operation.Type.String())}
	}
	return nil
}

func (e *analysisBuilder) inspectBorrowedViewFromRawParts(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	if len(operation.Operands) != 3 || operation.Entity == "" || operation.UnsafeScope.Source == "" {
		return fmt.Errorf("borrowed-view raw-parts construction in %s has incomplete operands or unsafe provenance", function.Name)
	}
	_, ownerMutable, ownerBorrow := hir.BorrowElement(operation.OwnerType)
	wantType := hir.SharedBorrow(hir.TypeStr)
	wantOperator := "str::from_raw_parts"
	wantOwnerMutable := false
	wantDataMutable := false
	if operation.Kind == mir.OperationCStrFromRawParts {
		wantType = hir.SharedBorrow(hir.TypeCStr)
		wantOperator = "CStr::from_raw_parts"
	} else if operation.Kind == mir.OperationByteSliceFromRawParts {
		wantType = hir.SharedBorrow(hir.TypeByteSlice)
		wantOperator = "byte_slice::from_raw_parts"
	} else if operation.Kind == mir.OperationMutByteSliceFromRawParts {
		wantType = hir.MutableBorrow(hir.TypeByteSlice)
		wantOperator = "byte_slice::from_raw_parts_mut"
		wantOwnerMutable = true
		wantDataMutable = true
	}
	if !ownerBorrow || ownerMutable != wantOwnerMutable || operation.Type != wantType || operation.Operator != wantOperator {
		return fmt.Errorf("borrowed-view raw-parts construction in %s has incompatible owner or result type", function.Name)
	}
	ownerType, err := e.operandType(functionIndex, operation.Operands[0])
	if err != nil || ownerType != operation.OwnerType {
		return fmt.Errorf("borrowed-view raw-parts owner in %s has type %q, want %q", function.Name, ownerType, operation.OwnerType)
	}
	dataType, err := e.operandType(functionIndex, operation.Operands[1])
	if err != nil {
		return fmt.Errorf("borrowed-view raw-parts data in %s: %w", function.Name, err)
	}
	dataElement, dataMutable, dataPointer := hir.RawPointerElement(dataType)
	if !dataPointer || dataElement != hir.TypeU8 && dataElement != hir.TypeVoid || wantDataMutable && !dataMutable {
		return fmt.Errorf("borrowed-view raw-parts data in %s has incompatible type %q", function.Name, dataType)
	}
	lengthType, err := e.operandType(functionIndex, operation.Operands[2])
	if err != nil || lengthType != hir.TypeUSize {
		return fmt.Errorf("borrowed-view raw-parts length in %s has type %q, want %q", function.Name, lengthType, hir.TypeUSize)
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return CapabilityError{Function: function.Name, Feature: "borrowed-view raw-parts result type " + strconv.Quote(operation.Type.String())}
	}
	return nil
}

func (e *analysisBuilder) inspectByteSliceDataAddress(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("byte-slice data address in %s: %w", function.Name, err)
	}
	if operand.Kind != mir.OperandByteString && operand.Kind != mir.OperandLocal || operation.Entity == "" || operation.UnsafeScope.Source == "" {
		return fmt.Errorf("byte-slice data address in %s has incomplete storage or unsafe provenance", function.Name)
	}
	element, mutable, pointer := hir.RawPointerElement(operation.Type)
	ownerElement, ownerMutable, ownerBorrow := hir.BorrowElement(operation.OwnerType)
	staticStorageValid := operand.Kind != mir.OperandByteString || !ownerMutable
	if !pointer || element != hir.TypeU8 || !ownerBorrow || ownerElement != hir.TypeByteSlice ||
		!staticStorageValid || (!ownerMutable && mutable) || operation.Operator != "as" {
		return fmt.Errorf("byte-slice data address in %s has incompatible source and pointer types", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("byte-slice data address in %s: %w", function.Name, err)
	}
	if operandType != operation.OwnerType {
		return fmt.Errorf("byte-slice data address in %s has storage type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return CapabilityError{Function: function.Name, Feature: "byte-slice data address type " + strconv.Quote(operation.Type.String())}
	}
	return nil
}

func (e *analysisBuilder) inspectStrDataAddress(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("string data address in %s: %w", function.Name, err)
	}
	if operand.Kind != mir.OperandString && operand.Kind != mir.OperandLocal || operation.Entity == "" || operation.UnsafeScope.Source == "" {
		return fmt.Errorf("string data address in %s has incomplete storage or unsafe provenance", function.Name)
	}
	element, mutable, pointer := hir.RawPointerElement(operation.Type)
	if !pointer || mutable || element != hir.TypeU8 || operation.OwnerType != hir.SharedBorrow(hir.TypeStr) || operation.Operator != "as" {
		return fmt.Errorf("string data address in %s has incompatible source and pointer types", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("string data address in %s: %w", function.Name, err)
	}
	if operandType != operation.OwnerType {
		return fmt.Errorf("string data address in %s has storage type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return CapabilityError{Function: function.Name, Feature: "string data address type " + strconv.Quote(operation.Type.String())}
	}
	return nil
}

func (e *analysisBuilder) inspectCStrDataAddress(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("C string data address in %s: %w", function.Name, err)
	}
	if operand.Kind != mir.OperandCString && operand.Kind != mir.OperandLocal || operation.Entity == "" || operation.UnsafeScope.Source == "" {
		return fmt.Errorf("C string data address in %s has incomplete storage or unsafe provenance", function.Name)
	}
	element, mutable, pointer := hir.RawPointerElement(operation.Type)
	if !pointer || mutable || element != hir.TypeU8 || operation.OwnerType != hir.SharedBorrow(hir.TypeCStr) || operation.Operator != "as" {
		return fmt.Errorf("C string data address in %s has incompatible source and pointer types", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("C string data address in %s: %w", function.Name, err)
	}
	if operandType != operation.OwnerType {
		return fmt.Errorf("C string data address in %s has storage type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return CapabilityError{Function: function.Name, Feature: "C string data address type " + strconv.Quote(operation.Type.String())}
	}
	return nil
}
