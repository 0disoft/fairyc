package contractstatenative

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/target"
)

type executionPort struct {
	contractstate.PersistencePort
	snapshot                    contractstate.PersistentSnapshot
	snapshots, publishes        int
	snapshotError, publishError error
	afterSnapshot               func()
	onPublish                   func(contractstate.LayoutHash, uint64, []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error)
}

func (port *executionPort) Snapshot(contractstate.LayoutHash) (contractstate.PersistentSnapshot, error) {
	port.snapshots++
	if port.afterSnapshot != nil {
		port.afterSnapshot()
	}
	return port.snapshot, port.snapshotError
}
func (port *executionPort) CompareAndPublish(layout contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
	port.publishes++
	if port.onPublish != nil {
		return port.onPublish(layout, version, writes)
	}
	if port.publishError != nil {
		return contractstate.PersistentReceipt{}, port.publishError
	}
	if len(writes) != 0 {
		version++
	}
	return contractstate.PersistentReceipt{LayoutHash: layout, Version: version, Writes: len(writes)}, nil
}

// Controlled process/storage ports exercise host failure boundaries only.
// Separate Windows tests run actual generated C against real local storage.
func TestPersistentExecutionBoundaryFailures(t *testing.T) {
	_, prepared, snapshot := persistentFixture(t, contractstatemir.ExitCommit, false)
	snapshot.Version = 7
	cause := errors.New("injected process/storage failure")
	for _, name := range []string{"zero-built-owner", "bad-limits", "canceled-before", "snapshot-error", "snapshot-cancel", "bad-layout", "foreign-key", "process-error", "process-cancel-error", "cancel-after-process", "exit", "stderr", "trap-channel", "short-response", "conflict", "layout-drift", "backend", "version-exhausted"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			port := &executionPort{snapshot: snapshot}
			limits := persistentLimits()
			calls := 0
			wantSnapshots, wantRuns, wantPublishes := 1, 1, 0
			var want error
			switch name {
			case "zero-built-owner":
				wantSnapshots, wantRuns = 0, 0
			case "bad-limits":
				limits.MaxWrites = 0
				wantSnapshots, wantRuns = 0, 0
			case "canceled-before":
				cancel()
				want = context.Canceled
				wantSnapshots, wantRuns = 0, 0
			case "snapshot-error":
				port.snapshotError = cause
				want = cause
				wantRuns = 0
			case "snapshot-cancel":
				port.afterSnapshot = cancel
				want = context.Canceled
				wantRuns = 0
			case "bad-layout":
				port.snapshot.LayoutHash[0] ^= 1
				want = contractstate.ErrLayoutMismatch
				wantRuns = 0
			case "foreign-key":
				port.snapshot.Entries = []contractstate.Entry{{Key: contractstate.Key{}, Value: make([]byte, 8)}}
				wantRuns = 0
			case "process-error":
				want = cause
			case "process-cancel-error":
				want = cause
			case "cancel-after-process":
				want = context.Canceled
			case "conflict":
				port.publishError = contractstate.ErrConflict
				want = contractstate.ErrConflict
				wantPublishes = 1
			case "layout-drift":
				port.publishError = contractstate.ErrLayoutMismatch
				want = contractstate.ErrLayoutMismatch
				wantPublishes = 1
			case "backend":
				port.publishError = cause
				want = cause
				wantPublishes = 1
			case "version-exhausted":
				port.snapshot.Version = math.MaxUint64
			}
			run := func(_ context.Context, input PersistentInput) (cdriver.WindowsHostOutcome, error) {
				calls++
				if name == "process-cancel-error" {
					cancel()
					return cdriver.WindowsHostOutcome{}, cause
				}
				if name == "process-error" {
					return cdriver.WindowsHostOutcome{}, cause
				}
				if name == "cancel-after-process" {
					cancel()
				}
				value := []byte{0, 0, 0, 0, 0, 0, 0, 42}
				reads := []contractstatemir.Read{{Operation: 1, Found: true, Value: contractstatemir.Value{Encoded: value}}}
				out := cdriver.WindowsHostOutcome{Stdout: string(persistentResponse(prepared, port.snapshot.Version, 0, 1, 1, value, reads))}
				switch name {
				case "exit":
					out.ExitCode = 4
				case "stderr":
					out.Stderr = "unexpected"
				case "trap-channel":
					out.TrapControl = "unexpected"
				case "short-response":
					out.Stdout = out.Stdout[:len(out.Stdout)-1]
				}
				return out, nil
			}
			var err error
			if name == "zero-built-owner" {
				_, err = ExecutePersistent(ctx, port, PersistentExecutable{}, limits)
			} else {
				_, err = executePersistent(ctx, port, prepared, limits, run)
			}
			if err == nil || want != nil && !errors.Is(err, want) {
				t.Fatalf("error %v, want cause %v", err, want)
			}
			if name == "process-cancel-error" && !errors.Is(err, context.Canceled) {
				t.Fatal("lost cancellation", err)
			}
			if port.snapshots != wantSnapshots || calls != wantRuns || port.publishes != wantPublishes {
				t.Fatalf("snapshot/run/publish %d/%d/%d", port.snapshots, calls, port.publishes)
			}
		})
	}
}

