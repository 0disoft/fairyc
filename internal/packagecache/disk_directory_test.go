package packagecache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiskRetentionRemovesEmptyKeyDirectories(t *testing.T) {
	root := t.TempDir()
	limits := DiskLimits{MaxEntries: 2, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute}
	store, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for i := byte(1); i <= 20; i++ {
		if err := store.Put(testRecord(t, Key{i}, "payload")); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, "v1"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("key directories accumulated: %d %v", len(entries), err)
	}
	// Clean up old empty directories too, while preserving unrelated contents.
	orphan := store.keyDirectory(Key{30})
	if err := os.Mkdir(orphan, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := store.Sweep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("empty orphan retained: %v", err)
	}
	stray := store.keyDirectory(Key{31})
	if err := os.WriteFile(stray, []byte("unowned"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Get(Key{31}); ok {
		t.Fatal("regular file used as a key directory")
	}
	if err := store.Sweep(); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(stray); err != nil || string(data) != "unowned" {
		t.Fatalf("unowned file removed: %q %v", data, err)
	}
}

func TestDiskStoreRetriesDirectoryCollectedBeforeTemporaryCreation(t *testing.T) {
	root := t.TempDir()
	limits := DiskLimits{MaxEntries: 2, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute}
	writer, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	collector, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer collector.Close()
	record := testRecord(t, Key{1}, "publication survives")
	hooks := diskPutHooks{
		beforeTemporaryCreate: func(path string) {
			if err := collector.Sweep(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("collector did not remove empty directory: %v", err)
			}
		},
		afterTemporaryClosed: func(path string) {
			if err := collector.Sweep(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("collector removed live temporary: %v", err)
			}
		},
	}
	if err := writer.put(record, hooks); err != nil {
		t.Fatal(err)
	}
	if got, ok := collector.Get(record.Key); !ok || !recordsEqual(got, record) {
		t.Fatal("publication lost after retry")
	}
}
