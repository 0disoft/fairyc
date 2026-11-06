package contractstatenative

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/target"
)

func TestGraphSourceNativeBorrows(t *testing.T) {
	data, err := os.ReadFile("../../examples/contract_state_graph/borrows/transaction.fy")
	if err != nil {
		t.Fatal(err)
	}
	for _, exit := range []string{"commit", "reject", "trap"} {
		text := string(data)
		if exit == "reject" {
			text = strings.Replace(text, "storage::commit()", "storage::reject()", 1)
		}
		if exit == "trap" {
			text = strings.Replace(text, "storage::commit()", `storage::trap("overflow")`, 1)
		}
		h := compileNativeGraphSource(t, text)
		p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
		if err != nil {
			t.Fatal(err)
		}
		layout := graphOracleLayout(t, h.program)
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(exit+"/"+profile.Name, func(t *testing.T) {
				executable, err := BuildGraph(p, profile)
				if err != nil {
					t.Fatal(err)
				}
				for _, enabled := range []bool{false, true} {
					t.Run(fmt.Sprint(enabled), func(t *testing.T) {
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
						flag, _ := contractstate.Bool.Encode(enabled)
						for _, port := range []contractstate.PersistencePort{store, reference} {
							if _, err := port.CompareAndPublish(layout.Hash(), 0, []contractstate.PersistentWrite{{Key: layout.Fields()[2].Prefix, Value: flag}}); err != nil {
								t.Fatal(err)
							}
						}
						for range 2 {
							observed := &overlayObservedPort{PersistencePort: store}
							got, trap, err := runGraphPersistentNative(context.Background(), observed, executable, overlayPersistentLimits())
							want, wantTrap, wantErr := persistentOracle(reference, h.program, overlayPersistentLimits())
							if err != nil || wantErr != nil || trap != wantTrap || !sameOverlayOutcome(got, want) {
								t.Fatalf("borrow native %+v/%q/%v; Go %+v/%q/%v", got, trap, err, want, wantTrap, wantErr)
							}
							if observed.snapshots != 1 || observed.publishes != map[bool]int{false: 0, true: 1}[exit == "commit"] {
								t.Fatal("borrow publication replay", observed.snapshots, observed.publishes)
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
								t.Fatal("borrow reopen mismatch", err, wantErr)
							}
							if exit == "commit" {
								counts := map[uint64]int{}
								for _, entry := range gotState.Entries {
									if len(entry.Value) == 8 {
										value, err := contractstate.U64.Decode(entry.Value)
										if err != nil {
											t.Fatal(err)
										}
										counts[value]++
									}
								}
								if counts[42] != 1 || counts[99] != 1 {
									t.Fatal("root copy did not survive helper mutation/branch", counts)
								}
							}
						}
					})
				}
			})
		}
	}
}

func TestGraphSourceNativeMapBorrowCodecs(t *testing.T) {
	for _, trial := range []struct{ codec, key, other, value string }{
		{"bool", "false", "true", "true"}, {"u32", "7", "8", "4294967295"}, {"u64", "7", "8", "18446744073709551615"},
		{"bytes20", `b"` + strings.Repeat(`\x00`, 20) + `"`, `b"` + strings.Repeat(`\xff`, 20) + `"`, `b"` + strings.Repeat(`\xff`, 20) + `"`},
		{"bytes32", `b"` + strings.Repeat(`\x00`, 32) + `"`, `b"` + strings.Repeat(`\xff`, 32) + `"`, `b"` + strings.Repeat(`\xff`, 32) + `"`},
	} {
		t.Run(trial.codec, func(t *testing.T) {
			text := fmt.Sprintf(`module transaction; @contract_uid("test.native-map-borrow")
state { cell %s value; map %s %s values; }
@effects(storage_read,storage_write)
fn void transaction() {
storage::write("values",%s,%s);
let b=storage::borrow("values",%s);
storage::write("values",%s,%s);
let saved=storage::copy(b);
storage::write("value",saved);
storage::read("value");
storage::commit();
}`, trial.codec, trial.codec, trial.codec, trial.key, trial.value, trial.key, trial.other, trial.value)
			h := compileNativeGraphSource(t, text)
			p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
			if err != nil {
				t.Fatal(err)
			}
			layout := graphOracleLayout(t, h.program)
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					executable, err := BuildGraph(p, profile)
					if err != nil {
						t.Fatal(err)
					}
					store, err := contractstate.NewReferencePersistence(layout, nil)
					if err != nil {
						t.Fatal(err)
					}
					reference, err := contractstate.NewReferencePersistence(layout, nil)
					if err != nil {
						t.Fatal(err)
					}
					got, trap, err := runGraphPersistentNative(context.Background(), store, executable, overlayPersistentLimits())
					want, wantTrap, wantErr := persistentOracle(reference, h.program, overlayPersistentLimits())
					if err != nil || wantErr != nil || trap != wantTrap || !sameOverlayOutcome(got, want) {
						t.Fatalf("map borrow native %+v/%q/%v; Go %+v/%q/%v", got, trap, err, want, wantTrap, wantErr)
					}
					if len(got.Reads) != 2 || !bytes.Equal(got.Reads[0].Value.Encoded, got.Reads[1].Value.Encoded) {
						t.Fatal("unrelated map write changed borrowed bytes", got.Reads)
					}
				})
			}
		})
	}
}

func TestGraphSourceNativeBorrowFailures(t *testing.T) {
	for _, trial := range []struct {
		name, body  string
		invalidated bool
	}{
		{"missing", `let b=storage::borrow("value"); let v=storage::copy(b); storage::commit();`, true},
		{"one-past-end", `storage::write("value",42); let b=storage::borrow("value"); let v=storage::view(b,1,8); storage::commit();`, false},
		{"max-offset", `storage::write("value",42); let b=storage::borrow("value"); let v=storage::view(b,4294967295,1); storage::commit();`, false},
		{"noncanonical-copy", `storage::write("value",42); let b=storage::borrow("value"); let v=storage::view(b,1,7); let saved=storage::copy(v); storage::write("value",saved); storage::commit();`, false},
	} {
		text := `module transaction; @contract_uid("test.native-borrow-failure") state {cell u64 value;} @effects(storage_read,storage_write) fn void transaction() {` + trial.body + `}`
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
				before := persistentStateBytes(t, directory)
				observed := &overlayObservedPort{PersistencePort: store}
				got, trap, err := runGraphPersistentNative(context.Background(), observed, executable, overlayPersistentLimits())
				_, wantTrap, wantErr := persistentOracle(reference, h.program, overlayPersistentLimits())
				if err == nil || wantErr == nil || trap != wantTrap || got.Exit != "" || observed.snapshots != 1 || observed.publishes != 0 || !bytes.Equal(before, persistentStateBytes(t, directory)) {
					t.Fatalf("failed borrow published/replayed %+v/%q/%v; Go %q/%v", got, trap, err, wantTrap, wantErr)
				}
				if trial.invalidated && (!errors.Is(err, contractstate.ErrBorrowInvalidated) || !errors.Is(wantErr, contractstate.ErrBorrowInvalidated)) {
					t.Fatal("missing borrow cause lost", err, wantErr)
				}
			})
		}
	}
}
