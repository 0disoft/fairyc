package contractstatenative

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPersistentTransportNative(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("native state transport requires Windows amd64")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := cstdlib.ContractStateRuntimeSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"commit", "reject", "trap", "delete-commit", "delete-reject", "delete-trap", "read-only", "empty", "intermediate-write", "reads-around-mutations", "max-reads"} {
		t.Run(name, func(t *testing.T) {
			exit := contractstatemir.ExitCommit
			if name == "reject" || name == "delete-reject" {
				exit = contractstatemir.ExitReject
			}
			if name == "trap" || name == "delete-trap" {
				exit = contractstatemir.ExitTrap
			}
			handoff, artifact, snapshot := persistentFixture(t, exit, len(name) >= 7 && name[:7] == "delete-")
			if name == "read-only" || name == "empty" || name == "intermediate-write" || name == "reads-around-mutations" || name == "max-reads" {
				input, coordinate := nativeStateInput(t, exit)
				read := hir.StateAccess{Kind: hir.StateAccessCellRead, Field: "value"}
				write := input.StateTransactions[0].Accesses[0]
				deleting := hir.StateAccess{Kind: hir.StateAccessCellDelete, Field: "value"}
				switch name {
				case "read-only":
					input.StateTransactions[0].Accesses = []hir.StateAccess{read}
				case "empty":
					input.StateTransactions[0].Accesses = nil
				case "intermediate-write":
					input.StateTransactions[0].Accesses = []hir.StateAccess{write, deleting, read}
				case "reads-around-mutations":
					input.StateTransactions[0].Accesses = []hir.StateAccess{read, write, read, deleting, read, write, read}
				case "max-reads":
					input.StateTransactions[0].Accesses = make([]hir.StateAccess, persistentMaxOperations)
					for i := range input.StateTransactions[0].Accesses {
						input.StateTransactions[0].Accesses[i] = read
					}
				}
				handoff, err = Prepare(input, 1, 1, coordinate)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err = EmitPersistent(handoff, target.X8664WindowsMSVC)
				if err != nil {
					t.Fatal(err)
				}
				snapshot.LayoutHash = artifact.LayoutHash()
				snapshot.Entries[0].Key = handoff.program.Layout.Fields[0].Prefix
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					image, err := cdriver.BuildWindowsHost(toolchain, profile, artifact.Source(), runtimeSource)
					if err != nil {
						t.Fatal(err)
					}
					for _, present := range []bool{false, true} {
						for _, policy := range []struct {
							name       string
							writeBytes uint64
							valueBytes int
						}{
							{"exact-write", 41, 8}, {"before-write", 40, 8}, {"before-delete", 32, 8}, {"exact-delete", 33, 8}, {"before-value", 41, 7}, {"after-boundaries", 42, 9},
						} {
							if name == "max-reads" && policy.name != "exact-write" {
								continue
							}
							t.Run(fmt.Sprintf("%s/present-%t", policy.name, present), func(t *testing.T) {
								selected := snapshot
								if !present {
									selected.Entries = nil
								}
								limits := persistentLimits()
								limits.MaxWriteBytes = policy.writeBytes
								limits.MaxValueBytes = policy.valueBytes
								input, err := artifact.Input(selected, limits)
								if err != nil {
									t.Fatal(err)
								}
								outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(input.Bytes()))
								if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" {
									t.Fatalf("native: %+v %v", outcome, err)
								}
								got, err := artifact.Decode(input, []byte(outcome.Stdout))
								if err != nil {
									t.Fatal(err)
								}
								comparePersistentOracle(t, handoff, selected, limits, got)
							})
						}
					}
					if name == "commit" {
						input, err := artifact.Input(snapshot, persistentLimits())
						if err != nil {
							t.Fatal(err)
						}
						for _, fault := range []string{"short", "trailing", "magic", "layout", "body", "program", "presence", "absent-padding", "write-limit", "value-limit"} {
							packet := input.Bytes()
							switch fault {
							case "short":
								packet = packet[:len(packet)-1]
							case "trailing":
								packet = append(packet, 0)
							case "magic":
								packet[0] ^= 1
							case "layout":
								packet[8] ^= 1
							case "body":
								packet[40] ^= 1
							case "program":
								packet[72] ^= 1
							case "presence":
								packet[128] = 2
							case "absent-padding":
								packet[128] = 0
							case "write-limit":
								binary.BigEndian.PutUint64(packet[112:], 0)
							case "value-limit":
								binary.BigEndian.PutUint64(packet[120:], 0)
							}
							outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(packet))
							if err != nil || outcome.ExitCode != 4 || outcome.Stdout != "" || outcome.Stderr != "" {
								t.Fatalf("malformed %s: %+v %v", fault, outcome, err)
							}
						}
					}
				})
			}
		})
	}
}

