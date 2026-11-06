package contractstatenative

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

func persistentStoreLayout(t *testing.T, handoff Handoff) contractstate.Layout {
	t.Helper()
	builder, err := contractstate.NewLayoutBuilder(handoff.program.Layout.ContractUID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = contractstate.DeclareEncodedCell(builder, handoff.program.Layout.Fields[0].Name, contractstate.U64.TypeID())
	if err != nil {
		t.Fatal(err)
	}
	layout, err := builder.Build()
	if err != nil || layout.Hash() != handoff.program.Layout.Hash {
		t.Fatal("layout differs", err)
	}
	return layout
}

func persistentLocalLimits() contractstate.LocalPersistenceLimits {
	return contractstate.LocalPersistenceLimits{MaxStateBytes: 1 << 20, MaxEntries: 16, MaxValueBytes: 64}
}

func runPersistentNative(ctx context.Context, store contractstate.PersistencePort, executable PersistentExecutable, limits contractstate.Limits) (out contractstatemir.Outcome, category string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if trap, ok := recovered.(contractstatemir.Trap); ok {
				category = trap.Category
			} else {
				panic(recovered)
			}
		}
	}()
	out, err = ExecutePersistent(ctx, store, executable, limits)
	return
}

func persistentStateBytes(t *testing.T, directory string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(directory, "state-v1.bin"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPersistentExecutionNativeLocalStore(t *testing.T) {
	for _, name := range []string{"commit", "reject", "trap", "delete-commit", "delete-reject", "delete-trap", "read-only", "empty"} {
		t.Run(name, func(t *testing.T) {
			exit := contractstatemir.ExitCommit
			if name == "reject" || name == "delete-reject" {
				exit = contractstatemir.ExitReject
			}
			if name == "trap" || name == "delete-trap" {
				exit = contractstatemir.ExitTrap
			}
			handoff, prepared, initial := persistentFixture(t, exit, strings.HasPrefix(name, "delete-"))
			if name == "read-only" || name == "empty" {
				input, coordinate := nativeStateInput(t, exit)
				input.StateTransactions[0].Accesses = nil
				if name == "read-only" {
					input.StateTransactions[0].Accesses = []hir.StateAccess{{Kind: hir.StateAccessCellRead, Field: "value"}}
				}
				var err error
				handoff, err = Prepare(input, 1, 1, coordinate)
				if err != nil {
					t.Fatal(err)
				}
				prepared, err = EmitPersistent(handoff, target.X8664WindowsMSVC)
				if err != nil {
					t.Fatal(err)
				}
			}
			layout := persistentStoreLayout(t, handoff)
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					executable, err := BuildPersistent(prepared, profile)
					if err != nil {
						t.Fatal(err)
					}
					for _, present := range []bool{false, true} {
						t.Run(map[bool]string{false: "absent", true: "present"}[present], func(t *testing.T) {
							directory := t.TempDir()
							store, err := contractstate.CreateLocalPersistence(directory, layout, persistentLocalLimits())
							if err != nil {
								t.Fatal(err)
							}
							t.Cleanup(func() { _ = store.Close() })
							reference, err := contractstate.NewReferencePersistence(layout, nil)
							if err != nil {
								t.Fatal(err)
							}
							if present {
								seed := []contractstate.PersistentWrite{{Key: prepared.key, Value: initial.Entries[0].Value}}
								if _, err := store.CompareAndPublish(layout.Hash(), 0, seed); err != nil {
									t.Fatal(err)
								}
								if _, err := reference.CompareAndPublish(layout.Hash(), 0, seed); err != nil {
									t.Fatal(err)
								}
							}
							before := persistentStateBytes(t, directory)
							got, category, err := runPersistentNative(context.Background(), store, executable, persistentLimits())
							want, wantCategory, wantErr := persistentOracle(reference, handoff.program, persistentLimits())
							if err != nil || wantErr != nil || category != wantCategory || !reflect.DeepEqual(got, want) {
								t.Fatalf("native %+v/%q/%v; Go %+v/%q/%v", got, category, err, want, wantCategory, wantErr)
							}
							if err := store.Close(); err != nil {
								t.Fatal(err)
							}
							reopened, err := contractstate.OpenLocalPersistence(directory, layout, persistentLocalLimits())
							if err != nil {
								t.Fatal(err)
							}
							defer reopened.Close()
							actual, err := reopened.Snapshot(layout.Hash())
							if err != nil {
								t.Fatal(err)
							}
							expected, err := reference.Snapshot(layout.Hash())
							if err != nil || !reflect.DeepEqual(actual, expected) {
								t.Fatal("reopened native state differs from Go", actual, expected, err)
							}
							if exit != contractstatemir.ExitCommit || !prepared.writes {
								if !bytes.Equal(before, persistentStateBytes(t, directory)) {
									t.Fatal("nonmutation changed durable bytes")
								}
							}
							// Repeat against the reopened store, including no-op version preservation.
							got, category, err = runPersistentNative(context.Background(), reopened, executable, persistentLimits())
							want, wantCategory, wantErr = persistentOracle(reference, handoff.program, persistentLimits())
							if err != nil || wantErr != nil || category != wantCategory || !reflect.DeepEqual(got, want) {
								t.Fatalf("reopened native %+v/%q/%v; Go %+v/%q/%v", got, category, err, want, wantCategory, wantErr)
							}
						})
					}
				})
			}
		})
	}
}

