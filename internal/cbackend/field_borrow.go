package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func (e *emitter) inspectFieldBorrow(functionIndex int, operation mir.Operation) error {
	element, mutable, borrowed := hir.BorrowElement(operation.Type)
	_, ownerMutable, ownerBorrowed := hir.BorrowElement(operation.OwnerType)
	if !borrowed || !ownerBorrowed || mutable != ownerMutable {
		return fmt.Errorf("field borrow requires matching owner and result reference kinds")
	}
	// Reuse definition and operand-type checks, not field-read value emission.
	operation.Kind, operation.Type = mir.OperationFieldRead, element
	if _, found := e.structByType[fieldAggregateType(operation.OwnerType)]; found {
		return e.inspectStructOperation(functionIndex, operation)
	}
	return e.inspectResourceOperation(functionIndex, operation)
}

func (e *emitter) writeFieldBorrow(output *strings.Builder, operation mir.Operation) error {
	owner, err := oneOperand(operation.Operands)
	if err != nil {
		return err
	}
	value, err := cOperand(owner)
	if err != nil {
		return err
	}
	aggregate := fieldAggregateType(operation.OwnerType)
	fieldName := ""
	if index, found := e.structByType[aggregate]; found {
		field, found := e.structTypes[index].fieldIndex[operation.Field]
		if !found {
			return fmt.Errorf("unknown borrowed struct field %q", operation.Field)
		}
		fieldName = e.structCFieldName(index, field)
	} else if index, found := e.resourceByType[aggregate]; found {
		field, found := e.resourceTypes[index].fieldIndex[operation.Field]
		if !found {
			return fmt.Errorf("unknown borrowed resource field %q", operation.Field)
		}
		fieldName = fmt.Sprintf("f_%d", field)
	} else {
		return fmt.Errorf("unknown field-borrow owner %q", aggregate)
	}
	fmt.Fprintf(output, "    fy_l_%d = &((%s)->%s);\n", operation.DestinationLocal, value, fieldName)
	return nil
}
