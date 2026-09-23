package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func (e *emitter) inspectStorageBorrow(functionIndex int, operation mir.Operation) error {
	owner, address, err := twoOperands(operation.Operands)
	if err != nil {
		return err
	}
	ownerType, ownerErr := e.operandType(functionIndex, owner)
	addressType, addressErr := e.operandType(functionIndex, address)
	if ownerErr != nil || addressErr != nil || ownerType != operation.OwnerType || addressType != operation.IndexType {
		return fmt.Errorf("private storage borrow has incompatible owner/address types %q/%q", ownerType, addressType)
	}
	element, mutable, borrowed := hir.BorrowElement(operation.Type)
	wantsMutable := operation.Kind == mir.OperationStorageBorrowMut
	_, ownerMutable, ownerBorrowed := hir.BorrowElement(operation.OwnerType)
	pointerElement, pointerMutable, pointer := hir.RawPointerElement(operation.IndexType)
	if !borrowed || mutable != wantsMutable || !ownerBorrowed || ownerMutable != wantsMutable ||
		!pointer || pointerElement != hir.TypeVoid || (wantsMutable && !pointerMutable) {
		return fmt.Errorf("private storage borrow has incompatible mutability")
	}
	computed, err := preparedStorageLayout(element, e.targetID, e.layoutEngine,
		e.structTypes, e.structByType, e.enumTypes, e.enumByType, e.resourceTypes, e.resourceByType)
	if err != nil {
		return err
	}
	if computed.SizeBytes == 0 {
		return fmt.Errorf("private storage borrow does not admit zero-sized elements")
	}
	return nil
}

func (e *emitter) writeStorageBorrow(output *strings.Builder, operation mir.Operation) error {
	address, err := cOperand(operation.Operands[1])
	if err != nil {
		return err
	}
	// The safe reference keeps the pointee in place; no representation is copied
	// and no cleanup obligation is created for the borrowed element.
	fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, address)
	return nil
}
