package packagecache

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiskScanBudgetFailsClosedWithoutEviction(t *testing.T) {
	root := t.TempDir()
	limits := DiskLimits{MaxEntries: 2, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute}
	store, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	first := testRecord(t, Key{1}, "first")
	second := testRecord(t, Key{2}, "second")
	for _, record := range []Record{first, second} {
		if err := store.Put(record); err != nil {
			t.Fatal(err)
		}
	}
	firstDirectory := store.keyDirectory(first.Key)
	entries, err := os.ReadDir(firstDirectory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("first generation entries = %v, %v", entries, err)
	}
	generationPath := filepath.Join(firstDirectory, entries[0].Name())
	if err := os.WriteFile(filepath.Join(firstDirectory, "unknown"), []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}

	store.scanEntryLimit = 1
	if _, found := store.Get(first.Key); found {
		t.Fatal("an incomplete key-directory scan produced a cache hit")
	}

	store.scanEntryLimit = 2
	store.limits.MaxEntries = 1
	if err := store.Sweep(); !errors.Is(err, ErrDiskScanLimit) {
		t.Fatalf("over-limit Sweep() error = %v", err)
	}
	if _, err := store.Stats(); !errors.Is(err, ErrDiskScanLimit) {
		t.Fatalf("over-limit Stats() error = %v", err)
	}
	if store.evicted != 0 {
		t.Fatalf("partial scan evicted %d generations", store.evicted)
	}
	for _, record := range []Record{first, second} {
		path := store.keyDirectory(record.Key)
		files, err := os.ReadDir(path)
		if err != nil || len(files) < 1 {
			t.Fatalf("generation directory %s changed: %v, %v", path, files, err)
		}
	}
	if contents, err := os.ReadFile(filepath.Join(firstDirectory, "unknown")); err != nil || string(contents) != "preserve" {
		t.Fatalf("partial scan pruned unknown file: %q, %v", contents, err)
	}
	if _, err := os.Stat(generationPath); err != nil {
		t.Fatalf("partial scan pruned generation: %v", err)
	}

	store.scanEntryLimit = MaxDiskScanEntries
	if got, found := store.Get(first.Key); !found || !recordsEqual(got, first) {
		t.Fatal("bounded miss hid a complete later hit")
	}
}

func TestDiskScanBudgetIsCumulativeAcrossNamespaces(t *testing.T) {
	root := t.TempDir()
	limits := DiskLimits{MaxEntries: 4, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute}
	for _, namespace := range []string{".", "generic-v0"} {
		store, err := NewNamespacedDiskStore(root, namespace, limits)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Put(testRecord(t, Key{1}, namespace)); err != nil {
			store.Close()
			t.Fatal(err)
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	}
	store, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	store.scanEntryLimit = 2
	if _, err := store.Stats(); !errors.Is(err, ErrDiskScanLimit) {
		t.Fatalf("cross-namespace budget error = %v", err)
	}
}
