package contractstatemir

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/contractstate"
)

func executeProgramBody(transaction *contractstate.Transaction, bound boundLayout, program Program, helpers map[string]Helper, outcome *Outcome, operationIndex *int) (ExitKind, error) {
	if len(program.Blocks) != 0 {
		return executeGraph(transaction, bound, program, helpers, outcome, operationIndex)
	}
	for _, operation := range program.Operations {
		if operation.Kind == OperationCall {
			(*operationIndex)++
			outcome.Cost += uint64(operation.Cost)
			exit, err := executeHelper(transaction, bound, program.Capability, helpers[operation.Callee], helpers, outcome, operationIndex)
			if err != nil || exit != ExitReturn {
				return exit, err
			}
			continue
		}
		field := bound.fields[operation.FieldIndex]
		switch operation.Kind {
		case OperationCellRead:
			encoded, found, err := field.cell.Get(transaction)
			if err != nil {
				return "", err
			}
			outcome.Reads = append(outcome.Reads, Read{Operation: *operationIndex, Found: found, Value: Value{TypeID: field.descriptor.ValueTypeID, Encoded: slices.Clone(encoded)}})
		case OperationCellWrite:
			if err := field.cell.Set(transaction, operation.Value.Encoded); err != nil {
				return "", err
			}
		case OperationCellDelete:
			if err := field.cell.Delete(transaction); err != nil {
				return "", err
			}
		case OperationMapRead:
			encoded, found, err := field.mapped.Get(transaction, operation.Key.Encoded)
			if err != nil {
				return "", err
			}
			outcome.Reads = append(outcome.Reads, Read{Operation: *operationIndex, Found: found, Value: Value{TypeID: field.descriptor.ValueTypeID, Encoded: slices.Clone(encoded)}})
		case OperationMapWrite:
			if err := field.mapped.Set(transaction, operation.Key.Encoded, operation.Value.Encoded); err != nil {
				return "", err
			}
		case OperationMapDelete:
			if err := field.mapped.Delete(transaction, operation.Key.Encoded); err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("verified contract state MIR contains operation %q", operation.Kind)
		}
		(*operationIndex)++
	}
	switch program.Exit {
	case ExitCommit, ExitReturn:
		return program.Exit, nil
	case ExitReject:
		return ExitReject, errRejected
	case ExitTrap:
		panic(Trap{Category: program.TrapCategory})
	default:
		return "", fmt.Errorf("verified contract state MIR contains exit %q", program.Exit)
	}
}

func executeHelper(transaction *contractstate.Transaction, bound boundLayout, capability TransactionCapability, helper Helper, helpers map[string]Helper, outcome *Outcome, operationIndex *int) (ExitKind, error) {
	program := Program{
		Capability: capability, Effects: helper.Effects,
		Operations: helper.Operations, Exit: helper.Exit, TrapCategory: helper.TrapCategory,
		Entry: helper.Entry, Blocks: helper.Blocks,
	}
	return executeProgramBody(transaction, bound, program, helpers, outcome, operationIndex)
}
