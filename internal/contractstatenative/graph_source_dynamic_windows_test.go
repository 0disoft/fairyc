package contractstatenative

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/target"
)

func dynamicNativeSource(codec, body string) string {
	return fmt.Sprintf(`module transaction; @contract_uid("test.native-dynamic-source")
state {cell %s value;cell %s target;map %s %s values;}
@effects(storage_read,storage_write) fn void transaction() {%s}`, codec, codec, codec, codec, body)
}

// The native process supplies its own response. The independent Go execution
// only checks the outcome and reopened state; it never constructs native output.
func checkDynamicSourceNative(t *testing.T, h Handoff, profile cdriver.Profile, limits contractstate.Limits, attempts int, valid bool, check func(contractstatemir.Outcome)) {
	t.Helper()
	prepared, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	executable, err := BuildGraph(prepared, profile)
	if err != nil {
		t.Fatal(err)
	}
	layout := graphOracleLayout(t, h.program)
	directory := t.TempDir()
	localLimits := overlayLocalLimits()
	localLimits.MaxValueBytes = 64 << 10
	store, err := contractstate.CreateLocalPersistence(directory, layout, localLimits)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	reference, err := contractstate.NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Preserve an unmentioned physical map entry through commit and rollback.
	var extra contractstate.Key
	extra[0], extra[len(extra)-1] = 0xff, 0x79
	for _, port := range []contractstate.PersistencePort{store, reference} {
		if _, err := port.CompareAndPublish(layout.Hash(), 0, []contractstate.PersistentWrite{{Key: extra, Value: []byte{0, 255, 26}}}); err != nil {
			t.Fatal(err)
		}
	}
	for range attempts {
		before := persistentStateBytes(t, directory)
		observed := &overlayObservedPort{PersistencePort: store}
		got, trap, gotErr := runGraphPersistentNative(context.Background(), observed, executable, limits)
		want, wantTrap, wantErr := persistentOracle(reference, h.program, limits)
		if valid {
			if gotErr != nil || wantErr != nil || trap != wantTrap || !sameOverlayOutcome(got, want) {
				t.Fatalf("dynamic native/Go mismatch: errors %v / %v; traps %q / %q; costs %d / %d", gotErr, wantErr, trap, wantTrap, got.Cost, want.Cost)
			}
			if check != nil {
				check(got)
			}
		} else if gotErr == nil || wantErr == nil || trap != wantTrap || got.Exit != "" {
			t.Fatalf("invalid dynamic attempt: errors %v / %v, traps %q / %q, exit %q", gotErr, wantErr, trap, wantTrap, got.Exit)
		}
		publishes := 0
		if valid && got.Exit == contractstatemir.ExitCommit {
			publishes = 1
		}
		if observed.snapshots != 1 || observed.publishes != publishes {
			t.Fatal("dynamic attempt replayed snapshot/CAS", observed.snapshots, observed.publishes)
		}
		if publishes == 0 && !bytes.Equal(before, persistentStateBytes(t, directory)) {
			t.Fatal("dynamic rollback changed durable bytes")
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
		store, err = contractstate.OpenLocalPersistence(directory, layout, localLimits)
		if err != nil {
			t.Fatal(err)
		}
		gotState, err := store.Snapshot(layout.Hash())
		wantState, wantErr := reference.Snapshot(layout.Hash())
		if err != nil || wantErr != nil || !reflect.DeepEqual(gotState, wantState) {
			t.Fatal("dynamic reopened state differs from Go", err, wantErr)
		}
		found := false
		for _, entry := range gotState.Entries {
			if entry.Key == extra {
				found = bytes.Equal(entry.Value, []byte{0, 255, 26})
			}
		}
		if !found {
			t.Fatal("dynamic publication lost the unmentioned entry")
		}
	}
}

func TestGraphSourceNativeDynamic(t *testing.T) {
	data, err := os.ReadFile("../../examples/contract_state_graph/dynamic/transaction.fy")
	if err != nil {
		t.Fatal(err)
	}
	for _, exit := range []string{"commit", "reject", "trap"} {
		text := string(data)
		if exit == "reject" {
			text = strings.Replace(text, "storage::commit()", "storage::reject()", 1)
		} else if exit == "trap" {
			text = strings.Replace(text, "storage::commit()", `storage::trap("overflow")`, 1)
		}
		h := compileNativeGraphSource(t, text)
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(exit+"/"+profile.Name, func(t *testing.T) {
				checkDynamicSourceNative(t, h, profile, overlayPersistentLimits(), 2, true, func(out contractstatemir.Outcome) {
					if exit == "commit" && (out.Cost != 25 || len(out.Reads) != 7 || out.Receipt.Writes != 6 || !bytes.Equal(out.Reads[5].Value.Encoded, []byte{'A', 'B', 26}) || string(out.Reads[6].Value.Encoded) != "나") {
						t.Fatal("authored dynamic source lost independent cost/read/write/byte expectations")
					}
				})
			})
		}
	}
}

