package contractstatenative

import (
	"context"
	"crypto/sha256"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/target"
)

// Rebind the separately authored wire fixture to the exact request issued by
// the executor. Its event/overlay bytes are never produced by the decoder.
func graphReplyForInput(input GraphInput, template []byte) []byte {
	frame := slices.Clone(template)
	request := input.Bytes()
	copy(frame[8:112], request[8:112])
	digest := sha256.Sum256(request)
	copy(frame[112:144], digest[:])
	return frame
}

func TestGraphPersistenceBoundaryFailures(t *testing.T) {
	h, p, _, template := graphWireFixture(t, contractstatemir.ExitCommit, false)
	cause := errors.New("graph boundary failure")
	for _, name := range []string{"zero-built", "empty-image", "zero-prepared", "nil-context", "nil-store", "nil-run", "bad-limits", "canceled-before", "snapshot-error", "snapshot-cancel-error", "snapshot-cancel", "bad-layout", "duplicate-snapshot", "bad-canonical", "process-error", "process-cancel-error", "cancel-after-process", "exit", "stderr", "trap-channel", "short-response", "foreign-entry", "request-hash", "cost", "conflict", "layout-drift", "backend", "publish-cancel-error", "version-exhausted"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			port := &executionPort{snapshot: contractstate.PersistentSnapshot{LayoutHash: p.entry.layout, Version: 7}}
			limits, owner := overlayPersistentLimits(), p
			wantSnapshots, wantRuns, wantPublishes := 1, 1, 0
			var want error
			switch name {
			case "zero-built", "empty-image", "zero-prepared", "nil-context", "nil-store", "nil-run":
				wantSnapshots, wantRuns = 0, 0
			case "bad-limits":
				limits.MaxWrites, wantSnapshots, wantRuns = 0, 0, 0
			case "canceled-before":
				cancel()
				want, wantSnapshots, wantRuns = context.Canceled, 0, 0
			case "snapshot-error", "snapshot-cancel-error":
				port.snapshotError, want, wantRuns = cause, cause, 0
				if name == "snapshot-cancel-error" {
					port.afterSnapshot = cancel
				}
			case "snapshot-cancel":
				port.afterSnapshot, want, wantRuns = cancel, context.Canceled, 0
			case "bad-layout":
				port.snapshot.LayoutHash[0] ^= 1
				want, wantRuns = contractstate.ErrLayoutMismatch, 0
			case "duplicate-snapshot":
				e := contractstate.Entry{Key: h.program.Layout.Fields[0].Prefix, Value: make([]byte, 8)}
				port.snapshot.Entries, wantRuns = []contractstate.Entry{e, e}, 0
			case "bad-canonical":
				port.snapshot.Entries = []contractstate.Entry{{Key: h.program.Layout.Fields[0].Prefix, Value: []byte{0}}}
				wantRuns = 0
			case "process-error", "process-cancel-error":
				want = cause
			case "cancel-after-process":
				want = context.Canceled
			case "conflict":
				port.publishError, want, wantPublishes = contractstate.ErrConflict, contractstate.ErrConflict, 1
			case "layout-drift":
				port.publishError, want, wantPublishes = contractstate.ErrLayoutMismatch, contractstate.ErrLayoutMismatch, 1
			case "backend", "publish-cancel-error":
				port.publishError, want, wantPublishes = cause, cause, 1
				if name == "publish-cancel-error" {
					port.onPublish = func(contractstate.LayoutHash, uint64, []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
						cancel()
						return contractstate.PersistentReceipt{}, cause
					}
				}
			case "version-exhausted":
				port.snapshot.Version = math.MaxUint64
			}
			calls := 0
			run := graphExecution(func(_ context.Context, input GraphInput) (cdriver.WindowsHostOutcome, error) {
				calls++
				if name == "process-cancel-error" || name == "cancel-after-process" {
					cancel()
				}
				if name == "process-error" || name == "process-cancel-error" {
					return cdriver.WindowsHostOutcome{}, cause
				}
				frame := graphReplyForInput(input, template)
				out := cdriver.WindowsHostOutcome{}
				switch name {
				case "exit":
					out.ExitCode = 4
				case "stderr":
					out.Stderr = "unexpected"
				case "trap-channel":
					out.TrapControl = "unexpected"
				case "short-response":
					frame = frame[:len(frame)-1]
				case "foreign-entry":
					frame[72] ^= 1
				case "request-hash":
					frame[112] ^= 1
				case "cost":
					frame[157] = 1
				}
				out.Stdout = string(frame)
				return out, nil
			})
			var err error
			switch name {
			case "zero-built":
				_, err = ExecuteGraphPersistent(ctx, port, GraphExecutable{}, limits)
			case "empty-image":
				_, err = ExecuteGraphPersistent(ctx, port, GraphExecutable{valid: true, prepared: p}, limits)
			case "zero-prepared":
				owner = GraphTransport{}
			case "nil-context":
				ctx = nil
			case "nil-run":
				run = nil
			}
			if name == "nil-store" {
				_, err = executeGraphPersistent(ctx, nil, owner, limits, run)
			} else if name != "zero-built" && name != "empty-image" {
				_, err = executeGraphPersistent(ctx, port, owner, limits, run)
			}
			if err == nil || want != nil && !errors.Is(err, want) {
				t.Fatal("lost boundary failure", err, want)
			}
			if strings.HasSuffix(name, "cancel-error") && !errors.Is(err, context.Canceled) {
				t.Fatal("lost simultaneous cancellation cause", err)
			}
			if port.snapshots != wantSnapshots || calls != wantRuns || port.publishes != wantPublishes {
				t.Fatalf("snapshot/run/CAS %d/%d/%d, want %d/%d/%d", port.snapshots, calls, port.publishes, wantSnapshots, wantRuns, wantPublishes)
			}
		})
	}
}

