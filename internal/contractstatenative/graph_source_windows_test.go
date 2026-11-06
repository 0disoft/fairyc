package contractstatenative

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

func compileNativeGraphSource(t *testing.T, text string) Handoff {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("transaction.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatal("parse source", err)
	}
	compiled, err := compiler.CheckExperimentalStateGraph(semantic.File{Package: hir.DefaultPackage, Path: "transaction.fy", Module: parsed.AST})
	if err != nil || !compiled.Verified || len(compiled.StateTransactions) != 1 {
		t.Fatalf("compile graph: %v %+v", err, compiled.Semantic)
	}
	_, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	owner := compiled.StateTransactions[0]
	handoff, err := PrepareCompiled(compiled, owner.Package, owner.Function, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	return handoff
}

func TestGraphSourceNativePersistence(t *testing.T) {
	data, err := os.ReadFile("../../examples/contract_state_graph/transaction.fy")
	if err != nil {
		t.Fatal(err)
	}
	for _, exit := range []contractstatemir.ExitKind{contractstatemir.ExitCommit, contractstatemir.ExitReject, contractstatemir.ExitTrap} {
		text := string(data)
		if exit != contractstatemir.ExitCommit {
			replacement := "storage::reject()"
			if exit == contractstatemir.ExitTrap {
				replacement = `storage::trap("overflow")`
			}
			text = strings.Replace(text, "storage::commit()", replacement, 1)
		}
		h := compileNativeGraphSource(t, text)
		p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
		if err != nil {
			t.Fatal(err)
		}
		layout := graphOracleLayout(t, h.program)
		fields := layout.Fields()
		keyBytes, _ := contractstate.U64.Encode(7)
		oldBytes, _ := contractstate.U64.Encode(73)
		mapKey := overlayOperationKey(h.program.Layout.Fields[3], keyBytes)
		otherBytes, _ := contractstate.U64.Encode(8)
		otherKey := overlayOperationKey(h.program.Layout.Fields[3], otherBytes)
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(string(exit)+"/"+profile.Name, func(t *testing.T) {
				executable, err := BuildGraph(p, profile)
				if err != nil {
					t.Fatal(err)
				}
				for _, trial := range []struct {
					name             string
					enabled, present bool
					want             uint64
				}{
					{"disabled", false, true, 0}, {"enabled-missing", true, false, 42}, {"enabled-found", true, true, 73},
				} {
					t.Run(trial.name, func(t *testing.T) {
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
						enabled, _ := contractstate.Bool.Encode(trial.enabled)
						seed := []contractstate.PersistentWrite{
							{Key: fields[0].Prefix, Value: keyBytes}, {Key: fields[1].Prefix, Value: enabled},
							{Key: otherKey, Value: oldBytes},
						}
						if trial.present {
							seed = append(seed, contractstate.PersistentWrite{Key: mapKey, Value: oldBytes})
						}
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
								t.Fatalf("source native %+v/%q/%v; Go %+v/%q/%v", got, trap, err, want, wantTrap, wantErr)
							}
							committed := exit == contractstatemir.ExitCommit
							if observed.snapshots != 1 || observed.publishes != map[bool]int{false: 0, true: 1}[committed] {
								t.Fatal("source attempt replayed snapshot/CAS", observed.snapshots, observed.publishes)
							}
							if committed {
								value, decodeErr := contractstate.U64.Decode(got.Reads[len(got.Reads)-1].Value.Encoded)
								if decodeErr != nil || value != trial.want {
									t.Fatal("wrong source branch value", value, decodeErr)
								}
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
								t.Fatal("source reopen mismatch", err, wantErr)
							}
						}
					})
				}
			})
		}
	}
}

func TestGraphSourceNativeCodecs(t *testing.T) {
	for _, trial := range []struct{ codec, literal string }{
		{"bool", "true"}, {"u32", "4294967295"}, {"u64", "18446744073709551615"},
		{"bytes20", `b"` + strings.Repeat(`\xff`, 20) + `"`}, {"bytes32", `b"` + strings.Repeat(`\x00`, 31) + `\xff"`},
	} {
		t.Run(trial.codec, func(t *testing.T) {
			text := fmt.Sprintf(`module transaction;
@contract_uid("test.native-source-codec")
state { cell %s value; map %s %s values; }
@effects(storage_read, storage_write)
fn void transaction() {
storage::write("value", %s);
let value = storage::read("value");
storage::write("values", value, value);
storage::read("values", value);
storage::delete("values", value);
storage::commit();
}`, trial.codec, trial.codec, trial.codec, trial.literal)
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
						t.Fatalf("codec native %+v/%q/%v; Go %+v/%q/%v", got, trap, err, want, wantTrap, wantErr)
					}
				})
			}
		})
	}
}