func TestGraphSourceNativeDynamicPresence(t *testing.T) {
	for _, trial := range []struct {
		codec, literal string
		want           []byte
	}{
		{"bytes", `b""`, []byte{}}, {"bytes", `b"\x00\xff\r\n\x1a"`, []byte{0, 255, 13, 10, 26}},
		{"text", `""`, []byte{}}, {"text", `"가\0나"`, []byte("가\x00나")},
	} {
		body := fmt.Sprintf(`storage::read("value");storage::write("value",%s);let %s value=storage::read("value");storage::write("values",value,value);storage::read("values",value);storage::commit();`, trial.literal, trial.codec)
		h := compileNativeGraphSource(t, dynamicNativeSource(trial.codec, body))
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(trial.codec+"/"+trial.literal+"/"+profile.Name, func(t *testing.T) {
				checkDynamicSourceNative(t, h, profile, overlayPersistentLimits(), 1, true, func(out contractstatemir.Outcome) {
					if len(out.Reads) != 3 || out.Reads[0].Found || len(out.Reads[0].Value.Encoded) != 0 || !out.Reads[1].Found || !out.Reads[2].Found || !bytes.Equal(out.Reads[1].Value.Encoded, trial.want) || !bytes.Equal(out.Reads[2].Value.Encoded, trial.want) {
						t.Fatal("dynamic absent/present value or binary key collapsed")
					}
				})
			})
		}
	}
}

func TestGraphSourceNativeDynamicTextViews(t *testing.T) {
	for _, trial := range []struct {
		name           string
		offset, length int
		valid          bool
		want           string
	}{
		{"scalar", 3, 3, true, "나"}, {"whole", 0, 9, true, "가나다"},
		{"split-scalar", 1, 1, false, ""}, {"trailing-fragment", 3, 2, false, ""},
	} {
		body := fmt.Sprintf(`storage::write("value","가나다");let b=storage::borrow("value");let v=storage::view(b,%d,%d);let text saved=storage::copy(v);storage::write("target",saved);storage::read("target");storage::commit();`, trial.offset, trial.length)
		h := compileNativeGraphSource(t, dynamicNativeSource("text", body))
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(trial.name+"/"+profile.Name, func(t *testing.T) {
				checkDynamicSourceNative(t, h, profile, overlayPersistentLimits(), 1, trial.valid, func(out contractstatemir.Outcome) {
					if len(out.Reads) != 2 || string(out.Reads[1].Value.Encoded) != trial.want {
						t.Fatal("text view did not preserve exact UTF-8 bytes")
					}
				})
			})
		}
	}
	text := dynamicNativeSource("text", `storage::write("values","가","one");storage::write("values","가","two");storage::read("values","가");storage::read("values","가");storage::commit();`)
	h := compileNativeGraphSource(t, text)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run("no-normalization/"+profile.Name, func(t *testing.T) {
			checkDynamicSourceNative(t, h, profile, overlayPersistentLimits(), 1, true, func(out contractstatemir.Outcome) {
				if len(out.Reads) != 2 || string(out.Reads[0].Value.Encoded) != "one" || string(out.Reads[1].Value.Encoded) != "two" {
					t.Fatal("distinct canonical UTF-8 keys were normalized")
				}
			})
		})
	}
}

func TestGraphSourceNativeDynamicLimits(t *testing.T) {
	for _, codec := range []string{"bytes", "text"} {
		for _, key := range []bool{false, true} {
			bound := 64 << 10
			if key {
				bound = 1024
			}
			for _, n := range []int{bound - 1, bound, bound + 1} {
				value := strings.Repeat("x", n)
				if codec == "text" {
					value = strings.Repeat("가", n/3) + strings.Repeat("x", n%3)
				}
				literal, small := strconv.Quote(value), `"v"`
				if codec == "bytes" {
					literal, small = "b"+literal, "b"+small
				}
				body := fmt.Sprintf(`storage::write("value",%s);storage::read("value");storage::commit();`, literal)
				if key {
					body = fmt.Sprintf(`storage::write("values",%s,%s);storage::read("values",%s);storage::commit();`, literal, small, literal)
				}
				h := compileNativeGraphSource(t, dynamicNativeSource(codec, body))
				limits := contractstate.Limits{MaxWrites: 1024, MaxWriteBytes: 1 << 20, MaxValueBytes: 64 << 10, MaxMapKeyBytes: 1024}
				for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
					t.Run(fmt.Sprintf("%s/key=%v/%d/%s", codec, key, n, profile.Name), func(t *testing.T) {
						checkDynamicSourceNative(t, h, profile, limits, 1, n <= bound, func(out contractstatemir.Outcome) {
							want := []byte(value)
							if key {
								want = []byte("v")
							}
							if len(out.Reads) != 1 || !out.Reads[0].Found || !bytes.Equal(out.Reads[0].Value.Encoded, want) {
								t.Fatal("boundary value changed bytes or presence")
							}
						})
					})
				}
			}
		}
	}
}