func TestGraphPersistenceAcknowledgement(t *testing.T) {
	_, p, _, template := graphWireFixture(t, contractstatemir.ExitCommit, true)
	for _, name := range []string{"success", "cancel-during-publish", "receipt-layout", "receipt-version", "receipt-writes"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			port := &executionPort{snapshot: contractstate.PersistentSnapshot{LayoutHash: p.entry.layout, Version: 7}}
			port.onPublish = func(layout contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
				if layout != p.entry.layout || version != 7 || len(writes) != 1 || writes[0].Deleted || len(writes[0].Value) != 8 || writes[0].Value[7] != 42 {
					t.Fatal("wrong admitted CAS overlay", layout, version, writes)
				}
				receipt := contractstate.PersistentReceipt{LayoutHash: layout, Version: 8, Writes: 1}
				switch name {
				case "cancel-during-publish":
					cancel()
				case "receipt-layout":
					receipt.LayoutHash[0] ^= 1
				case "receipt-version":
					receipt.Version++
				case "receipt-writes":
					receipt.Writes--
				}
				return receipt, nil
			}
			calls := 0
			out, err := executeGraphPersistent(ctx, port, p, overlayPersistentLimits(), func(_ context.Context, input GraphInput) (cdriver.WindowsHostOutcome, error) {
				calls++
				return cdriver.WindowsHostOutcome{Stdout: string(graphReplyForInput(input, template))}, nil
			})
			if strings.HasPrefix(name, "receipt-") {
				var published PublicationReceiptError
				if !errors.As(err, &published) || out.Exit != contractstatemir.ExitCommit || published.Receipt != (contractstate.PersistentReceipt{LayoutHash: out.Receipt.LayoutHash, Version: out.Receipt.Version, Writes: out.Receipt.Writes}) {
					t.Fatal("lost acknowledged receipt", out, err)
				}
			} else if err != nil || out.Receipt.Version != 8 || out.Receipt.Writes != 1 || out.Receipt.WriteBytes != 41 {
				t.Fatal(out, err)
			}
			if out.Cost != 1 || len(out.Reads) != 1 || out.Reads[0].Operation != 2 || port.snapshots != 1 || port.publishes != 1 || calls != 1 {
				t.Fatal("cost, observations or one-attempt acknowledgement lost", out, port.snapshots, port.publishes, calls)
			}
		})
	}
}

