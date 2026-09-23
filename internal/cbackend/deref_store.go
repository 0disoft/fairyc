package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/mir"
)

func (e *emitter) inspectDerefStore(functionIndex int, operation mir.Operation) error {
	if !mir.ValidDerefStoreShape(operation) {
		return fmt.Errorf("invalid scalar dereference store shape")
	}
	receiverType, err := e.operandType(functionIndex, operation.Operands[0])
	if err != nil || receiverType != operation.OwnerType {
		return fmt.Errorf("dereference store receiver type %q does not match %q: %v", receiverType, operation.OwnerType, err)
	}
	valueType, err := e.operandType(functionIndex, operation.Operands[1])
	if err != nil || valueType != operation.Type {
		return fmt.Errorf("dereference store value type %q does not match %q: %v", valueType, operation.Type, err)
	}
	return nil
}

func (e *emitter) writeDerefStore(output *strings.Builder, operation mir.Operation) error {
	receiver, err := cOperand(operation.Operands[0])
	if err != nil {
		return err
	}
	value, err := cOperand(operation.Operands[1])
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "    *(%s) = %s;\n", receiver, value)
	return nil
}
