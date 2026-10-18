package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"strconv"
)

func (e *analysisBuilder) inspectFunction(functionIndex int) error {
	if e.ctx != nil {
		if err := e.ctx.Err(); err != nil {
			return err
		}
	}
	function := &e.functions[functionIndex]
	predecessors := make([]uint32, len(function.Blocks))
	for _, block := range function.Blocks {
		for _, successor := range block.Terminator.Successors {
			if successor >= 0 && int(successor) < len(predecessors) {
				predecessors[successor]++
			}
		}
	}
	if function.ResultType != hir.TypeVoid && !e.supportedCValueType(function.ResultType) {
		return CapabilityError{Function: function.Name, Feature: "result type " + strconv.Quote(function.ResultType.String())}
	}
	e.recordCValueTypeRequirements(function.ResultType)
	e.recordCScalarFeature(function.ResultType)
	if len(function.ParameterTypes) != len(function.Parameters) {
		return fmt.Errorf("MIR function %s has %d parameter types for %d parameters", function.Name, len(function.ParameterTypes), len(function.Parameters))
	}
	for _, parameterType := range function.ParameterTypes {
		if !e.supportedCValueType(parameterType) {
			return CapabilityError{Function: function.Name, Feature: "parameter type " + strconv.Quote(parameterType.String())}
		}
		e.recordCValueTypeRequirements(parameterType)
		e.recordCScalarFeature(parameterType)
	}
	for _, resource := range function.Resources {
		if !e.supportedNeedsDropType(resource.Type) {
			return CapabilityError{Function: function.Name, Feature: "resource local type " + strconv.Quote(resource.Type.String())}
		}
	}
	if len(function.DropFlags) != 0 {
		return CapabilityError{Function: function.Name, Feature: "partial resource initialization"}
	}
	if len(function.ReturnOrigins) != 0 {
		return CapabilityError{Function: function.Name, Feature: "safe-borrow returns"}
	}
	allowedUnsafeEntities := make(map[string]struct{})
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		if block.ID != mir.BlockID(blockIndex) {
			return fmt.Errorf("C emission requires canonical block ids in %s: block index %d has id %d", function.Name, blockIndex, block.ID)
		}
		for operationIndex := range block.Operations {
			if e.ctx != nil {
				if err := e.ctx.Err(); err != nil {
					return err
				}
			}
			operation := &block.Operations[operationIndex]
			e.recordCValueTypeRequirements(operation.Type)
			if err := validateOperationShape(function.Name, *operation); err != nil {
				return err
			}
			switch operation.Kind {
			case mir.OperationAssign:
				if err := e.inspectAssignment(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationInitialize:
				if err := e.inspectResourceInitialize(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationMove, mir.OperationDrop, mir.OperationBurn:
				if err := e.inspectResourceDischarge(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationRawAddress:
				if err := e.inspectRawAddress(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["raw-address"] = struct{}{}
			case mir.OperationRawBorrowPointer:
				if err := e.inspectRawBorrowPointer(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["raw-borrow-pointer"] = struct{}{}
			case mir.OperationStrFromRawParts, mir.OperationCStrFromRawParts, mir.OperationByteSliceFromRawParts, mir.OperationMutByteSliceFromRawParts:
				if err := e.inspectBorrowedViewFromRawParts(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				if operation.Kind == mir.OperationStrFromRawParts {
					e.features["utf8-string-view"] = struct{}{}
				} else if operation.Kind == mir.OperationCStrFromRawParts {
					e.features["c-string-view"] = struct{}{}
				} else if operation.Kind == mir.OperationByteSliceFromRawParts {
					e.features["byte-slice-view"] = struct{}{}
				} else {
					e.features["mutable-byte-slice-view"] = struct{}{}
				}
			case mir.OperationStrDataAddress:
				if err := e.inspectStrDataAddress(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["str-data-address"] = struct{}{}
			case mir.OperationByteSliceDataAddress:
				if err := e.inspectByteSliceDataAddress(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["byte-slice-data-address"] = struct{}{}
			case mir.OperationCStrDataAddress:
				if err := e.inspectCStrDataAddress(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["cstr-data-address"] = struct{}{}
			case mir.OperationStructConstruct:
				if err := e.inspectStructOperation(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationEnumConstruct, mir.OperationEnumPayload:
				if err := e.inspectEnumOperation(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationResourceConstruct:
				if err := e.inspectResourceOperation(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationFieldStore:
				if err := e.inspectFieldStore(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationDerefStore:
				if err := e.inspectDerefStore(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationFieldBorrow:
				if err := e.inspectFieldBorrow(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationFieldRead:
				if _, structure := e.structByType[fieldAggregateType(operation.OwnerType)]; structure {
					if err := e.inspectStructOperation(functionIndex, *operation); err != nil {
						return err
					}
				} else if err := e.inspectResourceOperation(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationViewLength:
				if err := e.inspectViewLength(functionIndex, *operation); err != nil {
					return err
				}
				e.features["view-length"] = struct{}{}
			case mir.OperationViewEmpty:
				if err := e.inspectViewEmpty(functionIndex, *operation); err != nil {
					return err
				}
				e.features["view-empty"] = struct{}{}
			case mir.OperationViewIndexCheck:
				if err := e.inspectViewIndex(functionIndex, *operation, false); err != nil {
					return err
				}
				e.features["view-index"] = struct{}{}
			case mir.OperationViewIndex:
				if err := e.inspectViewIndex(functionIndex, *operation, true); err != nil {
					return err
				}
				e.features["view-index"] = struct{}{}
			case mir.OperationViewElementBorrow:
				if !mir.ValidViewElementBorrowShape(*operation) {
					return fmt.Errorf("invalid indexed element borrow")
				}
				check := *operation
				check.Kind = mir.OperationViewIndex
				check.Type, _, _ = hir.BorrowElement(operation.Type)
				if err := e.inspectViewIndex(functionIndex, check, true); err != nil {
					return err
				}
				e.features["view-element-borrow"] = struct{}{}
			case mir.OperationViewIndexStore:
				if err := e.inspectViewIndexStore(functionIndex, *operation); err != nil {
					return err
				}
				e.features["mutable-view-index-store"] = struct{}{}
			case mir.OperationViewSliceCheck:
				if err := e.inspectViewSlice(functionIndex, *operation, false); err != nil {
					return err
				}
				e.features["view-slice"] = struct{}{}
			case mir.OperationViewSlice:
				if err := e.inspectViewSlice(functionIndex, *operation, true); err != nil {
					return err
				}
				e.features["view-slice"] = struct{}{}
			case mir.OperationSharedReborrow:
				if !mir.ValidSharedReborrowShape(*operation) {
					return fmt.Errorf("invalid shared reborrow in %s", e.functions[functionIndex].Name)
				}
				operandType, err := e.operandType(functionIndex, operation.Operands[0])
				if err != nil || operandType != operation.OwnerType {
					return fmt.Errorf("shared reborrow source type mismatch")
				}
				e.features["shared-reborrow"] = struct{}{}
			case mir.OperationScalarCast:
				if err := e.inspectScalarCast(functionIndex, *operation); err != nil {
					return err
				}
				e.features["scalar-cast"] = struct{}{}
			case mir.OperationArrayConstruct:
				if err := e.inspectArrayConstruction(functionIndex, *operation); err != nil {
					return err
				}
				e.features["scalar-fixed-array"] = struct{}{}
			case mir.OperationArrayBorrowView:
				if err := e.inspectArrayBorrowView(functionIndex, *operation); err != nil {
					return err
				}
				if operation.Type == hir.MutableBorrow(hir.TypeByteSlice) {
					e.features["mutable-array-byte-view-borrow"] = struct{}{}
				} else if hir.I64View(operation.Type) {
					e.features["array-i64-view-borrow"] = struct{}{}
				} else if hir.NonByteIndexedView(operation.Type) {
					e.features["array-integer-view-borrow"] = struct{}{}
				} else {
					e.features["array-byte-view-borrow"] = struct{}{}
				}
			case mir.OperationStorageLayout:
				if _, err := e.storageQueryValue(*operation); err != nil {
					return err
				}
				e.features["private-storage-layout"] = struct{}{}
			case mir.OperationStorageTake:
				if err := e.inspectStorageTake(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["private-storage-take"] = struct{}{}
			case mir.OperationStoragePut:
				if err := e.inspectStoragePut(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["private-storage-put"] = struct{}{}
			case mir.OperationStorageBorrow, mir.OperationStorageBorrowMut:
				if err := e.inspectStorageBorrow(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["private-"+operation.Kind.String()] = struct{}{}
			case mir.OperationStorageElementAddress:
				if err := e.inspectStorageElementAddress(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["private-storage-element-address"] = struct{}{}
			case mir.OperationArrayLength:
				if err := e.inspectArrayLength(functionIndex, *operation); err != nil {
					return err
				}
				e.features["array-length"] = struct{}{}
			case mir.OperationArrayIndexCheck:
				if err := e.inspectArrayIndex(functionIndex, *operation, false); err != nil {
					return err
				}
				e.features["array-index"] = struct{}{}
			case mir.OperationArrayIndex:
				if err := e.inspectArrayIndex(functionIndex, *operation, true); err != nil {
					return err
				}
				e.features["array-index"] = struct{}{}
			case mir.OperationArrayIndexStore:
				if err := e.inspectArrayIndexStore(functionIndex, *operation); err != nil {
					return err
				}
				e.features["array-index-store"] = struct{}{}
			case mir.OperationRegionEnter, mir.OperationRegionPlace, mir.OperationRegionReset:
				if err := e.inspectRegionOperation(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationUse:
			case mir.OperationArithmeticCheck:
				e.features["checked-arithmetic"] = struct{}{}
				if operation.OwnerType == hir.TypeI32 {
					e.needsBitcast = true
					if operation.Operator == ">>" {
						e.needsShift = true
					}
				}
			case mir.OperationCheckedDivision:
				e.features["checked-division"] = struct{}{}
			case mir.OperationUnary:
				if err := e.inspectUnary(functionIndex, *operation); err != nil {
					return err
				}
				if operation.OwnerType == hir.TypeI64 && (operation.Operator == "-" || operation.Operator == "~") {
					e.i64Helpers.bitcast = true
				}
				if operation.OwnerType == hir.TypeI32 && (operation.Operator == "-" || operation.Operator == "~") {
					e.needsBitcast = true
				}
				if operation.Operator == "~" {
					e.features["scalar-bitwise"] = struct{}{}
				}
			case mir.OperationBinary:
				if err := e.inspectBinary(functionIndex, *operation); err != nil {
					return err
				}
				if operation.OwnerType == hir.TypeI64 {
					switch operation.Operator {
					case "+", "-", "*", "&", "^", "|", "<<":
						e.i64Helpers.bitcast = true
					case ">>":
						e.i64Helpers.shift = true
					}
				}
				switch operation.Operator {
				case "==", "!=", "<", "<=", ">", ">=":
					e.features["scalar-comparison"] = struct{}{}
				case "+", "-", "*":
					if operation.OwnerType == hir.TypeI32 {
						e.needsBitcast = true
					}
				case "&", "^", "|":
					e.features["scalar-bitwise"] = struct{}{}
					if operation.OwnerType == hir.TypeI32 {
						e.needsBitcast = true
					}
				case ">>", "<<":
					e.features["checked-shift"] = struct{}{}
					if operation.OwnerType == hir.TypeI32 {
						e.needsBitcast = true
						if operation.Operator == ">>" {
							e.needsShift = true
						}
					}
				}
			case mir.OperationResultConstruct, mir.OperationResultPayload,
				mir.OperationOptionConstruct, mir.OperationOptionPayload, mir.OperationResultSwitch:
				if err := e.inspectTaggedOperation(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationDropFlagInit, mir.OperationDropFlagClear:
				return CapabilityError{Function: function.Name, Feature: "MIR operation " + strconv.Quote(operation.Kind.String())}
			default:
				return CapabilityError{Function: function.Name, Feature: "MIR operation " + strconv.Quote(operation.Kind.String())}
			}
		}
		terminator := block.Terminator
		for _, successor := range terminator.Successors {
			if successor < 0 || int(successor) >= len(function.Blocks) {
				return fmt.Errorf("C emission successor %d is outside %s", successor, function.Name)
			}
		}
		switch terminator.Kind {
		case mir.TerminatorGoto:
			e.features["scalar-control-flow"] = struct{}{}
		case mir.TerminatorCall:
			e.features["scalar-call"] = struct{}{}
			calleeResult, calleeParameters, foreign, found := e.callSignature(terminator.Callee)
			if !found {
				return fmt.Errorf("C call in %s names missing function symbol %d", function.Name, terminator.Callee)
			}
			if foreign {
				e.features["foreign-c-call"] = struct{}{}
				e.usedForeignSymbols[e.foreignFunctions[e.foreignBySymbol[terminator.Callee]].LinkName] = struct{}{}
				allowedUnsafeEntities[terminator.Entity] = struct{}{}
				if terminator.ForeignABI != "C" || terminator.UnsafeScope.Source == "" {
					return fmt.Errorf("foreign C call in %s lacks ABI or unsafe provenance", function.Name)
				}
			} else if terminator.ForeignABI != "" {
				return fmt.Errorf("internal call in %s carries foreign ABI %q", function.Name, terminator.ForeignABI)
			}
			if calleeResult == hir.TypeVoid && terminator.DestinationLocal != mir.NoLocal {
				return fmt.Errorf("void C call in %s has destination local %d", function.Name, terminator.DestinationLocal)
			}
			if calleeResult != hir.TypeVoid && terminator.DestinationLocal < 0 {
				return fmt.Errorf("value C call in %s has no destination local", function.Name)
			}
			if len(terminator.Arguments) != len(calleeParameters) {
				return fmt.Errorf("C call in %s passes %d arguments, want %d", function.Name, len(terminator.Arguments), len(calleeParameters))
			}
			for argumentIndex, argument := range terminator.Arguments {
				argumentType, err := e.operandType(functionIndex, argument)
				if err != nil {
					return fmt.Errorf("call argument %d in %s: %w", argumentIndex, function.Name, err)
				}
				if canonicalCValueType(argumentType) != canonicalCValueType(calleeParameters[argumentIndex]) {
					return fmt.Errorf("call argument %d in %s has type %q, want %q", argumentIndex, function.Name, argumentType, calleeParameters[argumentIndex])
				}
			}
		case mir.TerminatorSwitch:
			e.features["scalar-control-flow"] = struct{}{}
			if err := e.inspectSwitch(functionIndex, terminator, predecessors); err != nil {
				return err
			}
		case mir.TerminatorReturn:
			if len(terminator.Successors) != 0 {
				return fmt.Errorf("return in %s has successors", function.Name)
			}
			if function.ResultType == hir.TypeVoid && terminator.ReturnValue.Kind != mir.OperandInvalid {
				return fmt.Errorf("void return in %s carries a value", function.Name)
			}
			if function.ResultType != hir.TypeVoid {
				if _, err := cOperand(terminator.ReturnValue); err != nil {
					return fmt.Errorf("return in %s: %w", function.Name, err)
				}
				returnType, err := e.operandType(functionIndex, terminator.ReturnValue)
				if err != nil {
					return fmt.Errorf("return in %s: %w", function.Name, err)
				}
				if canonicalCValueType(returnType) != canonicalCValueType(function.ResultType) {
					return fmt.Errorf("return in %s has type %q, want %q", function.Name, returnType, function.ResultType)
				}
			}
		case mir.TerminatorTrap:
			if len(terminator.Successors) != 0 {
				return fmt.Errorf("trap in %s has successors", function.Name)
			}
			e.features["trap"] = struct{}{}
			if err := e.recordTrapCategories(function.Name, terminator.TrapCategories); err != nil {
				return err
			}
		case mir.TerminatorUnreachable:
			if len(terminator.Successors) != 0 {
				return fmt.Errorf("unreachable terminator in %s has successors", function.Name)
			}
			e.features["trap"] = struct{}{}
			e.trapCategories["unreachable"] = struct{}{}
		default:
			return CapabilityError{Function: function.Name, Feature: "MIR terminator " + strconv.Quote(string(terminator.Kind))}
		}
	}
	for _, operation := range function.UnsafeOperations {
		if _, admitted := allowedUnsafeEntities[operation.Entity]; !admitted {
			return CapabilityError{Function: function.Name, Feature: "unsafe operation outside the admitted constrained-C subset"}
		}
	}
	return nil
}

func (e *analysisBuilder) recordCValueTypeRequirements(value hir.Type) {
	if payload, option := hir.OptionElement(value); option {
		e.recordCValueTypeRequirements(payload)
	} else if success, failure, result := hir.ResultElements(value); result {
		e.recordCValueTypeRequirements(success)
		e.recordCValueTypeRequirements(failure)
	}
	if _, structView := hir.StructViewElement(value); structView {
		if hir.MutableIndexedView(value) {
			e.needsMutStructSlice = true
		} else {
			e.needsStructSlice = true
		}
	}
	if _, nominal := hir.StructViewElement(value); hir.NonByteIndexedView(value) && !hir.I64View(value) && !nominal {
		if e.integerViews == nil {
			e.integerViews = make(map[hir.Type]bool)
		}
		e.integerViews[value] = true
	}
	switch canonicalCValueType(value) {
	case hir.SharedBorrow(hir.TypeByteSlice):
		e.needsByteSlice = true
	case hir.SharedBorrow(hir.TypeI64Slice):
		e.needsI64Slice = true
	case hir.MutableBorrow(hir.TypeI64Slice):
		e.needsMutI64Slice = true
	case hir.MutableBorrow(hir.TypeByteSlice):
		e.needsMutByteSlice = true
	case hir.SharedBorrow(hir.TypeStr):
		e.needsStringView = true
	}
}

func (e *emissionAnalysis) callSignature(symbol hir.SymbolID) (hir.Type, []hir.Type, bool, bool) {
	if index, found := e.bySymbol[symbol]; found {
		function := &e.functions[index]
		return function.ResultType, function.ParameterTypes, false, true
	}
	if index, found := e.foreignBySymbol[symbol]; found {
		function := &e.foreignFunctions[index]
		return function.ResultType, function.ParameterTypes, true, true
	}
	return hir.TypeInvalid, nil, false, false
}
