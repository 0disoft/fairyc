package contractstatenative

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

// PersistentExecutable owns a compiler-prepared protocol and its compiled image.
// No caller-supplied source, response or executable bytes grant publish authority.
type PersistentExecutable struct {
	prepared PersistentArtifact
	image    cdriver.WindowsHostArtifact
	valid    bool
}

func BuildPersistent(prepared PersistentArtifact, profile cdriver.Profile) (PersistentExecutable, error) {
	if !prepared.valid {
		return PersistentExecutable{}, fmt.Errorf("native persistent build requires a prepared owner")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return PersistentExecutable{}, fmt.Errorf("resolve native state compiler: %w", err)
	}
	runtimeSource, err := cstdlib.ContractStateRuntimeSource()
	if err != nil {
		return PersistentExecutable{}, fmt.Errorf("select native state runtime: %w", err)
	}
	image, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, prepared.Source(), runtimeSource)
	if err != nil {
		return PersistentExecutable{}, fmt.Errorf("build native state image: %w", err)
	}
	image.Bytes = slices.Clone(image.Bytes)
	return PersistentExecutable{prepared: prepared, image: image, valid: true}, nil
}

// PublicationReceiptError follows a port's success acknowledgement. Publication
// may already be durable; callers must inspect the receipt and never blindly retry.
type PublicationReceiptError struct {
	Receipt contractstate.PersistentReceipt
}

func (err PublicationReceiptError) Error() string {
	return "native state publication was acknowledged with an inconsistent receipt; inspect stored state before retrying"
}

// ExecutePersistent performs one snapshot, one real native execution and at most
// one layout/version CAS. Compilation happens separately and touches no store.
func ExecutePersistent(ctx context.Context, store contractstate.PersistencePort, executable PersistentExecutable, limits contractstate.Limits) (contractstatemir.Outcome, error) {
	if !executable.valid || len(executable.image.Bytes) == 0 {
		return contractstatemir.Outcome{}, fmt.Errorf("native persistent execution requires a built owner")
	}
	return executePersistent(ctx, store, executable.prepared, limits, func(ctx context.Context, input PersistentInput) (cdriver.WindowsHostOutcome, error) {
		return cdriver.ExecuteWindowsMSVCTargetWithContext(ctx, executable.image, bytes.NewReader(input.Bytes()), time.Minute)
	})
}

type persistentExecution func(context.Context, PersistentInput) (cdriver.WindowsHostOutcome, error)

func executePersistent(ctx context.Context, store contractstate.PersistencePort, prepared PersistentArtifact, limits contractstate.Limits, execute persistentExecution) (contractstatemir.Outcome, error) {
	if ctx == nil || store == nil || execute == nil {
		return contractstatemir.Outcome{}, fmt.Errorf("native persistence requires context, storage and execution")
	}
	// Validate the complete preparation and limit policy before any store I/O.
	if _, err := prepared.Input(contractstate.PersistentSnapshot{LayoutHash: prepared.LayoutHash()}, limits); err != nil {
		return contractstatemir.Outcome{}, err
	}
	if err := ctx.Err(); err != nil {
		return contractstatemir.Outcome{}, err
	}
	snapshot, err := store.Snapshot(prepared.LayoutHash())
	if err != nil {
		return contractstatemir.Outcome{}, fmt.Errorf("snapshot native state: %w", errors.Join(err, ctx.Err()))
	}
	if err := ctx.Err(); err != nil {
		return contractstatemir.Outcome{}, err
	}
	input, err := prepared.Input(snapshot, limits)
	if err != nil {
		return contractstatemir.Outcome{}, err
	}
	version := snapshot.Version
	result, err := execute(ctx, input)
	if err != nil {
		return contractstatemir.Outcome{}, fmt.Errorf("execute native state: %w", errors.Join(err, ctx.Err()))
	}
	if err := ctx.Err(); err != nil {
		return contractstatemir.Outcome{}, err
	}
	if result.ExitCode != 0 || result.Stderr != "" || result.TrapControl != "" {
		return contractstatemir.Outcome{}, fmt.Errorf("native state process failed its protocol boundary (exit %d)", result.ExitCode)
	}
	decoded, err := prepared.Decode(input, []byte(result.Stdout))
	if err != nil {
		return contractstatemir.Outcome{}, err
	}
	switch decoded.Status {
	case PersistentReject:
		return contractstatemir.Outcome{Exit: contractstatemir.ExitReject, Reads: decoded.Reads}, nil
	case PersistentTrap:
		panic(contractstatemir.Trap{Category: decoded.TrapCategory})
	case PersistentWriteBytesLimit:
		return contractstatemir.Outcome{}, fmt.Errorf("native state intermediate write: %w", contractstate.ErrWriteBytesLimit)
	case PersistentValueLimit:
		return contractstatemir.Outcome{}, fmt.Errorf("native state intermediate value: %w", contractstate.ErrValueTooLarge)
	case PersistentRuntimeFailure:
		return contractstatemir.Outcome{}, fmt.Errorf("native state runtime rejected execution")
	}
	var writes []contractstate.PersistentWrite
	var writeBytes uint64
	if decoded.Dirty {
		if version == math.MaxUint64 {
			return contractstatemir.Outcome{}, fmt.Errorf("native state version exhausted")
		}
		writes = []contractstate.PersistentWrite{{Key: prepared.key, Value: slices.Clone(decoded.Value), Deleted: !decoded.Present}}
		writeBytes = uint64(len(contractstate.Key{})) + 1 + uint64(len(decoded.Value))
	}
	if err := ctx.Err(); err != nil {
		return contractstatemir.Outcome{}, err
	}
	receipt, err := store.CompareAndPublish(prepared.LayoutHash(), version, writes)
	if err != nil {
		return contractstatemir.Outcome{}, fmt.Errorf("publish native state: %w", err)
	}
	outcome := contractstatemir.Outcome{Exit: contractstatemir.ExitCommit, Reads: decoded.Reads, Receipt: contractstate.CommitReceipt{LayoutHash: receipt.LayoutHash, Version: receipt.Version, Writes: receipt.Writes, WriteBytes: writeBytes}}
	expectedVersion := version
	if decoded.Dirty {
		expectedVersion++
	}
	// Never convert acknowledged publication into an ordinary cancellation/error.
	if receipt.LayoutHash != prepared.LayoutHash() || receipt.Version != expectedVersion || receipt.Writes != len(writes) {
		return outcome, PublicationReceiptError{Receipt: receipt}
	}
	return outcome, nil
}
