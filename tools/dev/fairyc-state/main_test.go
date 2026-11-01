package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestSourceExecutionPersistsAndRollsBack(t *testing.T) {
	directory := t.TempDir()
	store := t.TempDir()
	path := filepath.Join(directory, "transaction.fy")
	source := func(body string) {
		t.Helper()
		text := "module transaction;\n@contract_uid(\"example.local-state\")\nstate { cell u64 value; }\n@effects(storage_read, storage_write)\nfn void transaction() {\n" + body + "\n}\n"
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	call := func(action string, want int) report {
		t.Helper()
		var out, diagnostics bytes.Buffer
		if code := run([]string{action, store, path}, &out, &diagnostics); code != want {
			t.Fatalf("%s: code %d, want %d: %s", action, code, want, &diagnostics)
		}
		var result report
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatalf("report: %v: %s", err, &out)
		}
		return result
	}
	source("storage::write(\"value\", 42); storage::read(\"value\"); storage::commit();")
	if result := call("init", 0); result.Snapshot.Version != 0 {
		t.Fatal(result)
	}
	if result := call("execute", 0); result.Outcome.Receipt.Version != 1 || result.Outcome.Receipt.Writes != 1 || len(result.Outcome.Reads) != 1 {
		t.Fatal(result)
	}
	before, err := os.ReadFile(filepath.Join(store, "state-v1.bin"))
	if err != nil {
		t.Fatal(err)
	}
	source("storage::write(\"value\", 99); storage::read(\"value\"); storage::reject();")
	call("execute", 6)
	source("storage::write(\"value\", 99); storage::read(\"value\"); storage::trap(\"overflow\");")
	if result := call("execute", 8); result.TrapCategory != "overflow" {
		t.Fatal(result)
	}
	if result := call("snapshot", 0); result.Snapshot.Version != 1 || len(result.Snapshot.Entries) != 1 {
		t.Fatal(result)
	}
	after, _ := os.ReadFile(filepath.Join(store, "state-v1.bin"))
	if !bytes.Equal(before, after) {
		t.Fatal("rejected/trapped attempts changed persistent bytes")
	}
	source("storage::write(\"missing\", 99); storage::commit();")
	if code := run([]string{"execute", store, path}, io.Discard, io.Discard); code != 3 {
		t.Fatalf("invalid source code %d", code)
	}
	after, _ = os.ReadFile(filepath.Join(store, "state-v1.bin"))
	if !bytes.Equal(before, after) {
		t.Fatal("invalid source changed state")
	}
}

func TestInvocationCannotCreateStore(t *testing.T) {
	for _, args := range [][]string{nil, {"reset", "unused", "transaction.fy"}, {"init", "", "transaction.fy"}} {
		if code := run(args, io.Discard, io.Discard); code != 2 {
			t.Fatalf("invalid invocation = %d", code)
		}
	}
	if code := run([]string{"--help"}, io.Discard, io.Discard); code != 0 {
		t.Fatal(code)
	}
}
