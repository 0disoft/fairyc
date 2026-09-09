package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func (e *emitter) inspectFieldStore(functionIndex int, operation mir.Operation) error {
	if !mir.ValidFieldStoreShape(operation) {
		return fmt.Errorf("invalid scalar field store shape")
	}
	valueType, err := e.operandType(functionIndex, operation.Operands[1])
	if err != nil || valueType != operation.Type {
		return fmt.Errorf("field store value type %q does not match %q: %v", valueType, operation.Type, err)
	}
	// The read inspector owns the exact declared-field and receiver-type checks.
	operation.Kind, operation.Operands = mir.OperationFieldRead, operation.Operands[:1]
	if _, found := e.structByType[fieldAggregateType(operation.OwnerType)]; found {
		return e.inspectStructOperation(functionIndex, operation)
	}
	return e.inspectResourceOperation(functionIndex, operation)
}

func (e *emitter) writeFieldStore(output *strings.Builder, operation mir.Operation) error {
	receiver, err := cOperand(operation.Operands[0])
	if err != nil {
		return err
	}
	value, err := cOperand(operation.Operands[1])
	if err != nil {
		return err
	}
	aggregate := fieldAggregateType(operation.OwnerType)
	name := ""
	if index, found := e.structByType[aggregate]; found {
		field, found := e.structTypes[index].fieldIndex[operation.Field]
		if !found {
			return fmt.Errorf("missing stored struct field")
		}
		name = e.structCFieldName(index, field)
	} else if index, found := e.resourceByType[aggregate]; found {
		field, found := e.resourceTypes[index].fieldIndex[operation.Field]
		if !found {
			return fmt.Errorf("missing stored resource field")
		}
		name = fmt.Sprintf("f_%d", field)
	} else {
		return fmt.Errorf("missing field store owner")
	}
	access := "."
	if _, _, borrowed := hir.BorrowElement(operation.OwnerType); borrowed {
		access = "->"
	}
	fmt.Fprintf(output, "    (%s)%s%s = %s;\n", receiver, access, name, value)
	return nil
}
