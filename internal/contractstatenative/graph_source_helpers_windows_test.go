package contractstatenative

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/target"
)

func TestGraphSourceNativeHelpers(t *testing.T) {
	data, err := os.ReadFile("../../examples/contract_state_graph/helpers/transaction.fy")
	if err != nil {
		t.Fatal(err)
	}
	for _, trial := range []struct {
		name         string
		reject, trap bool
		cost         uint64
	}{
		{"commit", false, false, 12}, {"helper-reject", true, false, 5}, {"helper-trap", true, true, 5},
	} {
		text := string(data)
		if trial.trap {
			text = strings.Replace(text, "storage::reject()", `storage::trap("overflow")`, 1)
		}
		h := compileNativeGraphSource(t, text)
		p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
		if err != nil {
			t.Fatal(err)
		}
		layout := graphOracleLayout(t, h.program)
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(trial.name+"/"+profile.Name, func(t *testing.T) {
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
				seedValue, _ := contractstate.Bool.Encode(trial.reject)
				seed := []contractstate.PersistentWrite{{Key: layout.Fields()[0].Prefix, Value: seedValue}}
				for _, port := range []contractstate.PersistencePort{store, reference} {
					if _, err := port.CompareAndPublish(layout.Hash(), 0, seed); err != nil {
						t.Fatal(err)
					}
				}
				for range 2 {
					observed := &overlayObservedPort{PersistencePort: store}
					got, trap, err := runGraphPersistentNative(context.Background(), observed, executable, overlayPersistentLimits())
					want, wantTrap, wantErr := persistentOracle(reference, h.program, overlayPersistentLimits())
					if err != nil || wantErr != nil || trap != wantTrap || !sameOverlayOutcome(got, want) {
						t.Fatalf("helper native %+v/%q/%v; Go %+v/%q/%v", got, trap, err, want, wantTrap, wantErr)
					}
					if !trial.trap && got.Cost != trial.cost {
						t.Fatal("helper source cost", got.Cost, trial.cost)
					}
					if observed.snapshots != 1 || observed.publishes != map[bool]int{false: 1, true: 0}[trial.reject] {
						t.Fatal("helper replayed publication", observed.snapshots, observed.publishes)
					}
					if err := store.Close(); err != nil {
						t.Fatal(err)
					}
					store, err = contractstate.OpenLocalPersistence(directory, layout, overlayLocalLimits())
					if err != nil {
						t.Fatal(err)
					}
					gotState, err := store.Snapshot(layout.Hash())
					wantState, wantErr := reference.Snapshot(layout.Hash())
					if err != nil || wantErr != nil || !reflect.DeepEqual(gotState, wantState) {
						t.Fatal("helper reopened state mismatch", err, wantErr)
					}
				}
			})
		}
	}
}
