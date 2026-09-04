package contractstatemir

import (
	"bytes"
	"fmt"

	"github.com/0disoft/fairyc/internal/contractstate"
)

type graphBorrowOrigin struct {
	field uint32
	key   []byte
}

type graphBorrow struct {
	typeID string
	origin graphBorrowOrigin
	valid  bool
}

func isBorrowOperation(kind OperationKind) bool {
	return kind == OperationCellBorrow || kind == OperationMapBorrow || kind == OperationBorrowView || kind == OperationBorrowCopy
}

func graphBorrowResultType(operation Operation, bound boundLayout, local map[BorrowID]string) string {
	switch operation.Kind {
	case OperationCellBorrow, OperationMapBorrow:
		if int(operation.FieldIndex) < len(bound.fields) {
			return bound.fields[operation.FieldIndex].descriptor.ValueTypeID
		}
	case OperationBorrowView:
		return local[operation.BorrowInput]
	}
	return ""
}

func defineGraphBorrow(types map[BorrowID]string, id BorrowID, typeID string) string {
	if id == 0 || !knownGraphType(typeID) {
		return "borrow identity and compiler-known value type must be nonzero"
	}
	if _, found := types[id]; found {
		return fmt.Sprintf("borrow %d is defined more than once", id)
	}
	types[id] = typeID
	return ""
}

func verifyGraphBorrowOperation(operation Operation, field contractstate.Field, values map[ValueID]string, borrows map[BorrowID]graphBorrow) string {
	if operation.Callee != "" || operation.ValueInput != 0 || operation.Value.TypeID != "" || operation.Value.Encoded != nil {
		return "borrow operation carries unrelated call or write fields"
	}
	switch operation.Kind {
	case OperationCellBorrow:
		if field.Kind != contractstate.FieldCell || operation.KeyInput != 0 || operation.Key.TypeID != "" || operation.Key.Encoded != nil || operation.Result != 0 || operation.Found == 0 || operation.BorrowInput != 0 || operation.BorrowResult == 0 || operation.ViewOffset != 0 || operation.ViewLength != 0 {
			return "cell borrow requires one borrow result and found result with no key or view fields"
		}
	case OperationMapBorrow:
		if field.Kind != contractstate.FieldMap || operation.KeyInput != 0 || operation.Result != 0 || operation.Found == 0 || operation.BorrowInput != 0 || operation.BorrowResult == 0 || operation.ViewOffset != 0 || operation.ViewLength != 0 {
			return "map borrow requires one literal exact key, borrow result, and found result"
		}
		if detail := verifyValue("key", operation.Key, field.KeyTypeID); detail != "" {
			return detail
		}
	case OperationBorrowView:
		borrow, found := borrows[operation.BorrowInput]
		if !found || !borrow.valid {
			return "nested view requires one available non-invalidated borrow"
		}
		if operation.FieldIndex != 0 || operation.KeyInput != 0 || operation.Key.TypeID != "" || operation.Key.Encoded != nil || operation.Result != 0 || operation.Found != 0 || operation.BorrowResult == 0 || operation.ViewLength == 0 {
			return "nested view carries storage fields or lacks a nonempty range and result"
		}
	case OperationBorrowCopy:
		borrow, found := borrows[operation.BorrowInput]
		if !found || !borrow.valid {
			return "borrow copy requires one available non-invalidated borrow"
		}
		if operation.FieldIndex != 0 || operation.KeyInput != 0 || operation.Key.TypeID != "" || operation.Key.Encoded != nil || operation.Result == 0 || operation.Found != 0 || operation.BorrowResult != 0 || operation.ViewOffset != 0 || operation.ViewLength != 0 {
			return "borrow copy carries storage/view fields or lacks a value result"
		}
		if existing := values[operation.Result]; existing != "" && existing != borrow.typeID {
			return "borrow copy result type is inconsistent"
		}
	default:
		return fmt.Sprintf("borrow operation kind %q is unsupported", operation.Kind)
	}
	return ""
}

func applyGraphBorrowOperation(operation Operation, field contractstate.Field, borrows map[BorrowID]graphBorrow) {
	switch operation.Kind {
	case OperationCellBorrow:
		borrows[operation.BorrowResult] = graphBorrow{typeID: field.ValueTypeID, origin: graphBorrowOrigin{field: operation.FieldIndex}, valid: true}
	case OperationMapBorrow:
		borrows[operation.BorrowResult] = graphBorrow{typeID: field.ValueTypeID, origin: graphBorrowOrigin{field: operation.FieldIndex, key: bytes.Clone(operation.Key.Encoded)}, valid: true}
	case OperationBorrowView:
		borrows[operation.BorrowResult] = borrows[operation.BorrowInput]
	}
}

func invalidateGraphBorrows(operation Operation, field contractstate.Field, values map[ValueID]string, borrows map[BorrowID]graphBorrow) string {
	mutates := operation.Kind == OperationCellWrite || operation.Kind == OperationCellDelete || operation.Kind == OperationMapWrite || operation.Kind == OperationMapDelete
	if !mutates {
		return ""
	}
	if field.Kind == contractstate.FieldMap && operation.KeyInput != 0 {
		for _, borrow := range borrows {
			if borrow.valid && borrow.origin.field == operation.FieldIndex {
				if values[operation.KeyInput] == "" {
					return "dynamic map mutation key is unavailable"
				}
				return "dynamic map mutation has ambiguous alias with a live state borrow"
			}
		}
		return ""
	}
	for id, borrow := range borrows {
		if !borrow.valid || borrow.origin.field != operation.FieldIndex {
			continue
		}
		if field.Kind == contractstate.FieldMap && !bytes.Equal(borrow.origin.key, operation.Key.Encoded) {
			continue
		}
		borrow.valid = false
		borrows[id] = borrow
	}
	return ""
}

func verifyGraphBorrowEscape(term Terminator, values map[ValueID]string, borrows map[BorrowID]graphBorrow) string {
	arguments := append([]ValueID(nil), term.Arguments...)
	arguments = append(arguments, term.TrueArguments...)
	arguments = append(arguments, term.FalseArguments...)
	for _, argument := range arguments {
		if values[argument] != "" {
			continue
		}
		if _, found := borrows[BorrowID(argument)]; found {
			return fmt.Sprintf("owner-bound borrow %d cannot cross a block edge", argument)
		}
	}
	return ""
}
