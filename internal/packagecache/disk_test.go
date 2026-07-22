package packagecache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDiskStoreRoundTripsAcrossIndependentInstancesAndConcurrentWriters(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	limits := DiskLimits{MaxEntries: 4, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute}
	first, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	key, err := Compute(testInputs(t))
	if err != nil {
		t.Fatal(err)
	}
	record := testRecord(t, key, "stable-package-bytes")
	var wait sync.WaitGroup
	errors := make(chan error, 2)
	for _, store := range []*DiskStore{first, second} {
		wait.Add(1)
		go func(store *DiskStore) {
			defer wait.Done()
			errors <- store.Put(record)
		}(store)
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	got, found := second.Get(key)
	if !found || !recordsEqual(got, record) {
		t.Fatalf("Get() = %+v, %v", got, found)
	}
	got.Artifacts[0].Data[0] ^= 0xff
	again, found := first.Get(key)
	if !found || !recordsEqual(again, record) {
		t.Fatal("disk store returned aliased or unstable bytes")
	}
	stats, err := first.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Entries != 1 || stats.Bytes == 0 || stats.Hits != 1 {
		t.Fatalf("Stats() = %+v", stats)
	}
}

func TestDiskStoreRejectsCorruptionAndConflictingGenerationsAsMiss(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	limits := DiskLimits{MaxEntries: 4, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute}
	store, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	key, err := Compute(testInputs(t))
	if err != nil {
		t.Fatal(err)
	}
	record := testRecord(t, key, "valid")
	if err := store.Put(record); err != nil {
		t.Fatal(err)
	}
	keyDirectory := store.keyDirectory(key)
	if err := os.WriteFile(filepath.Join(keyDirectory, strings.Repeat("a", 64)+".json"), []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, found := store.Get(key); !found || !recordsEqual(got, record) {
		t.Fatalf("valid generation was lost beside corruption: %+v, %v", got, found)
	}

	conflict := testRecord(t, key, "different-valid-payload")
	encoded, err := encodeDiskRecord(conflict, limits.MaxRecordBytes)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(encoded)
	if err := os.WriteFile(filepath.Join(keyDirectory, hex.EncodeToString(digest[:])+".json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found := store.Get(key); found {
		t.Fatal("conflicting immutable generations produced a cache hit")
	}
	stats, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Rejected < 3 || stats.Misses == 0 {
		t.Fatalf("Stats() = %+v", stats)
	}
}

func TestDiskStoreSweepsExpiredTempsAndOldestGenerationsWithinBounds(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	limits := DiskLimits{MaxEntries: 2, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Second}
	store, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	baseKey, err := Compute(testInputs(t))
	if err != nil {
		t.Fatal(err)
	}
	for index := byte(0); index < 3; index++ {
		record := testRecord(t, keyWithLastByte(baseKey, index), string([]byte{'a' + index}))
		if err := store.Put(record); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	staleDirectory := store.keyDirectory(keyWithLastByte(baseKey, 9))
	if err := os.MkdirAll(staleDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(staleDirectory, ".tmp-abandoned")
	if err := os.WriteFile(stale, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * limits.TempMaxAge)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	if err := store.Sweep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale temporary still exists: %v", err)
	}
	stats, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Entries != 2 || stats.Evictions == 0 {
		t.Fatalf("Stats() = %+v", stats)
	}
}

func TestDiskStoreRejectsMissingLimitsAndOversizedRecordWithoutDisturbingHit(t *testing.T) {
	t.Parallel()
	if _, err := NewDiskStore(t.TempDir(), DiskLimits{}); err == nil {
		t.Fatal("zero disk limits accepted")
	}
	root := t.TempDir()
	limits := DiskLimits{MaxEntries: 2, MaxBytes: 4096, MaxRecordBytes: 1024, TempMaxAge: time.Minute}
	store, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	key, err := Compute(testInputs(t))
	if err != nil {
		t.Fatal(err)
	}
	record := testRecord(t, key, "small")
	if err := store.Put(record); err != nil {
		t.Fatal(err)
	}
	oversized := testRecord(t, keyWithLastByte(key, 1), string(bytes.Repeat([]byte{'x'}, 2048)))
	if err := store.Put(oversized); err == nil {
		t.Fatal("oversized disk record accepted")
	}
	if got, found := store.Get(key); !found || !recordsEqual(got, record) {
		t.Fatal("oversize rejection disturbed an existing disk generation")
	}
}

func TestDiskStoreDeleteIfMatchPreservesDifferentGeneration(t *testing.T) {
	t.Parallel()
	limits := DiskLimits{MaxEntries: 4, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute}
	store, err := NewDiskStore(t.TempDir(), limits)
	if err != nil {
		t.Fatal(err)
	}
	key, err := Compute(testInputs(t))
	if err != nil {
		t.Fatal(err)
	}
	oldRecord := testRecord(t, key, "old")
	currentRecord := testRecord(t, key, "current")
	if err := store.Put(currentRecord); err != nil {
		t.Fatal(err)
	}
	if store.DeleteIfMatch(oldRecord) {
		t.Fatal("DeleteIfMatch removed a different generation")
	}
	if got, found := store.Get(key); !found || !recordsEqual(got, currentRecord) {
		t.Fatalf("current generation after mismatched delete = %+v, %v", got, found)
	}
	if !store.DeleteIfMatch(currentRecord) {
		t.Fatal("DeleteIfMatch did not remove the expected generation")
	}
	if _, found := store.Get(key); found {
		t.Fatal("deleted generation still returned a hit")
	}
}
