package contractstatenative

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/target"
)

func overlayPersistentLimits() contractstate.Limits {
	return contractstate.Limits{MaxWrites: 16, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64}
}

func TestOverlayPersistenceBoundaryFailures(t *testing.T) {
	handoff, prepared, _, layout := overlayWireFixture(t, contractstatemir.ExitCommit)
	cause := errors.New("overlay boundary failure")
	for _, name := range []string{"zero-built", "zero-prepared", "nil-context", "nil-store", "nil-run", "bad-limits", "canceled-before", "snapshot-error", "snapshot-cancel-error", "snapshot-cancel", "bad-layout", "duplicate-snapshot", "bad-canonical", "process-error", "process-cancel-error", "cancel-after-process", "exit", "stderr", "trap-channel", "short-response", "foreign-entry", "conflict", "layout-drift", "backend", "publish-cancel-error", "version-exhausted"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			port := &executionPort{snapshot: contractstate.PersistentSnapshot{LayoutHash: layout.Hash(), Version: 7}}
			limits := overlayPersistentLimits()
			owner := prepared
			wantSnapshots, wantRuns, wantPublishes := 1, 1, 0
			var want error
			switch name {
			case "zero-built", "zero-prepared", "nil-context", "nil-store", "nil-run":
				wantSnapshots, wantRuns = 0, 0
			case "bad-limits":
				limits.MaxWrites = 0
				wantSnapshots, wantRuns = 0, 0
			case "canceled-before":
				cancel()
				want = context.Canceled
				wantSnapshots, wantRuns = 0, 0
			case "snapshot-error", "snapshot-cancel-error":
				port.snapshotError, want = cause, cause
				wantRuns = 0
				if name == "snapshot-cancel-error" {
					port.afterSnapshot = cancel
				}
			case "snapshot-cancel":
				port.afterSnapshot, want = cancel, context.Canceled
				wantRuns = 0
			case "bad-layout":
				port.snapshot.LayoutHash[0] ^= 1
				want, wantRuns = contractstate.ErrLayoutMismatch, 0
			case "duplicate-snapshot":
				entry := contractstate.Entry{Key: layout.Fields()[0].Prefix, Value: []byte{1}}
				port.snapshot.Entries = []contractstate.Entry{entry, entry}
				wantRuns = 0
			case "bad-canonical":
				port.snapshot.Entries = []contractstate.Entry{{Key: layout.Fields()[0].Prefix, Value: []byte{2}}}
				wantRuns = 0
			case "process-error", "process-cancel-error":
				want = cause
			case "cancel-after-process":
				want = context.Canceled
			case "conflict":
				port.publishError, want = contractstate.ErrConflict, contractstate.ErrConflict
				wantPublishes = 1
			case "layout-drift":
				port.publishError, want = contractstate.ErrLayoutMismatch, contractstate.ErrLayoutMismatch
				wantPublishes = 1
			case "backend", "publish-cancel-error":
				port.publishError, want = cause, cause
				wantPublishes = 1
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
			run := overlayExecution(func(context.Context, OverlayInput) (cdriver.WindowsHostOutcome, error) {
				calls++
				if name == "process-cancel-error" {
					cancel()
				}
				if name == "process-error" || name == "process-cancel-error" {
					return cdriver.WindowsHostOutcome{}, cause
				}
				if name == "cancel-after-process" {
					cancel()
				}
				response := overlayWireResponse(t, handoff, layout, port.snapshot.Version)
				out := cdriver.WindowsHostOutcome{}
				switch name {
				case "exit":
					out.ExitCode = 4
				case "stderr":
					out.Stderr = "unexpected"
				case "trap-channel":
					out.TrapControl = "unexpected"
				case "short-response":
					response = response[:len(response)-1]
				case "foreign-entry":
					response[72] ^= 1
				}
				out.Stdout = string(response)
				return out, nil
			})
			var err error
			switch name {
			case "zero-built":
				_, err = ExecuteOverlayPersistent(ctx, port, OverlayExecutable{}, limits)
			case "zero-prepared":
				owner = OverlayTransport{}
			case "nil-context":
				ctx = nil
			case "nil-run":
				run = nil
			}
			if name == "nil-store" {
				_, err = executeOverlayPersistent(ctx, nil, owner, limits, run)
			} else if name != "zero-built" {
				_, err = executeOverlayPersistent(ctx, port, owner, limits, run)
			}
			if err == nil || want != nil && !errors.Is(err, want) {
				t.Fatalf("error %v, want cause %v", err, want)
			}
			if strings.HasSuffix(name, "cancel-error") && !errors.Is(err, context.Canceled) {
				t.Fatal("lost simultaneous cancellation", err)
			}
			if port.snapshots != wantSnapshots || calls != wantRuns || port.publishes != wantPublishes {
				t.Fatalf("snapshot/run/CAS %d/%d/%d, want %d/%d/%d", port.snapshots, calls, port.publishes, wantSnapshots, wantRuns, wantPublishes)
			}
		})
	}
}

