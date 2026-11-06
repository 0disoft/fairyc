package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
)

func TestNativeGraphCommand(t *testing.T) {
	for _, profile := range []string{"low", "optimized"} {
		t.Run(profile, func(t *testing.T) {
			directory, store := t.TempDir(), t.TempDir()
			path := filepath.Join(directory, "transaction.fy")
			write := func(value uint64, exit string) {
				t.Helper()
				text := fmt.Sprintf(`module transaction;
@contract_uid("test.native-state-command")
state { cell u64 value; map u64 u64 values; }
@effects(storage_read, storage_write)
fn void transaction() {
 storage::write("value", %d);
 let value = storage::read("value");
 if (storage::found(value)) { storage::write("values", value, value); }
 storage::read("values", value);
 %s;
}`, value, exit)
				if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			args := func(action string) []string {
				return []string{action, store, path, "--engine", "native-graph", "--profile", profile}
			}
			call := func(action string, want int) nativeReport {
				t.Helper()
				var out, diagnostics bytes.Buffer
				if code := run(args(action), &out, &diagnostics); code != want {
					t.Fatalf("%s %d: %s", action, code, &diagnostics)
				}
				var result nativeReport
				if err := json.Unmarshal(out.Bytes(), &result); err != nil {
					t.Fatal("report", err, out.String())
				}
				if result.Schema != "fairyc.local-state-run/v2" || result.Engine != "native-graph" || result.Profile != profile || len(result.ProgramHash) != 64 || result.BodyHash == "" {
					t.Fatalf("incomplete identity %+v", result)
				}
				return result
			}
			write(42, "storage::commit()")
			initial := call("init", 0)
			if initial.Snapshot == nil || initial.Snapshot.Version != 0 || len(initial.Snapshot.Entries) != 0 || initial.Outcome != nil {
				t.Fatalf("init ran body: %+v", initial)
			}
			if code := run(args("init"), io.Discard, io.Discard); code != 4 {
				t.Fatal("init replaced existing authority", code)
			}
			if code := run([]string{"execute", store, path}, io.Discard, io.Discard); code != 3 {
				t.Fatal("graph fell back to original Go mode", code)
			}
			first := call("execute", 0)
			if first.ProgramHash != initial.ProgramHash || first.BodyHash != initial.BodyHash || first.Outcome == nil || first.Outcome.Receipt.Version != 1 || first.Outcome.Receipt.Writes != 2 || len(first.Outcome.Reads) != 2 {
				t.Fatalf("native first commit: %+v", first)
			}
			second := call("execute", 0)
			if second.Outcome.Receipt.Version != 2 || second.ProgramHash != first.ProgramHash {
				t.Fatal("native replay count or source identity", second)
			}
			before, err := os.ReadFile(filepath.Join(store, "state-v1.bin"))
			if err != nil {
				t.Fatal(err)
			}
			write(73, "storage::commit()")
			snapshot := call("snapshot", 0)
			if snapshot.Snapshot.Version != 2 || snapshot.ProgramHash == first.ProgramHash || snapshot.BodyHash == first.BodyHash || snapshot.LayoutHash != first.LayoutHash {
				t.Fatal("source edit did not rebind body", snapshot)
			}
			after, err := os.ReadFile(filepath.Join(store, "state-v1.bin"))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("snapshot ran changed source", err)
			}
			changed := call("execute", 0)
			decoded, err := contractstate.U64.Decode(changed.Outcome.Reads[1].Value.Encoded)
			if err != nil || decoded != 73 || changed.Outcome.Receipt.Version != 3 {
				t.Fatalf("fresh source execution %+v %v", changed, err)
			}
			before, err = os.ReadFile(filepath.Join(store, "state-v1.bin"))
			if err != nil {
				t.Fatal(err)
			}
			for _, trial := range []struct {
				exit string
				code int
			}{{"storage::reject()", 6}, {`storage::trap("overflow")`, 8}} {
				write(99, trial.exit)
				result := call("execute", trial.code)
				if trial.code == 8 && result.TrapCategory != "overflow" {
					t.Fatal("lost trap", result)
				}
				after, err := os.ReadFile(filepath.Join(store, "state-v1.bin"))
				if err != nil || !bytes.Equal(before, after) {
					t.Fatal("rollback changed persistent state", err)
				}
			}
			// A fresh report sink failure happens after one actual commit. It
			// preserves that publication and explicitly tells callers to inspect it.
			write(88, "storage::commit()")
			var diagnostics bytes.Buffer
			if code := run(args("execute"), nativeBrokenWriter{}, &diagnostics); code != 4 || !strings.Contains(diagnostics.String(), "may already be published") {
				t.Fatal("ambiguous output failure", code, diagnostics.String())
			}
			if result := call("snapshot", 0); result.Snapshot.Version != 4 {
				t.Fatal("output failure replayed/lost commit", result)
			}
			before, err = os.ReadFile(filepath.Join(store, "state-v1.bin"))
			if err != nil {
				t.Fatal(err)
			}
			changedLayout, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			changedLayout = bytes.Replace(changedLayout, []byte("test.native-state-command"), []byte("test.changed-native-state-command"), 1)
			if err := os.WriteFile(path, changedLayout, 0o600); err != nil {
				t.Fatal(err)
			}
			if code := run(args("snapshot"), io.Discard, io.Discard); code != 7 {
				t.Fatal("layout mismatch status", code)
			}
			after, err = os.ReadFile(filepath.Join(store, "state-v1.bin"))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("layout mismatch changed store", err)
			}
		})
	}
}

