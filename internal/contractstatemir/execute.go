package contractstatemir

import (
	"errors"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/contractstate"
)

var errRejected = errors.New("contract state MIR rejected execution")

// Trap is the exact panic payload used by an explicit transaction trap exit.
// contractstate.Execute rolls back the overlay before this value is re-panicked.
type Trap struct {
	Category string
}

func (trap Trap) Error() string {
	return "contract state MIR trap: " + trap.Category
}

// Read records one deterministic storage observation in operation order.
type Read struct {
	Operation int
	Found     bool
	Value     Value
}

// Outcome records the explicit non-trap exit plus commit and read evidence.
type Outcome struct {
	Exit    ExitKind
	Receipt contractstate.CommitReceipt
	Reads   []Read
}

// NewMemoryStore constructs the semantic oracle only after verifying the
// compiler-private layout identity.
func NewMemoryStore(layout Layout) (*contractstate.MemoryStore, error) {
	bound, err := bindLayout(layout)
	if err != nil {
		return nil, fmt.Errorf("verify contract state MIR layout: %w", err)
	}
	return contractstate.NewMemoryStore(bound.layout)
}

// Execute verifies the complete program before beginning a transaction, then
// delegates publication and rollback to the contractstate semantic oracle.
func Execute(store *contractstate.MemoryStore, program Program, limits contractstate.Limits) (Outcome, error) {
	if store == nil {
		return Outcome{}, fmt.Errorf("contract state MIR store is nil")
	}
	failures := Verify(program)
	if len(failures) != 0 {
		return Outcome{}, VerificationErrors{Failures: failures}
	}
	bound, err := bindLayout(program.Layout)
	if err != nil {
		return Outcome{}, fmt.Errorf("bind verified contract state MIR layout: %w", err)
	}
	outcome := Outcome{Exit: program.Exit}
	receipt, err := contractstate.Execute(store, bound.layout, limits, func(transaction *contractstate.Transaction) error {
		for index, operation := range program.Operations {
			field := bound.fields[operation.FieldIndex]
			switch operation.Kind {
			case OperationCellRead:
				encoded, found, err := field.cell.Get(transaction)
				if err != nil {
					return err
				}
				outcome.Reads = append(outcome.Reads, Read{Operation: index, Found: found, Value: Value{TypeID: field.descriptor.ValueTypeID, Encoded: slices.Clone(encoded)}})
			case OperationCellWrite:
				if err := field.cell.Set(transaction, operation.Value.Encoded); err != nil {
					return err
				}
			case OperationCellDelete:
				if err := field.cell.Delete(transaction); err != nil {
					return err
				}
			case OperationMapRead:
				encoded, found, err := field.mapped.Get(transaction, operation.Key.Encoded)
				if err != nil {
					return err
				}
				outcome.Reads = append(outcome.Reads, Read{Operation: index, Found: found, Value: Value{TypeID: field.descriptor.ValueTypeID, Encoded: slices.Clone(encoded)}})
			case OperationMapWrite:
				if err := field.mapped.Set(transaction, operation.Key.Encoded, operation.Value.Encoded); err != nil {
					return err
				}
			case OperationMapDelete:
				if err := field.mapped.Delete(transaction, operation.Key.Encoded); err != nil {
					return err
				}
			default:
				return fmt.Errorf("verified contract state MIR contains operation %q", operation.Kind)
			}
		}
		switch program.Exit {
		case ExitCommit:
			return nil
		case ExitReject:
			return errRejected
		case ExitTrap:
			panic(Trap{Category: program.TrapCategory})
		default:
			return fmt.Errorf("verified contract state MIR contains exit %q", program.Exit)
		}
	})
	if errors.Is(err, errRejected) {
		return outcome, nil
	}
	if err != nil {
		return Outcome{}, err
	}
	outcome.Receipt = receipt
	return outcome, nil
}

// VerificationErrors keeps every deterministic verifier failure together.
type VerificationErrors struct {
	Failures []VerificationError
}

func (failures VerificationErrors) Error() string {
	if len(failures.Failures) == 0 {
		return "contract state MIR verification failed"
	}
	return fmt.Sprintf("contract state MIR verification failed with %d error(s): %s", len(failures.Failures), failures.Failures[0])
}
