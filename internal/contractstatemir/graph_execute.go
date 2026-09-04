package contractstatemir

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/contractstate"
)

func executeGraph(transaction *contractstate.Transaction, bound boundLayout, program Program, outcome *Outcome) (ExitKind, error) {
	values := make(map[ValueID]Value)
	current := program.Entry
	operationIndex := 0
	for {
		block := program.Blocks[current]
		for _, operation := range block.Operations {
			field := bound.fields[operation.FieldIndex]
			key := operation.Key
			if operation.KeyInput != 0 {
				key = values[operation.KeyInput]
			}
			value := operation.Value
			if operation.ValueInput != 0 {
				value = values[operation.ValueInput]
			}
			switch operation.Kind {
			case OperationCellRead:
				encoded, found, err := field.cell.Get(transaction)
				if err != nil {
					return "", err
				}
				outcome.Reads = append(outcome.Reads, Read{Operation: operationIndex, Found: found, Value: Value{TypeID: field.descriptor.ValueTypeID, Encoded: slices.Clone(encoded)}})
				values[operation.Result] = graphReadValue(field.descriptor.ValueTypeID, encoded, found)
				values[operation.Found] = boolGraphValue(found)
			case OperationCellWrite:
				if err := field.cell.Set(transaction, value.Encoded); err != nil {
					return "", err
				}
			case OperationCellDelete:
				if err := field.cell.Delete(transaction); err != nil {
					return "", err
				}
			case OperationMapRead:
				encoded, found, err := field.mapped.Get(transaction, key.Encoded)
				if err != nil {
					return "", err
				}
				outcome.Reads = append(outcome.Reads, Read{Operation: operationIndex, Found: found, Value: Value{TypeID: field.descriptor.ValueTypeID, Encoded: slices.Clone(encoded)}})
				values[operation.Result] = graphReadValue(field.descriptor.ValueTypeID, encoded, found)
				values[operation.Found] = boolGraphValue(found)
			case OperationMapWrite:
				if err := field.mapped.Set(transaction, key.Encoded, value.Encoded); err != nil {
					return "", err
				}
			case OperationMapDelete:
				if err := field.mapped.Delete(transaction, key.Encoded); err != nil {
					return "", err
				}
			default:
				return "", fmt.Errorf("verified contract state graph contains operation %q", operation.Kind)
			}
			operationIndex++
			outcome.Cost += uint64(operation.Cost)
		}
		term := block.Terminator
		outcome.Cost += uint64(term.Cost)
		switch term.Kind {
		case TerminatorJump:
			assignGraphArguments(values, program.Blocks[term.Target].Parameters, term.Arguments)
			current = term.Target
		case TerminatorBranch:
			condition, err := contractstate.Bool.Decode(values[term.Condition].Encoded)
			if err != nil {
				return "", fmt.Errorf("decode verified branch condition: %w", err)
			}
			if condition {
				assignGraphArguments(values, program.Blocks[term.TrueTarget].Parameters, term.TrueArguments)
				current = term.TrueTarget
			} else {
				assignGraphArguments(values, program.Blocks[term.FalseTarget].Parameters, term.FalseArguments)
				current = term.FalseTarget
			}
		case TerminatorCommit:
			return ExitCommit, nil
		case TerminatorReject:
			return ExitReject, errRejected
		case TerminatorTrap:
			panic(Trap{Category: term.TrapCategory})
		default:
			return "", fmt.Errorf("verified contract state graph contains terminator %q", term.Kind)
		}
	}
}

func assignGraphArguments(values map[ValueID]Value, parameters []BlockParameter, arguments []ValueID) {
	resolved := make([]Value, len(arguments))
	for index, argument := range arguments {
		resolved[index] = values[argument]
	}
	for index, parameter := range parameters {
		resolved[index].Encoded = slices.Clone(resolved[index].Encoded)
		values[parameter.ID] = resolved[index]
	}
}

func boolGraphValue(value bool) Value {
	encoded, _ := contractstate.Bool.Encode(value)
	return Value{TypeID: contractstate.Bool.TypeID(), Encoded: encoded}
}

func graphReadValue(typeID string, encoded []byte, found bool) Value {
	if found {
		return Value{TypeID: typeID, Encoded: slices.Clone(encoded)}
	}
	return Value{TypeID: typeID, Encoded: zeroGraphEncoding(typeID)}
}

func zeroGraphEncoding(typeID string) []byte {
	switch typeID {
	case contractstate.Bool.TypeID():
		return []byte{0}
	case contractstate.U32.TypeID():
		return make([]byte, 4)
	case contractstate.U64.TypeID():
		return make([]byte, 8)
	case contractstate.Byte20.TypeID():
		return make([]byte, 20)
	case contractstate.Byte32.TypeID():
		return make([]byte, 32)
	case contractstate.Bytes.TypeID(), contractstate.Text.TypeID():
		return []byte{}
	default:
		panic("verified graph has unknown value type")
	}
}
