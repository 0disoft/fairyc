package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"strconv"
)

func (e *analysisBuilder) inspectScalarCast(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("value-preserving integer cast in %s: %w", function.Name, err)
	}
	if operation.DestinationLocal == mir.NoLocal || operation.Operator != "as" ||
		!hir.ValuePreservingIntegerCast(operation.OwnerType, operation.Type) {
		return fmt.Errorf("value-preserving integer cast in %s has an invalid shape", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("value-preserving integer cast in %s: %w", function.Name, err)
	}
	if operandType != operation.OwnerType {
		return fmt.Errorf("value-preserving integer cast in %s has operand type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return fmt.Errorf("value-preserving integer cast result in %s: %w", function.Name, err)
	}
	e.recordCScalarFeature(operation.OwnerType)
	e.recordCScalarFeature(operation.Type)
	return nil
}

func (e *analysisBuilder) inspectBinary(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	left, right, err := twoOperands(operation.Operands)
	if err != nil {
		return fmt.Errorf("typed scalar binary operation in %s: %w", function.Name, err)
	}
	if operation.DestinationLocal == mir.NoLocal || !supportedBinary(operation.Operator) {
		return fmt.Errorf("typed scalar binary operation in %s has an invalid destination or operator", function.Name)
	}
	validShape := false
	switch operation.Operator {
	case "==", "!=":
		_, _, pointer := hir.RawPointerElement(operation.OwnerType)
		validShape = (hir.ScalarEqualityType(operation.OwnerType) || pointer) && operation.Type == hir.TypeBool
	case "<", "<=", ">", ">=":
		validShape = hir.ScalarOrderedType(operation.OwnerType) && operation.Type == hir.TypeBool
	case "+", "-", "*":
		validShape = hir.CheckedArithmeticType(operation.OwnerType) && operation.Type == operation.OwnerType
	case "/", "%":
		validShape = hir.CheckedDivisionType(operation.OwnerType) && operation.Type == operation.OwnerType
	case ">>", "<<":
		validShape = hir.CheckedShiftType(operation.OwnerType) && operation.Type == operation.OwnerType
	case "&", "^", "|":
		validShape = hir.BitwiseIntegerType(operation.OwnerType) && operation.Type == operation.OwnerType
	}
	if !validShape {
		return fmt.Errorf("typed scalar binary operation in %s has incompatible owner or result type", function.Name)
	}
	leftType, err := e.operandType(functionIndex, left)
	if err != nil {
		return fmt.Errorf("typed scalar binary left operand in %s: %w", function.Name, err)
	}
	rightType, err := e.operandType(functionIndex, right)
	if err != nil {
		return fmt.Errorf("typed scalar binary right operand in %s: %w", function.Name, err)
	}
	if leftType != operation.OwnerType || rightType != operation.OwnerType {
		return fmt.Errorf("typed scalar binary operation in %s has operand types %q and %q, want %q", function.Name, leftType, rightType, operation.OwnerType)
	}
	return nil
}

func (e *analysisBuilder) inspectUnary(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("typed scalar unary operation in %s: %w", function.Name, err)
	}
	if operation.DestinationLocal == mir.NoLocal || !supportedUnary(operation.Operator) {
		return fmt.Errorf("typed scalar unary operation in %s has an invalid destination or operator", function.Name)
	}
	if operation.Operator == "&" || operation.Operator == "&mut" {
		if operand.Kind != mir.OperandLocal {
			return fmt.Errorf("safe borrow in %s does not name a local place", function.Name)
		}
		wantType := hir.SharedBorrow(operation.OwnerType)
		if operation.Operator == "&mut" {
			wantType = hir.MutableBorrow(operation.OwnerType)
		}
		if operation.Type != wantType {
			return fmt.Errorf("safe borrow in %s has type %q, want %q", function.Name, operation.Type, wantType)
		}
		if _, ok := e.safeBorrowCType(operation.Type); !ok {
			return CapabilityError{Function: function.Name, Feature: "safe borrow type " + strconv.Quote(operation.Type.String())}
		}
		operandType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return fmt.Errorf("safe borrow in %s: %w", function.Name, err)
		}
		if operandType != operation.OwnerType {
			return fmt.Errorf("safe borrow in %s has operand type %q, want %q", function.Name, operandType, operation.OwnerType)
		}
		return nil
	}
	if operation.Operator == "*" {
		element, _, borrow := hir.BorrowElement(operation.OwnerType)
		if !borrow || operation.Type != element {
			return fmt.Errorf("safe dereference in %s has incompatible owner or result type", function.Name)
		}
		operandType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return fmt.Errorf("safe dereference in %s: %w", function.Name, err)
		}
		if operandType != operation.OwnerType {
			return fmt.Errorf("safe dereference in %s has operand type %q, want %q", function.Name, operandType, operation.OwnerType)
		}
		return nil
	}
	validShape := false
	switch operation.Operator {
	case "!":
		validShape = operation.OwnerType == hir.TypeBool && operation.Type == hir.TypeBool
	case "+", "-":
		validShape = (operation.OwnerType == hir.TypeI32 || operation.OwnerType == hir.TypeI64) && operation.Type == operation.OwnerType
	case "~":
		validShape = hir.BitwiseNotType(operation.OwnerType) && operation.Type == operation.OwnerType
	}
	if !validShape {
		return fmt.Errorf("typed scalar unary operation in %s has incompatible owner or result type", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("typed scalar unary operand in %s: %w", function.Name, err)
	}
	if operandType != operation.OwnerType {
		return fmt.Errorf("typed scalar unary operation in %s has operand type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	return nil
}
