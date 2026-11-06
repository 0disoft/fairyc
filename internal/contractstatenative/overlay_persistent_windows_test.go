package contractstatenative

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/target"
)

func overlayLocalLimits() contractstate.LocalPersistenceLimits {
	return contractstate.LocalPersistenceLimits{MaxStateBytes: 1 << 20, MaxEntries: 64, MaxValueBytes: 1 << 20}
}

func runOverlayPersistentNative(ctx context.Context, store contractstate.PersistencePort, executable OverlayExecutable, limits contractstate.Limits) (out contractstatemir.Outcome, category string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if trap, ok := recovered.(contractstatemir.Trap); ok {
				category = trap.Category
			} else {
				panic(recovered)
			}
		}
	}()
	out, err = ExecuteOverlayPersistent(ctx, store, executable, limits)
	return
}

func sameOverlayOutcome(a, b contractstatemir.Outcome) bool {
	return a.Exit == b.Exit && a.Receipt == b.Receipt && a.Cost == b.Cost && slices.EqualFunc(a.Reads, b.Reads, func(a, b contractstatemir.Read) bool {
		return a.Operation == b.Operation && a.Found == b.Found && a.Value.TypeID == b.Value.TypeID && bytes.Equal(a.Value.Encoded, b.Value.Encoded)
	})
}

func sameOverlaySnapshot(a, b contractstate.PersistentSnapshot) bool {
	return a.LayoutHash == b.LayoutHash && a.Version == b.Version && slices.EqualFunc(a.Entries, b.Entries, func(a, b contractstate.Entry) bool {
		return a.Key == b.Key && bytes.Equal(a.Value, b.Value)
	})
}

func overlaySeedSnapshot(t *testing.T) contractstate.PersistentSnapshot {
	t.Helper()
	handoff, layout := overlayMIRFixture(t, contractstatemir.ExitCommit, false)
	store, err := contractstate.NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	program := handoff.program
	program.Operations = nil
	for _, op := range handoff.program.Operations {
		if op.Kind == contractstatemir.OperationCellWrite || op.Kind == contractstatemir.OperationMapWrite {
			program.Operations = append(program.Operations, op)
		}
	}
	// The transaction never mentions this second bool-map key. A complete
	// durable overlay must retain it without trying to infer its field from a hash.
	for _, op := range handoff.program.Operations {
		if op.Kind == contractstatemir.OperationMapWrite && op.Key.TypeID == contractstate.Bool.TypeID() {
			op.Key.Encoded, op.Value.Encoded = []byte{0}, []byte{0}
			program.Operations = append(program.Operations, op)
			break
		}
	}
	// Go prepares independent initial state only. Native outputs still come
	// exclusively from the generated C process and its admitted response.
	if _, err := contractstatemir.ExecutePersistent(store, program, overlayPersistentLimits()); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Snapshot(layout.Hash())
	if err != nil || len(snapshot.Entries) != 15 {
		t.Fatal("independent complete initial state", err)
	}
	return snapshot
}

func TestOverlayPersistenceNativeLocalStore(t *testing.T) {
	seed := overlaySeedSnapshot(t)
	for _, family := range []string{"commit", "reject", "trap", "empty"} {
		t.Run(family, func(t *testing.T) {
			exit := contractstatemir.ExitCommit
			if family == "reject" {
				exit = contractstatemir.ExitReject
			} else if family == "trap" {
				exit = contractstatemir.ExitTrap
			}
			handoff, layout := overlayMIRFixture(t, exit, family == "empty")
			prepared, err := PrepareOverlayTransport(handoff, target.X8664WindowsMSVC)
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					executable, err := BuildOverlay(prepared, profile)
					if err != nil {
						t.Fatal(err)
					}
					for _, present := range []bool{false, true} {
						t.Run(map[bool]string{false: "absent", true: "present"}[present], func(t *testing.T) {
							directory := t.TempDir()
							store, err := contractstate.CreateLocalPersistence(directory, layout, overlayLocalLimits())
							if err != nil {
								t.Fatal(err)
							}
							t.Cleanup(func() { _ = store.Close() })
							reference, err := contractstate.NewReferencePersistence(layout, nil)
							if err != nil {
								t.Fatal(err)
							}
							if present {
								var writes []contractstate.PersistentWrite
								for _, entry := range seed.Entries {
									writes = append(writes, contractstate.PersistentWrite{Key: entry.Key, Value: entry.Value})
								}
								for _, port := range []contractstate.PersistencePort{store, reference} {
									if _, err := port.CompareAndPublish(layout.Hash(), 0, writes); err != nil {
										t.Fatal(err)
									}
								}
							}
							for attempt := 0; attempt < 2; attempt++ {
								before := persistentStateBytes(t, directory)
								got, trap, err := runOverlayPersistentNative(context.Background(), store, executable, overlayPersistentLimits())
								want, wantTrap, wantErr := persistentOracle(reference, handoff.program, overlayPersistentLimits())
								if err != nil || wantErr != nil || trap != wantTrap || !sameOverlayOutcome(got, want) {
									t.Fatalf("native %+v/%q/%v; Go %+v/%q/%v", got, trap, err, want, wantTrap, wantErr)
								}
								if exit != contractstatemir.ExitCommit || family == "empty" {
									if !bytes.Equal(before, persistentStateBytes(t, directory)) {
										t.Fatal("rollback/no-op changed durable bytes")
									}
								}
								if err := store.Close(); err != nil {
									t.Fatal(err)
								}
								store, err = contractstate.OpenLocalPersistence(directory, layout, overlayLocalLimits())
								if err != nil {
									t.Fatal(err)
								}
								actual, err := store.Snapshot(layout.Hash())
								if err != nil {
									t.Fatal(err)
								}
								expected, err := reference.Snapshot(layout.Hash())
								if err != nil || !sameOverlaySnapshot(actual, expected) {
									t.Fatal("reopened state differs", actual, expected, err)
								}
							}
						})
					}
				})
			}
		})
	}
}