func TestPersistentExecutionAcknowledgement(t *testing.T) {
	_, prepared, snapshot := persistentFixture(t, contractstatemir.ExitCommit, false)
	snapshot.Version = 7
	for _, name := range []string{"success", "cancel-during-publish", "receipt-layout", "receipt-version", "receipt-writes"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			port := &executionPort{snapshot: snapshot}
			port.onPublish = func(layout contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
				if layout != prepared.LayoutHash() || version != 7 || len(writes) != 1 || writes[0].Key != prepared.key || writes[0].Deleted || len(writes[0].Value) != 8 || writes[0].Value[7] != 42 {
					t.Fatal("wrong native overlay")
				}
				receipt := contractstate.PersistentReceipt{LayoutHash: layout, Version: 8, Writes: 1}
				switch name {
				case "cancel-during-publish":
					cancel()
				case "receipt-layout":
					receipt.LayoutHash[0] ^= 1
				case "receipt-version":
					receipt.Version = 9
				case "receipt-writes":
					receipt.Writes = 0
				}
				return receipt, nil
			}
			out, err := executePersistent(ctx, port, prepared, persistentLimits(), func(context.Context, PersistentInput) (cdriver.WindowsHostOutcome, error) {
				value := []byte{0, 0, 0, 0, 0, 0, 0, 42}
				return cdriver.WindowsHostOutcome{Stdout: string(persistentResponse(prepared, 7, 0, 1, 1, value, []contractstatemir.Read{{Operation: 1, Found: true, Value: contractstatemir.Value{Encoded: value}}}))}, nil
			})
			if strings.HasPrefix(name, "receipt-") {
				var acknowledged PublicationReceiptError
				if !errors.As(err, &acknowledged) || out.Exit != contractstatemir.ExitCommit || !strings.Contains(err.Error(), "acknowledged") {
					t.Fatal("lost acknowledged publication", out, err)
				}
			} else if err != nil || out.Receipt.Version != 8 || out.Receipt.WriteBytes != 41 {
				t.Fatal(out, err)
			}
			if port.snapshots != 1 || port.publishes != 1 {
				t.Fatal("execution retried", port)
			}
		})
	}
}

func TestPersistentExecutionNoncommit(t *testing.T) {
	for _, status := range []PersistentStatus{PersistentReject, PersistentTrap, PersistentWriteBytesLimit, PersistentValueLimit, PersistentRuntimeFailure} {
		t.Run(string(rune('0'+status)), func(t *testing.T) {
			exit := contractstatemir.ExitCommit
			if status == PersistentReject {
				exit = contractstatemir.ExitReject
			} else if status == PersistentTrap {
				exit = contractstatemir.ExitTrap
			}
			_, prepared, snapshot := persistentFixture(t, exit, false)
			snapshot.Version = 7
			port := &executionPort{snapshot: snapshot}
			run := func(context.Context, PersistentInput) (cdriver.WindowsHostOutcome, error) {
				var reads []contractstatemir.Read
				if status <= PersistentTrap {
					reads = []contractstatemir.Read{{Operation: 1, Found: true, Value: contractstatemir.Value{Encoded: make([]byte, 8)}}}
				}
				return cdriver.WindowsHostOutcome{Stdout: string(persistentResponse(prepared, 7, byte(status), 0, 1, snapshot.Entries[0].Value, reads))}, nil
			}
			var recovered any
			var out contractstatemir.Outcome
			var err error
			func() {
				defer func() { recovered = recover() }()
				out, err = executePersistent(context.Background(), port, prepared, persistentLimits(), run)
			}()
			switch status {
			case PersistentReject:
				if err != nil || out.Exit != contractstatemir.ExitReject {
					t.Fatal(out, err)
				}
			case PersistentTrap:
				if trap, ok := recovered.(contractstatemir.Trap); !ok || trap.Category != "overflow" {
					t.Fatal("lost typed trap", recovered)
				}
			case PersistentWriteBytesLimit:
				if !errors.Is(err, contractstate.ErrWriteBytesLimit) {
					t.Fatal(err)
				}
			case PersistentValueLimit:
				if !errors.Is(err, contractstate.ErrValueTooLarge) {
					t.Fatal(err)
				}
			case PersistentRuntimeFailure:
				if err == nil {
					t.Fatal("runtime failure accepted")
				}
			}
			if port.snapshots != 1 || port.publishes != 0 {
				t.Fatal("noncommit published", port)
			}
		})
	}
}

func TestPersistentExecutionNoopVersionBoundary(t *testing.T) {
	input, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	input.StateTransactions[0].Accesses = nil
	handoff, err := Prepare(input, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := EmitPersistent(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []uint64{0, math.MaxUint64 - 1, math.MaxUint64} {
		port := &executionPort{snapshot: contractstate.PersistentSnapshot{LayoutHash: prepared.LayoutHash(), Version: version}}
		calls := 0
		out, err := executePersistent(context.Background(), port, prepared, persistentLimits(), func(context.Context, PersistentInput) (cdriver.WindowsHostOutcome, error) {
			calls++
			return cdriver.WindowsHostOutcome{Stdout: string(persistentResponse(prepared, version, 0, 0, 0, nil, nil))}, nil
		})
		if err != nil || out.Receipt.Version != version || out.Receipt.Writes != 0 || out.Receipt.WriteBytes != 0 || port.snapshots != 1 || port.publishes != 1 || calls != 1 {
			t.Fatalf("no-op at version %d: %+v/%v, snapshot/run/publish %d/%d/%d", version, out, err, port.snapshots, calls, port.publishes)
		}
	}
}
