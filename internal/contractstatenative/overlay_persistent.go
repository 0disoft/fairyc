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

// ExecuteOverlayPersistent executes a sealed multi-key native image against one
// snapshot and publishes at most once through the layout/version CAS. It never
// re-executes a body after conflict or reverses acknowledged publication on cancel.
func ExecuteOverlayPersistent(ctx context.Context, store contractstate.PersistencePort, executable OverlayExecutable, limits contractstate.Limits) (contractstatemir.Outcome, error) {
	if !executable.valid || len(executable.image.Bytes) == 0 {
		return contractstatemir.Outcome{}, fmt.Errorf("native overlay persistence requires a built owner")
	}
	return executeOverlayPersistent(ctx, store, executable.prepared, limits, executable.run)
}

type overlayExecution func(context.Context, OverlayInput) (cdriver.WindowsHostOutcome, error)

func executeOverlayPersistent(ctx context.Context, store contractstate.PersistencePort, prepared OverlayTransport, limits contractstate.Limits, run overlayExecution) (contractstatemir.Outcome, error) {
	if ctx == nil || store == nil || run == nil {
		return contractstatemir.Outcome{}, fmt.Errorf("native overlay persistence requires context, storage and execution")
	}
	layout := prepared.entry.LayoutHash()
	// Preparation and policy must reject before any persistent observation.
	if _, err := prepared.Input(contractstate.PersistentSnapshot{LayoutHash: layout}, limits); err != nil {
		return contractstatemir.Outcome{}, err
	}
	if err := ctx.Err(); err != nil {
		return contractstatemir.Outcome{}, err
	}
	snapshot, err := store.Snapshot(layout)
	if err != nil {
		return contractstatemir.Outcome{}, fmt.Errorf("snapshot native overlay: %w", errors.Join(err, ctx.Err()))
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
		return contractstatemir.Outcome{}, fmt.Errorf("execute native overlay: %w", errors.Join(err, ctx.Err()))
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
		var cause error
		switch decoded.Status {
		case 3:
			cause = contractstate.ErrValueTooLarge
		case 4:
			cause = contractstate.ErrWriteLimit
		case 5:
			cause = contractstate.ErrWriteBytesLimit
		case 7:
			cause = contractstate.ErrMapKeyTooLarge
		default:
			return contractstatemir.Outcome{}, fmt.Errorf("native overlay runtime rejected execution (status %d)", decoded.Status)
		}
		return contractstatemir.Outcome{}, fmt.Errorf("native overlay execution: %w", cause)
	}
	if decoded.Exit == contractstatemir.ExitReject {
		return contractstatemir.Outcome{Exit: decoded.Exit, Reads: decoded.Reads}, nil
	}
	if decoded.Exit == contractstatemir.ExitTrap {
		panic(contractstatemir.Trap{Category: decoded.TrapCategory})
	}
	version := snapshot.Version
	expectedVersion := version
	if len(decoded.Writes) != 0 {
		if version == math.MaxUint64 {
			return contractstatemir.Outcome{}, fmt.Errorf("native overlay state version exhausted")
		}
		expectedVersion++
	}
	if err := ctx.Err(); err != nil {
		return contractstatemir.Outcome{}, err
	}
	receipt, err := store.CompareAndPublish(layout, version, decoded.Writes)
	if err != nil {
		return contractstatemir.Outcome{}, fmt.Errorf("publish native overlay: %w", errors.Join(err, ctx.Err()))
	}
	out := contractstatemir.Outcome{Exit: contractstatemir.ExitCommit, Reads: decoded.Reads, Receipt: contractstate.CommitReceipt{
		LayoutHash: receipt.LayoutHash, Version: receipt.Version, Writes: receipt.Writes, WriteBytes: decoded.WriteBytes,
	}}
	if receipt.LayoutHash != layout || receipt.Version != expectedVersion || receipt.Writes != len(decoded.Writes) {
		return out, PublicationReceiptError{Receipt: receipt}
	}
	return out, nil
}