func TestOverlayPersistenceNativeCompetingWriters(t *testing.T) {
	_, prepared, _, layout := overlayWireFixture(t, contractstatemir.ExitCommit)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			executable, err := BuildOverlay(prepared, profile)
			if err != nil {
				t.Fatal(err)
			}
			store, err := contractstate.CreateLocalPersistence(t.TempDir(), layout, overlayLocalLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			port := &nativeBarrierPort{PersistencePort: store, published: make(chan struct{})}
			port.ready.Add(2)
			finished := make(chan error, 2)
			for range 2 {
				go func() {
					_, err := ExecuteOverlayPersistent(context.Background(), port, executable, overlayPersistentLimits())
					finished <- err
				}()
			}
			successes, conflicts := 0, 0
			for range 2 {
				err := <-finished
				if err == nil {
					successes++
				} else if errors.Is(err, contractstate.ErrConflict) {
					conflicts++
				} else {
					t.Error(err)
				}
			}
			snapshot, err := store.Snapshot(layout.Hash())
			if err != nil || successes != 1 || conflicts != 1 || port.publishes.Load() != 2 || snapshot.Version != 1 || len(snapshot.Entries) != 7 {
				t.Fatalf("writers success/conflict/CAS %d/%d/%d, state %+v/%v", successes, conflicts, port.publishes.Load(), snapshot, err)
			}
		})
	}
}

type overlayObservedPort struct {
	contractstate.PersistencePort
	snapshots, publishes        int
	afterSnapshot, afterPublish func()
}

func (port *overlayObservedPort) Snapshot(hash contractstate.LayoutHash) (contractstate.PersistentSnapshot, error) {
	port.snapshots++
	snapshot, err := port.PersistencePort.Snapshot(hash)
	if err == nil && port.afterSnapshot != nil {
		port.afterSnapshot()
	}
	return snapshot, err
}

func (port *overlayObservedPort) CompareAndPublish(hash contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
	port.publishes++
	receipt, err := port.PersistencePort.CompareAndPublish(hash, version, writes)
	if err == nil && port.afterPublish != nil {
		port.afterPublish()
	}
	return receipt, err
}

