package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func (e *emitter) inspectStoragePut(functionIndex int, operation mir.Operation) error {
	pointer, value, err := twoOperands(operation.Operands)
	if err != nil {
		return err
	}
	pointerType, pointerErr := e.operandType(functionIndex, pointer)
	valueType, valueErr := e.operandType(functionIndex, value)
	if pointerErr != nil || valueErr != nil || pointerType != hir.RawPointer(hir.TypeVoid, true) || valueType != operation.OwnerType {
		return fmt.Errorf("private storage put has incompatible pointer/value types %q/%q", pointerType, valueType)
	}
	computed, err := preparedStorageLayout(operation.OwnerType, e.targetID, e.layoutEngine,
		e.structTypes, e.structByType, e.enumTypes, e.enumByType, e.resourceTypes, e.resourceByType)
	if err != nil {
		return err
	}
	if computed.SizeBytes == 0 {
		return fmt.Errorf("private storage put does not admit zero-sized values")
	}
	return nil
}

func (e *emitter) writeStoragePut(output *strings.Builder, operation mir.Operation) error {
	pointer, err := cOperand(operation.Operands[0])
	if err != nil {
		return err
	}
	value, err := cOperand(operation.Operands[1])
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "    memcpy(%s, &%s, sizeof %s);\n", pointer, value, value)
	return nil
}
