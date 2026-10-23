package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"slices"
	"unicode/utf8"
)

func (e *emissionAnalysis) inferLocalTypes(functionIndex int) (map[mir.LocalID]hir.Type, error) {
	function := &e.functions[functionIndex]
	localTypes := make(map[mir.LocalID]hir.Type)
	for index, parameter := range function.Parameters {
		if index >= len(function.ParameterTypes) {
			return nil, fmt.Errorf("MIR function %s has %d parameter types for %d parameters", function.Name, len(function.ParameterTypes), len(function.Parameters))
		}
		if _, err := assignLocalType(localTypes, parameter, function.ParameterTypes[index]); err != nil {
			return nil, fmt.Errorf("parameter type in %s: %w", function.Name, err)
		}
	}
	for _, resource := range function.Resources {
		if _, err := assignLocalType(localTypes, mir.LocalID(resource.Symbol), resource.Type); err != nil {
			return nil, fmt.Errorf("resource local type in %s: %w", function.Name, err)
		}
	}
	changed := true
	for changed {
		changed = false
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				var inferred hir.Type
				switch operation.Kind {
				case mir.OperationAssign, mir.OperationInitialize:
					if len(operation.Operands) == 1 {
						inferred, _ = inferredOperandType(localTypes, operation.Operands[0])
					}
				case mir.OperationUnary:
					inferred = operation.Type
				case mir.OperationBinary:
					inferred = operation.Type
				case mir.OperationResultConstruct, mir.OperationOptionConstruct,
					mir.OperationResultPayload, mir.OperationOptionPayload,
					mir.OperationEnumConstruct, mir.OperationEnumPayload,
					mir.OperationRawAddress, mir.OperationRawBorrowPointer, mir.OperationStrFromRawParts, mir.OperationCStrFromRawParts,
					mir.OperationByteSliceFromRawParts, mir.OperationMutByteSliceFromRawParts,
					mir.OperationStrDataAddress, mir.OperationByteSliceDataAddress, mir.OperationCStrDataAddress,
					mir.OperationStructConstruct, mir.OperationResourceConstruct, mir.OperationFieldRead, mir.OperationFieldBorrow, mir.OperationViewLength, mir.OperationViewEmpty, mir.OperationViewIndex, mir.OperationViewElementBorrow, mir.OperationViewSlice,
					mir.OperationCCallback, mir.OperationStorageLayout, mir.OperationStorageTake, mir.OperationStorageBorrow, mir.OperationStorageBorrowMut, mir.OperationStorageElementAddress, mir.OperationScalarCast, mir.OperationSharedReborrow, mir.OperationArrayConstruct, mir.OperationArrayBorrowView, mir.OperationArrayLength, mir.OperationArrayIndex,
					mir.OperationRegionPlace:
					inferred = operation.Type
				}
				if operation.DestinationLocal == mir.NoLocal || inferred == hir.TypeInvalid || inferred == hir.ParseType("") {
					continue
				}
				added, err := assignLocalType(localTypes, operation.DestinationLocal, inferred)
				if err != nil {
					return nil, fmt.Errorf("operation %s in %s: %w", operation.Kind, function.Name, err)
				}
				changed = changed || added
			}
			terminator := block.Terminator
			if terminator.Kind == mir.TerminatorCall && terminator.DestinationLocal != mir.NoLocal {
				resultType, found := e.callResultType(terminator.Callee)
				if !found {
					return nil, fmt.Errorf("C call in %s names missing function symbol %d", function.Name, terminator.Callee)
				}
				added, err := assignLocalType(localTypes, terminator.DestinationLocal, resultType)
				if err != nil {
					return nil, fmt.Errorf("call result in %s: %w", function.Name, err)
				}
				changed = changed || added
			}
		}
	}
	for _, local := range collectLocals(*function) {
		if _, found := localTypes[local]; !found {
			localTypes[local] = hir.TypeI32
		}
	}
	return localTypes, nil
}

func (e *emissionAnalysis) callResultType(symbol hir.SymbolID) (hir.Type, bool) {
	if index, found := e.bySymbol[symbol]; found {
		return e.functions[index].ResultType, true
	}
	if index, found := e.foreignBySymbol[symbol]; found {
		return e.foreignFunctions[index].ResultType, true
	}
	return hir.TypeInvalid, false
}

