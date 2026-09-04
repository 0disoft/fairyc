package contractstatemir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/contractstate"
)

func executeGraphBorrow(transaction *contractstate.Transaction, bound boundLayout, operation Operation, values map[ValueID]Value, borrows map[BorrowID]contractstate.BorrowedValue, outcome *Outcome, operationIndex int) error {
	switch operation.Kind {
	case OperationCellBorrow, OperationMapBorrow:
		field := bound.fields[operation.FieldIndex]
		var borrow contractstate.BorrowedValue
		var found bool
		var err error
		if operation.Kind == OperationCellBorrow {
			borrow, found, err = field.cell.Borrow(transaction)
		} else {
			borrow, found, err = field.mapped.Borrow(transaction, operation.Key.Encoded)
		}
		if err != nil {
			return err
		}
		encoded := zeroGraphEncoding(field.descriptor.ValueTypeID)
		if found {
			encoded, err = borrow.Bytes()
			if err != nil {
				return err
			}
			borrows[operation.BorrowResult] = borrow
		}
		outcome.Reads = append(outcome.Reads, Read{Operation: operationIndex, Found: found, Value: Value{TypeID: field.descriptor.ValueTypeID, Encoded: encoded}})
		values[operation.Found] = boolGraphValue(found)
		return nil
	case OperationBorrowView:
		borrow, err := borrows[operation.BorrowInput].View(int(operation.ViewOffset), int(operation.ViewLength))
		if err != nil {
			return fmt.Errorf("derive verified state borrow view: %w", err)
		}
		borrows[operation.BorrowResult] = borrow
		return nil
	case OperationBorrowCopy:
		encoded, err := borrows[operation.BorrowInput].Bytes()
		if err != nil {
			return fmt.Errorf("observe verified state borrow: %w", err)
		}
		values[operation.Result] = Value{TypeID: borrows[operation.BorrowInput].Origin().TypeID, Encoded: encoded}
		return nil
	default:
		return fmt.Errorf("verified contract state graph contains borrow operation %q", operation.Kind)
	}
}
