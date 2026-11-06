package contractstatenative

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
)

// GraphRuntimeError preserves a private native runtime status. Transaction-policy
// and borrow failures retain their existing semantic causes; a transport arena
// failure is not reclassified as a caller's value or write policy violation.
type GraphRuntimeError struct{ Status byte }

func (err GraphRuntimeError) Error() string {
	return fmt.Sprintf("native graph runtime rejected execution (status %d)", err.Status)
}

func (err GraphRuntimeError) Unwrap() error {
	switch err.Status {
	case 2:
		return contractstate.ErrClosed
	case 3:
		return contractstate.ErrValueTooLarge
	case 4:
		return contractstate.ErrWriteLimit
	case 5:
		return contractstate.ErrWriteBytesLimit
	case 7:
		return contractstate.ErrMapKeyTooLarge
	case 9:
		return contractstate.ErrBorrowInvalidated
	default:
		return nil
	}
}

// ExecuteGraphPersistent takes one exact snapshot, executes one sealed native
// graph and submits at most one complete overlay to the existing layout/version
// CAS. A conflict never replays the graph; acknowledged publication stays success
// even if cancellation arrives during the synchronous storage acknowledgement.
func ExecuteGraphPersistent(ctx context.Context, store contractstate.PersistencePort, executable GraphExecutable, limits contractstate.Limits) (contractstatemir.Outcome, error) {
	if !executable.valid || len(executable.image.Bytes) == 0 {
		return contractstatemir.Outcome{}, fmt.Errorf("native graph persistence requires a built owner")
	}
	return executeGraphPersistent(ctx, store, executable.prepared, limits, executable.run)
}

type graphExecution func(context.Context, GraphInput) (cdriver.WindowsHostOutcome, error)

func executeGraphPersistent(ctx context.Context, store contractstate.PersistencePort, prepared GraphTransport, limits contractstate.Limits, run graphExecution) (contractstatemir.Outcome, error) {
	if ctx == nil || store == nil || run == nil {
		return contractstatemir.Outcome{}, fmt.Errorf("native graph persistence requires context, storage and execution")
	}
	layout := prepared.entry.LayoutHash()
	// Invalid owners/policy cannot observe persistent state or start a process.
	if _, err := prepared.Input(contractstate.PersistentSnapshot{LayoutHash: layout}, limits); err != nil {
		return contractstatemir.Outcome{}, err
	}
	if err := ctx.Err(); err != nil {
		return contractstatemir.Outcome{}, err
	}
	snapshot, err := store.Snapshot(layout)
	if err != nil {
		return contractstatemir.Outcome{}, fmt.Errorf("snapshot native graph: %w", errors.Join(err, ctx.Err()))
	}
	if err := ctx.Err(); err != nil {
		return contractstatemir.Outcome{}, err
	}
	input, err := prepared.Input(snapshot, limits)
	if err != nil {
		return contractstatemir.Outcome{}, err
	}
	result, err := run(ctx, input)
	if err != nil {
		return contractstatemir.Outcome{}, fmt.Errorf("execute native graph: %w", errors.Join(err, ctx.Err()))
	}
	if err := ctx.Err(); err != nil {
		return contractstatemir.Outcome{}, err
	}
	decoded, err := prepared.decodeProcess(input, result)
	if err != nil {
		return contractstatemir.Outcome{}, err
	}
	if err := ctx.Err(); err != nil {
		return contractstatemir.Outcome{}, err
	}
	if decoded.Status != 0 {
		return contractstatemir.Outcome{}, GraphRuntimeError{Status: decoded.Status}
	}
	if decoded.Exit == contractstatemir.ExitReject {
		return contractstatemir.Outcome{Exit: decoded.Exit, Reads: decoded.Reads, Cost: decoded.Cost}, nil
	}
	if decoded.Exit == contractstatemir.ExitTrap {
		panic(contractstatemir.Trap{Category: decoded.TrapCategory})
	}
	expectedVersion := snapshot.Version
	if len(decoded.Writes) != 0 {
		if expectedVersion == math.MaxUint64 {
			return contractstatemir.Outcome{}, fmt.Errorf("native graph state version exhausted")
		}
		expectedVersion++
	}
	if err := ctx.Err(); err != nil {
		return contractstatemir.Outcome{}, err
	}
	// Empty commits still compare the observed version/layout. They do not
	// advance it, and cannot silently succeed over a competing publication.
	receipt, err := store.CompareAndPublish(layout, snapshot.Version, decoded.Writes)
	if err != nil {
		return contractstatemir.Outcome{}, fmt.Errorf("publish native graph: %w", errors.Join(err, ctx.Err()))
	}
	out := contractstatemir.Outcome{Exit: contractstatemir.ExitCommit, Reads: decoded.Reads, Cost: decoded.Cost,
		Receipt: contractstate.CommitReceipt{LayoutHash: receipt.LayoutHash, Version: receipt.Version,
			Writes: receipt.Writes, WriteBytes: decoded.WriteBytes}}
	if receipt.LayoutHash != layout || receipt.Version != expectedVersion || receipt.Writes != len(decoded.Writes) {
		return out, PublicationReceiptError{Receipt: receipt}
	}
	return out, nil
}
