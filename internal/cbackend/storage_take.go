package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// EmitStorageTransfers explicitly enables private unsafe typed storage transfers.
// It is not used by stable source, CLI or artifact consumers.
func EmitStorageTransfers(program mir.Program, entry string, profile Profile) (Unit, error) {
	return emit(program, entry, profile, true)
}

func (e *analysisBuilder) inspectStorageTake(functionIndex int, operation mir.Operation) error {
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return err
	}
	actual, err := e.operandType(functionIndex, operand)
	if err != nil || actual != hir.RawPointer(hir.TypeVoid, true) || actual != operation.OwnerType {
		return fmt.Errorf("private storage take requires an exact mutable void pointer, got %q: %v", actual, err)
	}
	computed, err := preparedStorageLayout(operation.Type, e.targetID, e.layoutEngine,
		e.structTypes, e.structByType, e.enumTypes, e.enumByType, e.resourceTypes, e.resourceByType)
	if err != nil {
		return err
	}
	if computed.SizeBytes == 0 {
		return fmt.Errorf("private storage take does not admit zero-sized values")
	}
	return nil
}

func (e *emitter) writeStorageTake(output *strings.Builder, operation mir.Operation) error {
	pointer, err := cOperand(operation.Operands[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "    memcpy(&fy_l_%d, %s, sizeof fy_l_%d);\n", operation.DestinationLocal, pointer, operation.DestinationLocal)
	return nil
}
