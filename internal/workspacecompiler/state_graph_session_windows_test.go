package workspacecompiler

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/contractstatenative"
)

type graphCacheObservedPort struct {
	contractstate.PersistencePort
	snapshots, publishes int
}

func (port *graphCacheObservedPort) Snapshot(layout contractstate.LayoutHash) (contractstate.PersistentSnapshot, error) {
	port.snapshots++
	return port.PersistencePort.Snapshot(layout)
}

func (port *graphCacheObservedPort) CompareAndPublish(layout contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
	port.publishes++
	return port.PersistencePort.CompareAndPublish(layout, version, writes)
}

func sameGraphCacheOutcome(got, want contractstatemir.Outcome) bool {
	if got.Exit != want.Exit || got.Cost != want.Cost || got.Receipt != want.Receipt || len(got.Reads) != len(want.Reads) {
		return false
	}
	for i, read := range got.Reads {
		other := want.Reads[i]
		if read.Operation != other.Operation || read.Found != other.Found || read.Value.TypeID != other.Value.TypeID || !bytes.Equal(read.Value.Encoded, other.Value.Encoded) {
			return false
		}
	}
	return true
}

func TestStateGraphSessionNative(t *testing.T) {
	for _, fixture := range []string{"transaction.fy", "helpers/transaction.fy", "borrows/transaction.fy", "dynamic/transaction.fy"} {
		for _, storage := range []string{StorageMemory, StorageDisk} {
			t.Run(fixture+"/"+storage, func(t *testing.T) {
				root, _ := stateGraphCacheWorkspace(t, fixture)
				configuration := graphCacheConfiguration(t, storage)
				session, err := NewExperimentalStateGraphSession(configuration)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = session.Close() })
				workspace := loadCacheWorkspace(t, root)
				if cold, err := session.Check(workspace); err != nil || !cold.Compiler.Verified || !cold.Cache.Lowered {
					t.Fatal("cold state cache", err)
				}
				if storage == StorageDisk {
					if err := session.Close(); err != nil {
						t.Fatal(err)
					}
					session, err = NewExperimentalStateGraphSession(configuration)
					if err != nil {
						t.Fatal(err)
					}
				}
				warm, err := session.Check(loadCacheWorkspace(t, root))
				if err != nil || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Lowered || warm.Cache.Hits != 1 {
					t.Fatalf("native state requires a restored hit %+v: %v", warm.Cache, err)
				}
				prepared := currentGraphTransport(t, workspace, warm.Compiler)
				program := warm.Compiler.StateTransactions[0].Program
				builder, err := contractstate.NewLayoutBuilder(program.Layout.ContractUID)
				if err != nil {
					t.Fatal(err)
				}
				for _, field := range program.Layout.Fields {
					if field.Kind == contractstate.FieldCell {
						_, err = contractstate.DeclareEncodedCell(builder, field.Name, field.ValueTypeID)
					} else {
						_, err = contractstate.DeclareEncodedMap(builder, field.Name, field.KeyTypeID, field.ValueTypeID)
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				layout, err := builder.Build()
				if err != nil || layout.Hash() != program.Layout.Hash {
					t.Fatal(err)
				}
				for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
					t.Run(profile.Name, func(t *testing.T) {
						executable, err := contractstatenative.BuildGraph(prepared, profile)
						if err != nil {
							t.Fatal(err)
						}
						localLimits := contractstate.LocalPersistenceLimits{MaxStateBytes: 1 << 20, MaxEntries: 64, MaxValueBytes: 64 << 10}
						directory := t.TempDir()
						store, err := contractstate.CreateLocalPersistence(directory, layout, localLimits)
						if err != nil {
							t.Fatal(err)
						}
						t.Cleanup(func() { _ = store.Close() })
						reference, err := contractstate.NewReferencePersistence(layout, nil)
						if err != nil {
							t.Fatal(err)
						}
						limits := contractstate.Limits{MaxWrites: 16, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64}
						for range 2 {
							observed := &graphCacheObservedPort{PersistencePort: store}
							got, err := contractstatenative.ExecuteGraphPersistent(context.Background(), observed, executable, limits)
							want, wantErr := contractstatemir.ExecutePersistent(reference, program, limits)
							if err != nil || wantErr != nil || !sameGraphCacheOutcome(got, want) || got.Exit != contractstatemir.ExitCommit || observed.snapshots != 1 || observed.publishes != 1 {
								t.Fatalf("restored native/Go outcome or replay mismatch: %v / %v; %+v / %+v", err, wantErr, got, want)
							}
							if err := store.Close(); err != nil {
								t.Fatal(err)
							}
							store, err = contractstate.OpenLocalPersistence(directory, layout, localLimits)
							if err != nil {
								t.Fatal(err)
							}
							snapshot, err := store.Snapshot(layout.Hash())
							expected, wantErr := reference.Snapshot(layout.Hash())
							if err != nil || wantErr != nil || !reflect.DeepEqual(snapshot, expected) {
								t.Fatal("restored graph state did not reopen identically", err, wantErr)
							}
						}
					})
				}
			})
		}
	}
}