func TestGraphPersistenceFailuresNeverPublish(t *testing.T) {
	_, p, _, _ := graphWireFixture(t, contractstatemir.ExitCommit, false)
	causes := map[byte]error{2: contractstate.ErrClosed, 3: contractstate.ErrValueTooLarge, 4: contractstate.ErrWriteLimit, 5: contractstate.ErrWriteBytesLimit, 7: contractstate.ErrMapKeyTooLarge, 9: contractstate.ErrBorrowInvalidated}
	for status := byte(1); status <= 11; status++ {
		port := &executionPort{snapshot: contractstate.PersistentSnapshot{LayoutHash: p.entry.layout, Version: 7}}
		out, err := executeGraphPersistent(context.Background(), port, p, overlayPersistentLimits(), func(_ context.Context, input GraphInput) (cdriver.WindowsHostOutcome, error) {
			return cdriver.WindowsHostOutcome{Stdout: string(graphWireResponse(input, status, 0, 0, 0, 0, 0, nil, nil))}, nil
		})
		var runtime GraphRuntimeError
		if !errors.As(err, &runtime) || runtime.Status != status || causes[status] != nil && !errors.Is(err, causes[status]) || out.Exit != "" || port.snapshots != 1 || port.publishes != 0 {
			t.Fatal("runtime failure published or lost cause", status, out, err)
		}
		if status == 8 && (errors.Is(err, contractstate.ErrValueTooLarge) || errors.Is(err, contractstate.ErrWriteBytesLimit)) {
			t.Fatal("native arena exhaustion became caller policy violation", err)
		}
	}
	for _, exit := range []contractstatemir.ExitKind{contractstatemir.ExitReject, contractstatemir.ExitTrap} {
		_, p, _, template := graphWireFixture(t, exit, true)
		port := &executionPort{snapshot: contractstate.PersistentSnapshot{LayoutHash: p.entry.layout}}
		var out contractstatemir.Outcome
		var err error
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			out, err = executeGraphPersistent(context.Background(), port, p, overlayPersistentLimits(), func(_ context.Context, input GraphInput) (cdriver.WindowsHostOutcome, error) {
				return cdriver.WindowsHostOutcome{Stdout: string(graphReplyForInput(input, template))}, nil
			})
		}()
		if exit == contractstatemir.ExitTrap {
			if trap, ok := recovered.(contractstatemir.Trap); !ok || trap.Category != "overflow" {
				t.Fatal("lost typed native trap", recovered)
			}
		} else if recovered != nil || err != nil || out.Exit != exit || out.Cost != 1 || len(out.Reads) != 1 {
			t.Fatal("rejected execution lost exact evidence", out, recovered, err)
		}
		if port.snapshots != 1 || port.publishes != 0 {
			t.Fatal("reject/trap reached CAS", port)
		}
	}
}

func TestGraphPersistenceNoopVersionBoundary(t *testing.T) {
	source, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	source.StateTransactions[0].Accesses = nil
	h, err := Prepare(source, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []uint64{0, math.MaxUint64 - 1, math.MaxUint64} {
		for _, conflict := range []bool{false, true} {
			port := &executionPort{snapshot: contractstate.PersistentSnapshot{LayoutHash: p.entry.layout, Version: version}}
			if conflict {
				port.publishError = contractstate.ErrConflict
			}
			calls := 0
			out, err := executeGraphPersistent(context.Background(), port, p, overlayPersistentLimits(), func(_ context.Context, input GraphInput) (cdriver.WindowsHostOutcome, error) {
				calls++
				trace := graphWireEvent(nil, 0, 0x80000001, nil, nil, 0)
				return cdriver.WindowsHostOutcome{Stdout: string(graphWireResponse(input, 0, 0, 0, 0, 0, 1, trace, nil))}, nil
			})
			if conflict && !errors.Is(err, contractstate.ErrConflict) || !conflict && (err != nil || out.Receipt.Version != version || out.Receipt.Writes != 0 || out.Receipt.WriteBytes != 0) {
				t.Fatal("empty commit lost version guard", version, conflict, out, err)
			}
			if port.snapshots != 1 || port.publishes != 1 || calls != 1 {
				t.Fatal("no-op commit skipped or retried CAS", port.snapshots, calls, port.publishes)
			}
		}
	}
}
