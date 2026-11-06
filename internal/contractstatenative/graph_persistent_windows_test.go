package contractstatenative

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatefixture"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/target"
)

func runGraphPersistentNative(ctx context.Context, store contractstate.PersistencePort, executable GraphExecutable, limits contractstate.Limits) (out contractstatemir.Outcome, category string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if trap, ok := recovered.(contractstatemir.Trap); ok {
				category = trap.Category
			} else {
				panic(recovered)
			}
		}
	}()
	out, err = ExecuteGraphPersistent(ctx, store, executable, limits)
	return
}

func TestGraphPersistenceNativeLocalStore(t *testing.T) {
	_, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	var cases []contractstatefixture.HIRCase
	for _, version := range []int{0, 1, 2, 3, 4} {
		path := fmt.Sprintf("../../spec/fixtures/contract-state/manifest-v%d.json", version)
		loaded, err := contractstatefixture.LoadHIRCases(path)
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, loaded...)
	}
	cases = append(cases, dynamicGraphCases(t)...)
	empty, _ := nativeStateInput(t, contractstatemir.ExitCommit)
	empty.StateTransactions[0].Accesses = nil
	cases = append(cases, contractstatefixture.HIRCase{Name: "empty-commit", Input: empty, Limits: overlayPersistentLimits()})
	for index, trial := range cases {
		t.Run(fmt.Sprintf("%02d-%s", index, trial.Name), func(t *testing.T) {
			h, err := Prepare(trial.Input, 1, 1, coordinate)
			if err != nil {
				t.Fatal(err)
			}
			layout := graphOracleLayout(t, h.program)
			p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					executable, err := BuildGraph(p, profile)
					if err != nil {
						t.Fatal(err)
					}
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
					// A physical map address absent from this transaction must survive
					// full snapshots, commits, rollbacks and process/store restarts.
					var extra contractstate.Key
					for i := range extra {
						extra[i] = 255
					}
					seed := []contractstate.PersistentWrite{{Key: extra, Value: []byte{255, 0, 13, 10}}}
					for _, port := range []contractstate.PersistencePort{store, reference} {
						if _, err := port.CompareAndPublish(layout.Hash(), 0, seed); err != nil {
							t.Fatal(err)
						}
					}
					for range 2 {
						before := persistentStateBytes(t, directory)
						port := &overlayObservedPort{PersistencePort: store}
						got, trap, gotErr := runGraphPersistentNative(context.Background(), port, executable, trial.Limits)
						want, wantTrap, wantErr := persistentOracle(reference, h.program, trial.Limits)
						if (gotErr == nil) != (wantErr == nil) || trap != wantTrap || !sameOverlayOutcome(got, want) {
							t.Fatalf("native %+v/%q/%v; Go %+v/%q/%v", got, trap, gotErr, want, wantTrap, wantErr)
						}
						for _, cause := range []error{contractstate.ErrWriteLimit, contractstate.ErrWriteBytesLimit, contractstate.ErrValueTooLarge, contractstate.ErrMapKeyTooLarge, contractstate.ErrBorrowInvalidated} {
							if errors.Is(wantErr, cause) && !errors.Is(gotErr, cause) {
								t.Fatal("native error lost Go semantic cause", gotErr, wantErr)
							}
						}
						committed := wantErr == nil && wantTrap == "" && want.Exit == contractstatemir.ExitCommit
						if port.snapshots != 1 || port.publishes != map[bool]int{false: 0, true: 1}[committed] {
							t.Fatal("native attempt skipped or retried snapshot/CAS", port.snapshots, port.publishes)
						}
						if !committed || want.Receipt.Writes == 0 {
							if !bytes.Equal(before, persistentStateBytes(t, directory)) {
								t.Fatal("failed/rejected/trapped/no-op execution changed durable bytes")
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
							t.Fatal("reopened full native state differs from Go", actual, expected, err)
						}
						preserved := false
						for _, e := range actual.Entries {
							if e.Key == extra && bytes.Equal(e.Value, seed[0].Value) {
								preserved = true
							}
						}
						if !preserved {
							t.Fatal("unmentioned physical map entry was lost")
						}
					}
				})
			}
		})
	}
}

type graphAttemptBarrier struct {
	ready                sync.WaitGroup
	snapshot             sync.Mutex
	published            chan struct{}
	snapshots, publishes atomic.Int32
}

type graphCompetingPort struct {
	contractstate.PersistencePort
	barrier *graphAttemptBarrier
}

func (port graphCompetingPort) Snapshot(hash contractstate.LayoutHash) (contractstate.PersistentSnapshot, error) {
	// The local provider deliberately reports busy rather than waiting on a
	// competing file lease. Schedule reads before either writer can publish,
	// so both real snapshots succeed and specifically exercise stale-version CAS.
	port.barrier.snapshot.Lock()
	snapshot, err := port.PersistencePort.Snapshot(hash)
	port.barrier.snapshot.Unlock()
	port.barrier.snapshots.Add(1)
	port.barrier.ready.Done()
	port.barrier.ready.Wait()
	return snapshot, err
}

func (port graphCompetingPort) CompareAndPublish(hash contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
	// Deterministic scheduling only: the two independent opened stores still
	// perform the real OS-locked layout/version comparison and publication.
	if port.barrier.publishes.Add(1) == 1 {
		defer close(port.barrier.published)
	} else {
		<-port.barrier.published
	}
	return port.PersistencePort.CompareAndPublish(hash, version, writes)
}

func TestGraphPersistenceNativeCompetingWriters(t *testing.T) {
	p, snapshot, _, key := graphBranchFixture(t)
	builder, err := contractstate.NewLayoutBuilder("native.state")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contractstate.DeclareEncodedCell(builder, "seed", "u64/be/v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := contractstate.DeclareEncodedMap(builder, "items", "u64/be/v1", "u64/be/v1"); err != nil {
		t.Fatal(err)
	}
	layout, err := builder.Build()
	if err != nil || layout.Hash() != p.entry.layout {
		t.Fatal("independent dynamic layout differs", err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			executable, err := BuildGraph(p, profile)
			if err != nil {
				t.Fatal(err)
			}
			directory := t.TempDir()
			first, err := contractstate.CreateLocalPersistence(directory, layout, overlayLocalLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer first.Close()
			if _, err := first.CompareAndPublish(layout.Hash(), 0, []contractstate.PersistentWrite{{Key: snapshot.Entries[0].Key, Value: snapshot.Entries[0].Value}}); err != nil {
				t.Fatal(err)
			}
			second, err := contractstate.OpenLocalPersistence(directory, layout, overlayLocalLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer second.Close()
			barrier := &graphAttemptBarrier{published: make(chan struct{})}
			barrier.ready.Add(2)
			finished := make(chan error, 2)
			for _, store := range []contractstate.PersistencePort{first, second} {
				go func() {
					_, err := ExecuteGraphPersistent(context.Background(), graphCompetingPort{store, barrier}, executable, overlayPersistentLimits())
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
			actual, err := second.Snapshot(layout.Hash())
			if err != nil || successes != 1 || conflicts != 1 || barrier.snapshots.Load() != 2 || barrier.publishes.Load() != 2 || actual.Version != 2 || len(actual.Entries) != 2 {
				t.Fatal("native writers retried or lost CAS conflict", successes, conflicts, actual, err)
			}
			found := false
			for _, e := range actual.Entries {
				if e.Key == key && bytes.Equal(e.Value, snapshot.Entries[0].Value) {
					found = true
				}
			}
			if !found {
				t.Fatal("dynamic map mutation did not reach real storage")
			}
		})
	}
}
