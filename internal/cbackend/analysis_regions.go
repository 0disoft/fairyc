package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/cstdlibcontract"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"strconv"
)

func (e *analysisBuilder) inspectRegionOperation(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	switch operation.Kind {
	case mir.OperationRegionEnter:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandLocal {
			return fmt.Errorf("Region entry in %s has no Arena owner local", function.Name)
		}
		ownerType, err := e.operandType(functionIndex, operand)
		if err != nil || ownerType != operation.OwnerType || operation.Type != hir.MutableBorrow(operation.OwnerType) {
			return fmt.Errorf("Region entry in %s has incompatible Arena ownership", function.Name)
		}
		if _, found := e.resourceByType[operation.OwnerType]; !found {
			return CapabilityError{Function: function.Name, Feature: "Region Arena type " + strconv.Quote(operation.OwnerType.String())}
		}
	case mir.OperationRegionPlace:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Region placement in %s: %w", function.Name, err)
		}
		ownerType, err := e.operandType(functionIndex, operand)
		if err != nil || ownerType != operation.OwnerType || e.supportedNeedsDropType(operation.OwnerType) {
			return CapabilityError{Function: function.Name, Feature: "Region placement type " + strconv.Quote(operation.OwnerType.String())}
		}
		successType, failureType, feature, supported := e.resultCSpec(operation.Type)
		placedType, mutable, borrow := hir.BorrowElement(successType)
		errorIndex, errorStruct := e.structByType[failureType]
		payloadValid := borrow && !mutable && placedType == operation.OwnerType && operation.Operator == ""
		if operation.Operator == "alloc_zeroed_bytes" {
			payloadValid = successType == hir.MutableBorrow(hir.TypeByteSlice) && operation.OwnerType == hir.TypeUSize
		}
		if operation.Operator == "copy_bytes" {
			payloadValid = successType == hir.MutableBorrow(hir.TypeByteSlice) && operation.OwnerType == hir.SharedBorrow(hir.TypeByteSlice)
		}
		if !supported || !payloadValid || !errorStruct ||
			len(e.structTypes[errorIndex].definition.Fields) != 1 || e.structTypes[errorIndex].definition.Fields[0].Type != hir.TypeU32 {
			return CapabilityError{Function: function.Name, Feature: "fallible Region placement result " + strconv.Quote(operation.Type.String())}
		}
		e.features[feature] = struct{}{}
	case mir.OperationRegionReset:
		if len(operation.Operands) != 0 {
			return fmt.Errorf("Region reset in %s has operands", function.Name)
		}
	default:
		return fmt.Errorf("inspect Region operation in %s received %q", function.Name, operation.Kind)
	}
	e.features["region-lifecycle"] = struct{}{}
	e.trapCategories["unreachable"] = struct{}{}
	symbols := []string{"fy_std_region_begin_v0", "fy_std_region_allocate_zeroed_v0", "fy_std_region_reset_v0"}
	if operation.Operator == "copy_bytes" {
		symbols = append(symbols, "fy_std_region_copy_bytes_v0")
	}
	for _, symbol := range symbols {
		if !cstdlibcontract.AdmittedForTarget(symbol, e.targetID) {
			return CapabilityError{Function: function.Name, Feature: "Region runtime symbol " + strconv.Quote(symbol)}
		}
		e.usedForeignSymbols[symbol] = struct{}{}
	}
	return nil
}
