package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func (e *emitter) inspectStorageBorrow(functionIndex int, operation mir.Operation) error {
	if hir.I64View(operation.Type) {
		return e.inspectI64StorageView(functionIndex, operation)
	}
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
	if hir.I64View(operation.Type) {
		length, err := cOperand(operation.Operands[2])
		if err != nil {
			return err
		}
		descriptor, pointer := "fy_i64_slice", "const int64_t *"
		if hir.MutableIndexedView(operation.Type) {
			descriptor, pointer = "fy_mut_i64_slice", "int64_t *"
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s){ .data = (%s)(%s), .length = %s };\n", operation.DestinationLocal, descriptor, pointer, address, length)
		return nil
	}
	// The safe reference keeps the pointee in place; no representation is copied
	// and no cleanup obligation is created for the borrowed element.
	fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, address)
	return nil
}

func (e *emitter) inspectI64StorageView(functionIndex int, operation mir.Operation) error {
	wantsMutable := operation.Kind == mir.OperationStorageBorrowMut
	if (operation.Kind != mir.OperationStorageBorrow && !wantsMutable) || len(operation.Operands) != 3 || !hir.I64View(operation.Type) || hir.MutableIndexedView(operation.Type) != wantsMutable {
		return fmt.Errorf("private i64 storage view has invalid shape")
	}
	_, mutable, borrowed := hir.BorrowElement(operation.OwnerType)
	element, pointerMutable, pointer := hir.RawPointerElement(operation.IndexType)
	if !borrowed || mutable != wantsMutable || !pointer || element != hir.TypeVoid || (wantsMutable && !pointerMutable) {
		return fmt.Errorf("private i64 storage view has incompatible owner or address type")
	}
	for index, want := range []hir.Type{operation.OwnerType, operation.IndexType, hir.TypeUSize} {
		actual, err := e.operandType(functionIndex, operation.Operands[index])
		if err != nil || actual != want {
			return fmt.Errorf("private i64 storage view argument %d has type %q, want %q", index, actual, want)
		}
	}
	if wantsMutable {
		e.needsMutI64Slice = true
	} else {
		e.needsI64Slice = true
	}
	return nil
}
