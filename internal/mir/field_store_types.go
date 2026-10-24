package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

// Operand annotations alone cannot grant writes to an unrelated local.
// Conflicts are errors: an absent type map would disable Region/type checks.
func fieldStoreLocalTypes(function *Function, functions map[hir.SymbolID]*Function, foreign map[hir.SymbolID]*ForeignFunction) (map[LocalID]hir.Type, error) {
	needed := false
	for _, typ := range function.ParameterTypes {
		needed = needed || typ == hir.TypeRegion
	}
	for _, block := range function.Blocks {
		for _, op := range block.Operations {
			needed = needed || op.Kind == OperationCCallback || op.Kind == OperationArrayConstruct || op.Kind == OperationArrayIndex || op.Kind == OperationArrayIndexStore || op.Kind == OperationArrayIndexCheck || op.Kind == OperationArrayLength || op.Kind == OperationFieldBorrow || op.Kind == OperationFieldStore || op.Kind == OperationDerefStore || op.Kind == OperationStorageElementAddress || op.Kind == OperationStorageBorrow || op.Kind == OperationStorageBorrowMut || op.Kind == OperationSharedReborrow || op.Kind == OperationViewSlice || op.Kind == OperationArrayBorrowView || op.Kind == OperationViewIndex || op.Kind == OperationViewElementBorrow || op.Kind == OperationViewIndexStore || op.Kind == OperationViewIndexCheck || op.Kind == OperationViewSliceCheck || op.Kind == OperationViewLength || op.Kind == OperationViewEmpty || op.Kind == OperationRegionPlace
		}
	}
	if !needed {
		return nil, nil
	}
	types := make(map[LocalID]hir.Type)
	var conflict error
	set := func(local LocalID, typ hir.Type) bool {
		if local < 0 || typ == hir.ParseType("") || typ == hir.TypeInvalid {
			return false
		}
		if previous, exists := types[local]; exists {
			if previous != typ {
				types[local] = hir.TypeInvalid
				if conflict == nil {
					conflict = fmt.Errorf("local %d has conflicting types %q and %q", local, previous, typ)
				}
			}
			return false
		}
		types[local] = typ
		return true
	}
	for index, local := range function.Parameters {
		if index < len(function.ParameterTypes) {
			set(local, function.ParameterTypes[index])
		}
	}
	for _, resource := range function.Resources {
		set(LocalID(resource.Symbol), resource.Type)
	}
	for changed := true; changed; {
		changed = false
		for _, block := range function.Blocks {
			for _, op := range block.Operations {
				typ := hir.TypeInvalid
				switch op.Kind {
				case OperationAssign, OperationInitialize:
					if len(op.Operands) == 1 {
						typ = fieldStoreOperandType(op.Operands[0], types)
					}
				case OperationRawAddress, OperationRawBorrowPointer, OperationCCallback, OperationUnary, OperationBinary, OperationScalarCast, OperationSharedReborrow,
					OperationResultConstruct, OperationResultPayload, OperationOptionConstruct, OperationOptionPayload,
					OperationEnumConstruct, OperationEnumPayload, OperationStructConstruct, OperationResourceConstruct,
					OperationFieldRead, OperationFieldBorrow, OperationArrayConstruct, OperationArrayBorrowView,
					OperationArrayLength, OperationArrayIndex, OperationViewLength, OperationViewEmpty,
					OperationViewIndex, OperationViewElementBorrow, OperationViewSlice, OperationStorageLayout, OperationStorageTake,
					OperationStorageBorrow, OperationStorageBorrowMut, OperationStorageElementAddress, OperationRegionPlace:
					typ = op.Type
				}
				changed = set(op.DestinationLocal, typ) || changed
			}
			term := block.Terminator
			if term.Kind == TerminatorCall {
				if callee := functions[term.Callee]; callee != nil {
					changed = set(term.DestinationLocal, callee.ResultType) || changed
				} else if callee := foreign[term.Callee]; callee != nil {
					changed = set(term.DestinationLocal, callee.ResultType) || changed
				}
			}
		}
	}
	if conflict != nil {
		return nil, conflict
	}
	return types, nil
}

func fieldStoreOperandType(operand Operand, locals map[LocalID]hir.Type) hir.Type {
	switch operand.Kind {
	case OperandLocal:
		return locals[operand.Local]
	case OperandI32:
		if operand.Type == hir.TypeBool {
			if operand.I32 == 0 || operand.I32 == 1 {
				return hir.TypeBool
			}
			return hir.TypeInvalid
		}
		return hir.TypeI32
	case OperandByteString:
		if operand.Type == hir.SharedBorrow(hir.TypeByteSlice) {
			return operand.Type
		}
		return hir.TypeInvalid
	case OperandI64:
		return hir.TypeI64
	case OperandU8:
		return hir.TypeU8
	case OperandU32:
		return hir.TypeU32
	case OperandUSize:
		return hir.TypeUSize
	default:
		return hir.TypeInvalid
	}
}