func comparePersistentOracle(t *testing.T, handoff Handoff, snapshot contractstate.PersistentSnapshot, limits contractstate.Limits, got PersistentResult) {
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
	if err != nil {
		t.Fatal(err)
	}
	store, err := contractstate.NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Entries) != 0 {
		_, err = store.CompareAndPublish(layout.Hash(), 0, []contractstate.PersistentWrite{{Key: snapshot.Entries[0].Key, Value: slices.Clone(snapshot.Entries[0].Value)}})
		if err != nil {
			t.Fatal(err)
		}
	}
	before, err := store.Snapshot(layout.Hash())
	if err != nil {
		t.Fatal(err)
	}
	want, trap, runErr := persistentOracle(store, handoff.program, limits)
	status := PersistentCommit
	if runErr != nil {
		if got.Status != PersistentWriteBytesLimit && got.Status != PersistentValueLimit {
			t.Fatalf("native accepted oracle error: %+v %v", got, runErr)
		}
		if got.Status == PersistentWriteBytesLimit && !errors.Is(runErr, contractstate.ErrWriteBytesLimit) {
			t.Fatal("different limit error", got, runErr)
		}
		if got.Status == PersistentValueLimit && !errors.Is(runErr, contractstate.ErrValueTooLarge) {
			t.Fatal("different limit error", got, runErr)
		}
		status = got.Status
	} else if trap != "" {
		status = PersistentTrap
	} else if want.Exit == contractstatemir.ExitReject {
		status = PersistentReject
	}
	after, err := store.Snapshot(layout.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != status || got.Version != math.MaxUint64 || got.Present != (len(after.Entries) != 0) || got.Dirty != (after.Version != before.Version) || got.TrapCategory != trap {
		t.Fatalf("native/oracle metadata: %+v want status %d trap %q", got, status, trap)
	}
	if len(after.Entries) != 0 && !bytes.Equal(got.Value, after.Entries[0].Value) {
		t.Fatal("native/oracle committed value differs", got, after)
	}
	if trap != "" {
		control := handoff.program
		control.Exit = contractstatemir.ExitReject
		control.TrapCategory = ""
		var controlTrap string
		want, controlTrap, err = persistentOracle(store, control, limits)
		if err != nil || controlTrap != "" {
			t.Fatal("trap read observer failed", controlTrap, err)
		}
	}
	if runErr == nil && !reflect.DeepEqual(got.Reads, want.Reads) {
		t.Fatalf("native/oracle reads: %+v / %+v", got.Reads, want.Reads)
	}
	if status != PersistentCommit && !reflect.DeepEqual(before, after) {
		t.Fatal("oracle rollback changed store")
	}
}

func persistentOracle(store contractstate.PersistencePort, program contractstatemir.Program, limits contractstate.Limits) (out contractstatemir.Outcome, category string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if trap, ok := recovered.(contractstatemir.Trap); ok {
				category = trap.Category
			} else {
				panic(recovered)
			}
		}
	}()
	out, err = contractstatemir.ExecutePersistent(store, program, limits)
	return
}