func TestOverlayPersistenceAcknowledgement(t *testing.T) {
	handoff, prepared, _, layout := overlayWireFixture(t, contractstatemir.ExitCommit)
	for _, name := range []string{"success", "cancel-during-publish", "receipt-layout", "receipt-version", "receipt-writes"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			port := &executionPort{snapshot: contractstate.PersistentSnapshot{LayoutHash: layout.Hash(), Version: 7}}
			port.onPublish = func(hash contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
				if hash != layout.Hash() || version != 7 || len(writes) != 14 {
					t.Fatal("wrong complete CAS overlay", hash, version, len(writes))
				}
				receipt := contractstate.PersistentReceipt{LayoutHash: hash, Version: 8, Writes: 14}
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
			out, err := executeOverlayPersistent(ctx, port, prepared, overlayPersistentLimits(), func(context.Context, OverlayInput) (cdriver.WindowsHostOutcome, error) {
				return cdriver.WindowsHostOutcome{Stdout: string(overlayWireResponse(t, handoff, layout, 7))}, nil
			})
			if strings.HasPrefix(name, "receipt-") {
				var acknowledged PublicationReceiptError
				if !errors.As(err, &acknowledged) || out.Exit != contractstatemir.ExitCommit || acknowledged.Receipt != (contractstate.PersistentReceipt{LayoutHash: out.Receipt.LayoutHash, Version: out.Receipt.Version, Writes: out.Receipt.Writes}) {
					t.Fatal("lost acknowledged receipt", out, err)
				}
			} else if err != nil || out.Receipt.Version != 8 || out.Receipt.Writes != 14 || out.Receipt.WriteBytes == 0 {
				t.Fatal(out, err)
			}
			if port.snapshots != 1 || port.publishes != 1 {
				t.Fatal("acknowledged execution retried", port)
			}
		})
	}
}

func TestOverlayPersistenceFailuresNeverPublish(t *testing.T) {
	for _, status := range []byte{1, 2, 3, 4, 5, 6, 7, 8, 9} {
		t.Run(string(rune('0'+status)), func(t *testing.T) {
			exit := contractstatemir.ExitCommit
			if status == 8 {
				exit = contractstatemir.ExitReject
			} else if status == 9 {
				exit = contractstatemir.ExitTrap
			}
			handoff, prepared, _, layout := overlayWireFixture(t, exit)
			port := &executionPort{snapshot: contractstate.PersistentSnapshot{LayoutHash: layout.Hash(), Version: 7}}
			var recovered any
			var out contractstatemir.Outcome
			var err error
			func() {
				defer func() { recovered = recover() }()
				out, err = executeOverlayPersistent(context.Background(), port, prepared, overlayPersistentLimits(), func(context.Context, OverlayInput) (cdriver.WindowsHostOutcome, error) {
					packet := overlayWireResponse(t, handoff, layout, 7)
					if status < 8 {
						packet = packet[:142]
						packet[112], packet[113] = status, 0
						clear(packet[114:142])
					}
					return cdriver.WindowsHostOutcome{Stdout: string(packet)}, nil
				})
			}()
			if status == 8 {
				if err != nil || recovered != nil || out.Exit != contractstatemir.ExitReject {
					t.Fatal(out, err, recovered)
				}
			} else if status == 9 {
				if trap, ok := recovered.(contractstatemir.Trap); !ok || trap.Category != handoff.program.TrapCategory {
					t.Fatal("lost typed trap", recovered)
				}
			} else {
				want := map[byte]error{3: contractstate.ErrValueTooLarge, 4: contractstate.ErrWriteLimit, 5: contractstate.ErrWriteBytesLimit, 7: contractstate.ErrMapKeyTooLarge}[status]
				if err == nil || recovered != nil || want != nil && !errors.Is(err, want) {
					t.Fatal("runtime failure lost cause or published", status, err, recovered)
				}
			}
			if port.snapshots != 1 || port.publishes != 0 {
				t.Fatal("noncommit reached CAS", port.snapshots, port.publishes)
			}
		})
	}
}

func TestOverlayPersistenceNoopVersionBoundary(t *testing.T) {
	handoff, layout := overlayMIRFixture(t, contractstatemir.ExitCommit, true)
	prepared, err := PrepareOverlayTransport(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []uint64{0, math.MaxUint64 - 1, math.MaxUint64} {
		for _, conflict := range []bool{false, true} {
			port := &executionPort{snapshot: contractstate.PersistentSnapshot{LayoutHash: layout.Hash(), Version: version}}
			if conflict {
				port.publishError = contractstate.ErrConflict
			}
			calls := 0
			out, err := executeOverlayPersistent(context.Background(), port, prepared, overlayPersistentLimits(), func(context.Context, OverlayInput) (cdriver.WindowsHostOutcome, error) {
				calls++
				packet := overlayWireResponse(t, handoff, layout, 0)
				binary.BigEndian.PutUint64(packet[104:112], version)
				return cdriver.WindowsHostOutcome{Stdout: string(packet)}, nil
			})
			if conflict {
				if !errors.Is(err, contractstate.ErrConflict) {
					t.Fatal("empty commit skipped CAS", err)
				}
			} else if err != nil || out.Receipt.Version != version || out.Receipt.Writes != 0 || out.Receipt.WriteBytes != 0 {
				t.Fatal("empty commit changed version", version, out, err)
			}
			if port.snapshots != 1 || port.publishes != 1 || calls != 1 {
				t.Fatal("empty commit retried", port.snapshots, port.publishes, calls)
			}
		}
	}
}