type nativeBrokenWriter struct{}

func (nativeBrokenWriter) Write([]byte) (int, error) { return 0, errors.New("test report sink failed") }

func TestNativeGraphSourceRefusalBeforeStore(t *testing.T) {
	directory, store := t.TempDir(), t.TempDir()
	path := filepath.Join(directory, "transaction.fy")
	text := `module transaction;
@contract_uid("test.native-state-command")
state { cell u64 value; }
@effects(storage_read, storage_write)
fn void transaction() { storage::write("value", 18446744073709551616); storage::commit(); }`
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"init", "execute", "snapshot"} {
		if code := run([]string{action, store, path, "--engine", "native-graph", "--profile", "low"}, io.Discard, io.Discard); code != 3 {
			t.Fatal("invalid source status", code)
		}
	}
	entries, err := os.ReadDir(store)
	if err != nil || len(entries) != 0 {
		t.Fatal("refused source touched store", err, entries)
	}
}

func TestNativeGraphHelperBorrowCommand(t *testing.T) {
	for _, trial := range []struct {
		fixture string
		cost    uint64
	}{{"helpers", 12}, {"borrows", 16}} {
		for _, profile := range []string{"low", "optimized"} {
			t.Run(trial.fixture+"/"+profile, func(t *testing.T) {
				directory, store := t.TempDir(), t.TempDir()
				path := filepath.Join(directory, "transaction.fy")
				data, err := os.ReadFile(filepath.Join("..", "..", "..", "examples", "contract_state_graph", trial.fixture, "transaction.fy"))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
				call := func(action string) nativeReport {
					t.Helper()
					var out, diagnostics bytes.Buffer
					if code := run([]string{action, store, path, "--engine", "native-graph", "--profile", profile}, &out, &diagnostics); code != 0 {
						t.Fatalf("%s code %d: %s", action, code, &diagnostics)
					}
					var result nativeReport
					if err := json.Unmarshal(out.Bytes(), &result); err != nil {
						t.Fatal(err, out.String())
					}
					return result
				}
				initial := call("init")
				if initial.Snapshot.Version != 0 || len(initial.Snapshot.Entries) != 0 {
					t.Fatal("source body ran during creation", initial)
				}
				executed := call("execute")
				if executed.Engine != "native-graph" || executed.Profile != profile || executed.ProgramHash != initial.ProgramHash || executed.Outcome == nil || executed.Outcome.Cost != trial.cost || executed.Outcome.Receipt.Version != 1 || executed.Outcome.Receipt.Writes != 2 || len(executed.Outcome.Reads) != 3 {
					t.Fatalf("helper/borrow source command %+v", executed)
				}
				snapshot := call("snapshot")
				if snapshot.Snapshot.Version != 1 || len(snapshot.Snapshot.Entries) != 2 {
					t.Fatal("helper/borrow state did not reopen", snapshot)
				}
			})
		}
	}
}
