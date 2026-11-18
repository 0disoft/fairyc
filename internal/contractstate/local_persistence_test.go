package contractstate

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func localTestLimits() LocalPersistenceLimits {
	return LocalPersistenceLimits{MaxStateBytes: 1 << 20, MaxEntries: 32, MaxValueBytes: 4096}
}

func localFixture(t *testing.T) (*LocalPersistence, string, Layout, Key) {
	t.Helper()
	layout, key, _ := persistenceFixtureLayout(t)
	directory := t.TempDir()
	store, err := CreateLocalPersistence(directory, layout, localTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, directory, layout, key
}

func TestLocalPersistenceRestartConflictAndLayout(t *testing.T) {
	store, directory, layout, key := localFixture(t)
	value := []byte("committed")
	receipt, err := store.CompareAndPublish(layout.Hash(), 0, []PersistentWrite{{Key: key, Value: value}})
	if err != nil || receipt.Version != 1 || receipt.Writes != 1 {
		t.Fatalf("publish = %+v/%v", receipt, err)
	}
	value[0] = 'X'
	other, err := OpenLocalPersistence(directory, layout, localTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	for _, mutation := range [][]PersistentWrite{nil, {{Key: key, Deleted: true}}} {
		if _, err := other.CompareAndPublish(layout.Hash(), 0, mutation); !errors.Is(err, ErrConflict) {
			t.Fatalf("stale = %v", err)
		}
	}
	snapshot, err := other.Snapshot(layout.Hash())
	if err != nil || string(snapshot.Entries[0].Value) != "committed" {
		t.Fatalf("snapshot = %+v/%v", snapshot, err)
	}
	snapshot.Entries[0].Value[0] = 'X'
	if receipt, err := other.CompareAndPublish(layout.Hash(), 1, nil); err != nil || receipt.Version != 1 {
		t.Fatalf("noop = %+v/%v", receipt, err)
	}
	if _, err := other.PublishLayout(layout.Hash(), 1, layout); err != nil {
		t.Fatal(err)
	}
	builder, _ := NewLayoutBuilder(layout.ContractUID())
	_, _ = DeclareCell(builder, "first", U32)
	_, _ = DeclareCell(builder, "second", U32)
	_, _ = DeclareCell(builder, "third", Bool)
	extended, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if receipt, err := other.PublishLayout(layout.Hash(), 1, extended); err != nil || receipt.Version != 2 {
		t.Fatalf("upgrade = %+v/%v", receipt, err)
	}
	if _, err := store.CompareAndPublish(layout.Hash(), 1, nil); !errors.Is(err, ErrLayoutMismatch) {
		t.Fatalf("old layout = %v", err)
	}
	if _, err := other.PublishLayout(extended.Hash(), 2, layout); !errors.Is(err, ErrIncompatibleLayout) {
		t.Fatalf("downgrade = %v", err)
	}
	if err := other.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := other.Snapshot(extended.Hash()); !errors.Is(err, ErrPersistenceClosed) {
		t.Fatalf("closed = %v", err)
	}
	reopened, err := OpenLocalPersistence(directory, extended, localTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	snapshot, err = reopened.Snapshot(extended.Hash())
	if err != nil || snapshot.Version != 2 || string(snapshot.Entries[0].Value) != "committed" {
		t.Fatalf("restart = %+v/%v", snapshot, err)
	}
	if _, err := reopened.CompareAndPublish(extended.Hash(), 2, []PersistentWrite{{Key: key, Deleted: true}}); err != nil {
		t.Fatal(err)
	}
	snapshot, _ = reopened.Snapshot(extended.Hash())
	if snapshot.Version != 3 || len(snapshot.Entries) != 0 {
		t.Fatalf("delete = %+v", snapshot)
	}
}

func TestLocalPersistenceRejectedWritesPreserveBytes(t *testing.T) {
	store, directory, layout, key := localFixture(t)
	path := filepath.Join(directory, localStateName)
	before, _ := os.ReadFile(path)
	for _, writes := range [][]PersistentWrite{
		{{Key: key}, {Key: key}},
		{{Key: key, Value: []byte{1}, Deleted: true}},
		{{Key: key, Value: make([]byte, 4097)}},
		make([]PersistentWrite, 33),
	} {
		if _, err := store.CompareAndPublish(layout.Hash(), 0, writes); err == nil {
			t.Fatal("invalid writes accepted")
		}
	}
	failure := errors.New("injected before rename")
	store.beforePublish = func() error { return failure }
	if _, err := store.CompareAndPublish(layout.Hash(), 0, []PersistentWrite{{Key: key, Value: []byte("unpublished")}}); !errors.Is(err, failure) {
		t.Fatalf("fault = %v", err)
	}
	store.beforePublish = nil
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("failed publication changed bytes")
	}
	files, _ := filepath.Glob(filepath.Join(directory, ".tmp-*"))
	if len(files) != 0 {
		t.Fatalf("temporary files leaked: %v", files)
	}
	current, err := decodeLocalState(before, localTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	current.version = math.MaxUint64
	encoded, err := encodeLocalState(current, localTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompareAndPublish(layout.Hash(), math.MaxUint64, []PersistentWrite{{Key: key}}); err == nil {
		t.Fatal("version wrapped")
	}
}

func TestLocalPersistenceMergesSortedStateAndMixedWrites(t *testing.T) {
	store, directory, layout, _ := localFixture(t)
	reference, err := NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	for version, writes := range [][]PersistentWrite{
		{{Key: Key{5}, Value: []byte("keep")}, {Key: Key{3}, Value: []byte("old")}, {Key: Key{2}, Value: []byte("delete")}},
		{{Key: Key{6}, Value: []byte{}}, {Key: Key{3}, Value: []byte("new")}, {Key: Key{4}, Deleted: true}, {Key: Key{2}, Deleted: true}, {Key: Key{1}, Value: []byte("insert")}},
	} {
		before, err := json.Marshal(writes)
		if err != nil {
			t.Fatal(err)
		}
		got, err := store.CompareAndPublish(layout.Hash(), uint64(version), writes)
		if err != nil {
			t.Fatal(err)
		}
		want, err := reference.CompareAndPublish(layout.Hash(), uint64(version), writes)
		if err != nil || got != want {
			t.Fatalf("receipt = %+v, want %+v (%v)", got, want, err)
		}
		after, err := json.Marshal(writes)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("publication changed caller writes: %s (%v)", after, err)
		}
	}
	snapshot, err := reference.Snapshot(layout.Hash())
	if err != nil {
		t.Fatal(err)
	}
	want, err := encodeLocalState(localState{layout: layout, version: snapshot.Version, entries: snapshot.Entries}, localTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(directory, localStateName))
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("published bytes differ from reference state: %v", err)
	}
}

func TestLocalPersistenceCorruptionFailsClosed(t *testing.T) {
	store, directory, layout, key := localFixture(t)
	path := filepath.Join(directory, localStateName)
	original, _ := os.ReadFile(path)
	var payload localPayload
	if err := json.Unmarshal(original[:len(original)-sha256.Size], &payload); err != nil {
		t.Fatal(err)
	}
	resign := func(p localPayload) []byte {
		encoded, _ := json.Marshal(p)
		digest := sha256.Sum256(encoded)
		return append(encoded, digest[:]...)
	}
	badLayout := payload
	badLayout.LayoutHash[0] ^= 1
	duplicates := payload
	duplicates.Entries = []Entry{{Key: key}, {Key: key}}
	oversized := bytes.Repeat([]byte{'x'}, int(localTestLimits().MaxStateBytes)+1)
	for _, bad := range [][]byte{nil, original[:len(original)-1], append([]byte("x"), original...), resign(badLayout), resign(duplicates), oversized} {
		if err := os.WriteFile(path, bad, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Snapshot(layout.Hash()); err == nil {
			t.Fatal("corruption accepted")
		}
		if opened, err := OpenLocalPersistence(directory, layout, localTestLimits()); err == nil {
			opened.Close()
			t.Fatal("corrupt store reopened")
		}
		after, _ := os.ReadFile(path)
		if !bytes.Equal(bad, after) {
			t.Fatal("corrupt state was silently repaired")
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if opened, err := OpenLocalPersistence(directory, layout, localTestLimits()); err == nil {
		opened.Close()
		t.Fatal("missing state reset")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing state recreated: %v", err)
	}
	if created, err := CreateLocalPersistence(directory, layout, localTestLimits()); err == nil {
		created.Close()
		t.Fatal("existing authority reset")
	}
}

func TestLocalPersistenceLocksAndProcessRestart(t *testing.T) {
	store, directory, layout, key := localFixture(t)
	if _, err := store.CompareAndPublish(layout.Hash(), 0, []PersistentWrite{{Key: key, Value: []byte("base")}}); err != nil {
		t.Fatal(err)
	}
	child := func(mode string, want int) {
		t.Helper()
		command := exec.Command(os.Args[0], "-test.run=^TestLocalPersistenceProcess$")
		command.Env = append(os.Environ(), "FAIRYC_LOCAL_STATE_MODE="+mode, "FAIRYC_LOCAL_STATE_DIR="+directory)
		output, err := command.CombinedOutput()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatal(err)
			}
			code = exit.ExitCode()
		}
		if code != want {
			t.Fatalf("child %s exit %d, want %d: %s", mode, code, want, output)
		}
	}
	if err := lockLocalState(store.lock); err != nil {
		t.Fatal(err)
	}
	child("locked", 0)
	unlockLocalState(store.lock)
	child("crash-before", 23)
	snapshot, err := store.Snapshot(layout.Hash())
	if err != nil || snapshot.Version != 1 || string(snapshot.Entries[0].Value) != "base" {
		t.Fatalf("crash before = %+v/%v", snapshot, err)
	}
	child("crash-after", 23)
	snapshot, err = store.Snapshot(layout.Hash())
	if err != nil || snapshot.Version != 2 || string(snapshot.Entries[0].Value) != "child" {
		t.Fatalf("crash after = %+v/%v", snapshot, err)
	}
}

func TestLocalPersistenceProcess(t *testing.T) {
	mode := os.Getenv("FAIRYC_LOCAL_STATE_MODE")
	if mode == "" {
		return
	}
	layout, key, _ := persistenceFixtureLayout(t)
	store, err := OpenLocalPersistence(os.Getenv("FAIRYC_LOCAL_STATE_DIR"), layout, localTestLimits())
	if mode == "locked" {
		if store != nil {
			store.Close()
		}
		if !errors.Is(err, ErrPersistenceBusy) {
			t.Fatalf("lock = %v", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if mode == "crash-before" {
		store.beforePublish = func() error { os.Exit(23); return nil }
	}
	if _, err := store.CompareAndPublish(layout.Hash(), 1, []PersistentWrite{{Key: key, Value: []byte("child")}}); err != nil {
		t.Fatal(err)
	}
	os.Exit(23)
}

func TestLocalPersistenceConcurrentCompareHasOneWinner(t *testing.T) {
	store, directory, layout, key := localFixture(t)
	other, err := OpenLocalPersistence(directory, layout, localTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, writer := range []*LocalPersistence{store, other} {
		workers.Go(func() {
			<-start
			_, err := writer.CompareAndPublish(layout.Hash(), 0, []PersistentWrite{{Key: key, Value: []byte("winner")}})
			results <- err
		})
	}
	close(start)
	workers.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		} else if !errors.Is(err, ErrConflict) && !errors.Is(err, ErrPersistenceBusy) {
			t.Fatal(err)
		}
	}
	snapshot, err := store.Snapshot(layout.Hash())
	if err != nil || winners != 1 || snapshot.Version != 1 || len(snapshot.Entries) != 1 {
		t.Fatalf("winners=%d snapshot=%+v error=%v", winners, snapshot, err)
	}
}

func TestLocalPersistenceRejectsSymlinks(t *testing.T) {
	for _, name := range []string{localStateName, localLockName} {
		t.Run(name, func(t *testing.T) {
			store, directory, layout, _ := localFixture(t)
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(directory, name)
			before, _ := os.ReadFile(path)
			outside := filepath.Join(t.TempDir(), "outside")
			if err := os.WriteFile(outside, before, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, path); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			if opened, err := OpenLocalPersistence(directory, layout, localTestLimits()); err == nil {
				opened.Close()
				t.Fatal("symlink accepted")
			}
			after, _ := os.ReadFile(outside)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("outside target changed")
			}
		})
	}
}