func assignLocalType(localTypes map[mir.LocalID]hir.Type, local mir.LocalID, value hir.Type) (bool, error) {
	if local < 0 {
		return false, fmt.Errorf("local has invalid id %d", local)
	}
	if value == hir.TypeVoid || value == hir.TypeInvalid || value == hir.ParseType("") {
		return false, fmt.Errorf("local %d has invalid value type %q", local, value)
	}
	if existing, found := localTypes[local]; found {
		if canonicalCValueType(existing) != canonicalCValueType(value) {
			return false, fmt.Errorf("local %d has conflicting types %q and %q", local, existing, value)
		}
		return false, nil
	}
	localTypes[local] = value
	return true, nil
}

func inferredOperandType(localTypes map[mir.LocalID]hir.Type, operand mir.Operand) (hir.Type, bool) {
	switch operand.Kind {
	case mir.OperandI64:
		return hir.TypeI64, true
	case mir.OperandI32:
		if operand.Type == hir.TypeBool {
			return hir.TypeBool, operand.I32 == 0 || operand.I32 == 1
		}
		return hir.TypeI32, true
	case mir.OperandU8:
		return hir.TypeU8, true
	case mir.OperandU32:
		return hir.TypeU32, true
	case mir.OperandUSize:
		return hir.TypeUSize, true
	case mir.OperandNull:
		if _, _, pointer := hir.RawPointerElement(operand.Type); pointer {
			return operand.Type, true
		}
		return hir.TypeInvalid, false
	case mir.OperandByteString:
		if operand.Type == hir.SharedBorrow(hir.TypeByteSlice) {
			return operand.Type, true
		}
		return hir.TypeInvalid, false
	case mir.OperandString:
		if operand.Type == hir.SharedBorrow(hir.TypeStr) && utf8.ValidString(operand.StaticData) {
			return operand.Type, true
		}
		return hir.TypeInvalid, false
	case mir.OperandCString:
		if operand.Type == hir.SharedBorrow(hir.TypeCStr) {
			return operand.Type, true
		}
		return hir.TypeInvalid, false
	case mir.OperandLocal:
		value, found := localTypes[operand.Local]
		return value, found
	default:
		return hir.TypeInvalid, false
	}
}

func (e *emissionAnalysis) operandType(functionIndex int, operand mir.Operand) (hir.Type, error) {
	switch operand.Kind {
	case mir.OperandRegion:
		return hir.TypeRegion, nil
	case mir.OperandI64:
		return hir.TypeI64, nil
	case mir.OperandI32:
		if operand.Type == hir.TypeBool {
			if operand.I32 == 0 || operand.I32 == 1 {
				return hir.TypeBool, nil
			}
			return hir.TypeInvalid, fmt.Errorf("boolean immediate has value %d", operand.I32)
		}
		return hir.TypeI32, nil
	case mir.OperandU8:
		return hir.TypeU8, nil
	case mir.OperandU32:
		return hir.TypeU32, nil
	case mir.OperandUSize:
		return hir.TypeUSize, nil
	case mir.OperandNull:
		if _, _, pointer := hir.RawPointerElement(operand.Type); pointer {
			return operand.Type, nil
		}
		return hir.TypeInvalid, fmt.Errorf("null operand has invalid type %q", operand.Type)
	case mir.OperandByteString:
		if operand.Type == hir.SharedBorrow(hir.TypeByteSlice) {
			return operand.Type, nil
		}
		return hir.TypeInvalid, fmt.Errorf("byte-string operand has invalid type %q", operand.Type)
	case mir.OperandString:
		if operand.Type == hir.SharedBorrow(hir.TypeStr) && utf8.ValidString(operand.StaticData) {
			return operand.Type, nil
		}
		return hir.TypeInvalid, fmt.Errorf("string operand has invalid type %q", operand.Type)
	case mir.OperandCString:
		if operand.Type == hir.SharedBorrow(hir.TypeCStr) {
			return operand.Type, nil
		}
		return hir.TypeInvalid, fmt.Errorf("C string operand has invalid type %q", operand.Type)
	case mir.OperandLocal:
		value, found := e.localTypes[functionIndex][operand.Local]
		if !found {
			return hir.TypeInvalid, fmt.Errorf("local operand %d has no inferred C type", operand.Local)
		}
		return value, nil
	default:
		return hir.TypeInvalid, fmt.Errorf("unsupported C operand kind %q", operand.Kind)
	}
}