func TestOverlayPersistenceNativeFailureAndCancellation(t *testing.T) {
	_, prepared, _, layout := overlayWireFixture(t, contractstatemir.ExitCommit)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			executable, err := BuildOverlay(prepared, profile)
			if err != nil {
				t.Fatal(err)
			}
			directory := t.TempDir()
			store, err := contractstate.CreateLocalPersistence(directory, layout, overlayLocalLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			before := persistentStateBytes(t, directory)
			for _, policy := range []struct {
				name   string
				limits contractstate.Limits
				cause  error
			}{
				{"count", contractstate.Limits{MaxWrites: 13, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64}, contractstate.ErrWriteLimit},
				{"value", contractstate.Limits{MaxWrites: 16, MaxWriteBytes: 4096, MaxValueBytes: 31, MaxMapKeyBytes: 64}, contractstate.ErrValueTooLarge},
				{"bytes", contractstate.Limits{MaxWrites: 16, MaxWriteBytes: 32, MaxValueBytes: 64, MaxMapKeyBytes: 64}, contractstate.ErrWriteBytesLimit},
				{"key", contractstate.Limits{MaxWrites: 16, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 31}, contractstate.ErrMapKeyTooLarge},
			} {
				t.Run(policy.name, func(t *testing.T) {
					port := &overlayObservedPort{PersistencePort: store}
					if _, err := ExecuteOverlayPersistent(context.Background(), port, executable, policy.limits); !errors.Is(err, policy.cause) || port.snapshots != 1 || port.publishes != 0 {
						t.Fatal("native policy reached publication", err, port.snapshots, port.publishes)
					}
				})
			}
			cause := errors.New("multi-key backend refused publication")
			failed := &failingNativePort{PersistencePort: store, cause: cause}
			if _, err := ExecuteOverlayPersistent(context.Background(), failed, executable, overlayPersistentLimits()); !errors.Is(err, cause) || failed.publishes != 1 {
				t.Fatal("backend error retried or lost cause", err, failed.publishes)
			}
			limits := overlayLocalLimits()
			limits.MaxValueBytes = 31
			limited, err := contractstate.OpenLocalPersistence(directory, layout, limits)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ExecuteOverlayPersistent(context.Background(), limited, executable, overlayPersistentLimits()); !errors.Is(err, contractstate.ErrValueTooLarge) {
				t.Fatal("local capacity admitted", err)
			}
			if err := limited.Close(); err != nil {
				t.Fatal(err)
			}
			broken := executable
			broken.image.Bytes = []byte("not an executable image")
			if _, err := ExecuteOverlayPersistent(context.Background(), store, broken, overlayPersistentLimits()); err == nil {
				t.Fatal("invalid native image published")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			observed := &overlayObservedPort{PersistencePort: store, afterSnapshot: cancel}
			if _, err := ExecuteOverlayPersistent(ctx, observed, executable, overlayPersistentLimits()); !errors.Is(err, context.Canceled) || observed.snapshots != 1 || observed.publishes != 0 {
				t.Fatal("snapshot cancellation published", err)
			}
			if !bytes.Equal(before, persistentStateBytes(t, directory)) {
				t.Fatal("failed/canceled attempt changed durable bytes")
			}
			ctx, cancelPublished := context.WithCancel(context.Background())
			defer cancelPublished()
			published := &overlayObservedPort{PersistencePort: store, afterPublish: cancelPublished}
			out, err := ExecuteOverlayPersistent(ctx, published, executable, overlayPersistentLimits())
			if err != nil || !errors.Is(ctx.Err(), context.Canceled) || out.Receipt.Version != 1 || out.Receipt.Writes != 14 || published.snapshots != 1 || published.publishes != 1 {
				t.Fatal("acknowledged publication became cancellation", out, err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := contractstate.OpenLocalPersistence(directory, layout, overlayLocalLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			snapshot, err := reopened.Snapshot(layout.Hash())
			if err != nil || snapshot.Version != 1 || len(snapshot.Entries) != 7 {
				t.Fatal("canceled acknowledgement lost durable state", snapshot, err)
			}
		})
	}
}

func TestOverlayPersistenceNativeReadOnly(t *testing.T) {
	handoff, layout := overlayReadFixture(t)
	prepared, err := PrepareOverlayTransport(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	value := bytes.Repeat([]byte{0, 255, 26, 13}, 16385)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			executable, err := BuildOverlay(prepared, profile)
			if err != nil {
				t.Fatal(err)
			}
			directory := t.TempDir()
			store, err := contractstate.CreateLocalPersistence(directory, layout, overlayLocalLimits())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = store.Close() })
			if _, err := store.CompareAndPublish(layout.Hash(), 0, []contractstate.PersistentWrite{{Key: layout.Fields()[0].Prefix, Value: value}}); err != nil {
				t.Fatal(err)
			}
			limits := contractstate.Limits{MaxWrites: 1, MaxWriteBytes: 1, MaxValueBytes: 1, MaxMapKeyBytes: 1}
			before := persistentStateBytes(t, directory)
			for range 2 {
				port := &overlayObservedPort{PersistencePort: store}
				out, err := ExecuteOverlayPersistent(context.Background(), port, executable, limits)
				if err != nil || out.Receipt.Version != 1 || out.Receipt.Writes != 0 || out.Receipt.WriteBytes != 0 || len(out.Reads) != 1 || !out.Reads[0].Found || !bytes.Equal(out.Reads[0].Value.Encoded, value) || port.snapshots != 1 || port.publishes != 1 {
					t.Fatal("large read-only commit failed", out.Receipt, err)
				}
				if !bytes.Equal(before, persistentStateBytes(t, directory)) {
					t.Fatal("read-only commit changed durable bytes")
				}
				if err := store.Close(); err != nil {
					t.Fatal(err)
				}
				store, err = contractstate.OpenLocalPersistence(directory, layout, overlayLocalLimits())
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
