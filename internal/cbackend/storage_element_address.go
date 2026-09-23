package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func (e *emitter) inspectStorageElementAddress(functionIndex int, operation mir.Operation) error {
	base, index, err := twoOperands(operation.Operands)
	if err != nil {
		return err
	}
	baseType, baseErr := e.operandType(functionIndex, base)
	indexType, indexErr := e.operandType(functionIndex, index)
	if baseErr != nil || indexErr != nil || baseType != hir.RawPointer(hir.TypeVoid, true) ||
		indexType != hir.TypeUSize || operation.Type != hir.RawPointer(hir.TypeVoid, true) {
		return fmt.Errorf("private storage element address requires mutable void pointer and usize, got %q/%q: %v %v", baseType, indexType, baseErr, indexErr)
	}
	layout, err := preparedStorageLayout(operation.OwnerType, e.targetID, e.layoutEngine,
		e.structTypes, e.structByType, e.enumTypes, e.enumByType, e.resourceTypes, e.resourceByType)
	if err != nil {
		return err
	}
	if layout.SizeBytes == 0 {
		return fmt.Errorf("private storage element address does not admit zero-sized elements")
	}
	return nil
}

func (e *emitter) writeStorageElementAddress(output *strings.Builder, operation mir.Operation) error {
	base, err := cOperand(operation.Operands[0])
	if err != nil {
		return err
	}
	index, err := cOperand(operation.Operands[1])
	if err != nil {
		return err
	}
	layout, err := preparedStorageLayout(operation.OwnerType, e.targetID, e.layoutEngine,
		e.structTypes, e.structByType, e.enumTypes, e.enumByType, e.resourceTypes, e.resourceByType)
	if err != nil || layout.SizeBytes == 0 {
		return fmt.Errorf("private storage element address layout: %v", err)
	}
	size := fmt.Sprintf("((uintptr_t)UINT64_C(%d))", layout.SizeBytes)
	fmt.Fprintf(output, "    if (%s == NULL || %s > UINTPTR_MAX / %s) { fy_l_%d = NULL; } else { fy_l_%d = (void *)((unsigned char *)%s + %s * %s); }\n",
		base, index, size, operation.DestinationLocal, operation.DestinationLocal, base, index, size)
	return nil
}