type nativeBarrierPort struct {
	contractstate.PersistencePort
	ready     sync.WaitGroup
	published chan struct{}
	publishes atomic.Int32
}

func (port *nativeBarrierPort) Snapshot(hash contractstate.LayoutHash) (contractstate.PersistentSnapshot, error) {
	snapshot, err := port.PersistencePort.Snapshot(hash)
	port.ready.Done()
	port.ready.Wait()
	return snapshot, err
}

func (port *nativeBarrierPort) CompareAndPublish(hash contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
	// Scheduling only: two stale native attempts reach the real OS-locked CAS in
	// a deterministic order, so the loser observes conflict rather than lock busy.
	if port.publishes.Add(1) == 1 {
		defer close(port.published)
	} else {
		<-port.published
	}
	return port.PersistencePort.CompareAndPublish(hash, version, writes)
}

func TestPersistentExecutionNativeCompetingWriters(t *testing.T) {
	handoff, prepared, _ := persistentFixture(t, contractstatemir.ExitCommit, false)
	layout := persistentStoreLayout(t, handoff)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			executable, err := BuildPersistent(prepared, profile)
			if err != nil {
				t.Fatal(err)
			}
			store, err := contractstate.CreateLocalPersistence(t.TempDir(), layout, persistentLocalLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			port := &nativeBarrierPort{PersistencePort: store, published: make(chan struct{})}
			port.ready.Add(2)
			finished := make(chan error, 2)
			for i := 0; i < 2; i++ {
				go func() {
					_, err := ExecutePersistent(context.Background(), port, executable, persistentLimits())
					finished <- err
				}()
			}
			successes, conflicts := 0, 0
			for i := 0; i < 2; i++ {
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
			if err != nil || successes != 1 || conflicts != 1 || port.publishes.Load() != 2 || snapshot.Version != 1 || len(snapshot.Entries) != 1 || snapshot.Entries[0].Value[7] != 42 {
				t.Fatalf("writers success/conflict/publishes %d/%d/%d state %+v %v", successes, conflicts, port.publishes.Load(), snapshot, err)
			}
		})
	}
}

type failingNativePort struct {
	contractstate.PersistencePort
	cause     error
	publishes int
}

func (port *failingNativePort) CompareAndPublish(contractstate.LayoutHash, uint64, []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
	port.publishes++
	return contractstate.PersistentReceipt{}, port.cause
}

func TestPersistentExecutionNativePublicationFailure(t *testing.T) {
	handoff, prepared, _ := persistentFixture(t, contractstatemir.ExitCommit, false)
	layout := persistentStoreLayout(t, handoff)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			executable, err := BuildPersistent(prepared, profile)
			if err != nil {
				t.Fatal(err)
			}
			directory := t.TempDir()
			store, err := contractstate.CreateLocalPersistence(directory, layout, persistentLocalLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			before := persistentStateBytes(t, directory)
			cause := errors.New("native backend rejected before publication")
			port := &failingNativePort{PersistencePort: store, cause: cause}
			if _, err := ExecutePersistent(context.Background(), port, executable, persistentLimits()); !errors.Is(err, cause) {
				t.Fatal("lost backend cause", err)
			}
			if port.publishes != 1 || !bytes.Equal(before, persistentStateBytes(t, directory)) {
				t.Fatal("publication failure changed state or retried")
			}
			limited, err := contractstate.OpenLocalPersistence(directory, layout, contractstate.LocalPersistenceLimits{MaxStateBytes: 1 << 20, MaxEntries: 16, MaxValueBytes: 7})
			if err != nil {
				t.Fatal(err)
			}
			defer limited.Close()
			if _, err := ExecutePersistent(context.Background(), limited, executable, persistentLimits()); !errors.Is(err, contractstate.ErrValueTooLarge) {
				t.Fatal("local capacity failure", err)
			}
			// Test-only corruption forces a real materialization/launch failure.
			broken := executable
			broken.image.Bytes = []byte("not a Windows executable")
			if _, err := ExecutePersistent(context.Background(), store, broken, persistentLimits()); err == nil {
				t.Fatal("invalid process image admitted")
			}
			if !bytes.Equal(before, persistentStateBytes(t, directory)) {
				t.Fatal("native process or local capacity failure changed state")
			}
		})
	}
}