func collectLocals(function mir.Function) []mir.LocalID {
	values := make(map[mir.LocalID]struct{})
	for _, parameter := range function.Parameters {
		values[parameter] = struct{}{}
	}
	addOperand := func(operand mir.Operand) {
		if operand.Kind == mir.OperandLocal {
			values[operand.Local] = struct{}{}
		}
	}
	for _, block := range function.Blocks {
		for _, operation := range block.Operations {
			if operationHasDestination(operation.Kind) && operation.DestinationLocal != mir.NoLocal {
				values[operation.DestinationLocal] = struct{}{}
			}
			for _, operand := range operation.Operands {
				addOperand(operand)
			}
		}
		terminator := block.Terminator
		if terminator.Kind == mir.TerminatorCall && terminator.DestinationLocal != mir.NoLocal {
			values[terminator.DestinationLocal] = struct{}{}
		}
		for _, operand := range terminator.Arguments {
			addOperand(operand)
		}
		for _, operand := range terminator.SwitchOperands {
			addOperand(operand)
		}
		addOperand(terminator.ReturnValue)
	}
	result := make([]mir.LocalID, 0, len(values))
	for value := range values {
		if value >= 0 {
			result = append(result, value)
		}
	}
	slices.Sort(result)
	return result
}

func functionRegionSymbols(function mir.Function) []hir.SymbolID {
	found := make(map[hir.SymbolID]struct{})
	for _, block := range function.Blocks {
		for _, operation := range block.Operations {
			if operation.Kind == mir.OperationRegionEnter {
				found[operation.Symbol] = struct{}{}
			}
		}
	}
	result := make([]hir.SymbolID, 0, len(found))
	for symbol := range found {
		if symbol != hir.NoSymbol {
			result = append(result, symbol)
		}
	}
	slices.Sort(result)
	return result
}

func functionCheckedDivisorTypes(function mir.Function) []hir.Type {
	found := make(map[hir.Type]bool)
	for _, block := range function.Blocks {
		for _, operation := range block.Operations {
			if operation.Kind == mir.OperationBinary && (operation.Operator == "/" || operation.Operator == "%") && !directUnsignedDivisor(operation) {
				found[operation.OwnerType] = true
			}
		}
	}
	result := make([]hir.Type, 0, len(found))
	for _, candidate := range []hir.Type{hir.TypeI32, hir.TypeI64, hir.TypeU8, hir.TypeU32, hir.TypeUSize} {
		if found[candidate] {
			result = append(result, candidate)
		}
	}
	return result
}

func operationHasDestination(kind mir.OperationKind) bool {
	switch kind {
	case mir.OperationInitialize, mir.OperationAssign, mir.OperationUnary, mir.OperationBinary,
		mir.OperationResultConstruct, mir.OperationResultPayload,
		mir.OperationOptionConstruct, mir.OperationOptionPayload,
		mir.OperationEnumConstruct, mir.OperationEnumPayload,
		mir.OperationRawAddress, mir.OperationRawBorrowPointer, mir.OperationStrFromRawParts, mir.OperationCStrFromRawParts,
		mir.OperationByteSliceFromRawParts, mir.OperationMutByteSliceFromRawParts,
		mir.OperationStrDataAddress, mir.OperationByteSliceDataAddress, mir.OperationCStrDataAddress,
		mir.OperationStructConstruct, mir.OperationResourceConstruct, mir.OperationFieldRead, mir.OperationFieldBorrow, mir.OperationViewLength, mir.OperationViewEmpty, mir.OperationViewIndex, mir.OperationViewElementBorrow, mir.OperationViewSlice,
		mir.OperationCCallback, mir.OperationStorageLayout, mir.OperationStorageTake, mir.OperationStorageBorrow, mir.OperationStorageBorrowMut, mir.OperationStorageElementAddress, mir.OperationScalarCast, mir.OperationSharedReborrow, mir.OperationArrayConstruct, mir.OperationArrayBorrowView, mir.OperationArrayLength, mir.OperationArrayIndex, mir.OperationRegionPlace:
		return true
	default:
		return false
	}
}
