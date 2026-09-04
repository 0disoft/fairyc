package contractstatemir

import (
	"errors"
	"fmt"

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
	Cost    uint64
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
	helpers := make(map[string]Helper, len(program.Helpers))
	for _, helper := range program.Helpers {
		helpers[helper.Name] = helper
	}
	operationIndex := 0
	receipt, err := contractstate.Execute(store, bound.layout, limits, func(transaction *contractstate.Transaction) error {
		exit, err := executeProgramBody(transaction, bound, program, helpers, &outcome, &operationIndex)
		outcome.Exit = exit
		return err
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
