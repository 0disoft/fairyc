package packagecache

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDiskBatchStreamsRecordsAndCollectsOnce(t *testing.T) {
	store, err := NewDiskStore(t.TempDir(), DiskLimits{MaxEntries: 64, MaxBytes: 8 << 20, MaxRecordBytes: 1 << 20, TempMaxAge: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	batch := store.BeginBatch()
	scans := 0
	batch.sweep = func() error { scans++; return store.Sweep() }
	for i := 1; i <= 32; i++ {
		if err := batch.Put(testRecord(t, Key{byte(i)}, fmt.Sprintf("record-%d", i))); err != nil {
			t.Fatal(err)
		}
	}
	if scans != 0 {
		t.Fatalf("performed %d premature full scans", scans)
	}
	// An invalid record must not prevent the caller's deferred Close collecting
	// earlier successful publications.
	if err := batch.Put(Record{}); err == nil {
		t.Fatal("invalid record accepted")
	}
	if err := batch.Close(); err != nil {
		t.Fatal(err)
	}
	if err := batch.Close(); err != nil {
		t.Fatal(err)
	}
	if scans != 1 {
		t.Fatalf("32 streamed writes required %d scans, want 1", scans)
	}
	stats, err := store.Stats()
	if err != nil || stats.Entries != 32 {
		t.Fatalf("stats %+v: %v", stats, err)
	}
	if _, hit := store.Get(Key{32}); !hit {
		t.Fatal("batch record not reusable")
	}
	if err := batch.Put(testRecord(t, Key{33}, "late")); err == nil {
		t.Fatal("closed batch admitted publication")
	}
	if err := store.Put(testRecord(t, Key{33}, "ordinary")); err != nil {
		t.Fatal("batch closed borrowed store", err)
	}
}

func TestDiskBatchPreservesEntryAndByteRetention(t *testing.T) {
	for _, byteBound := range []bool{false, true} {
		t.Run(fmt.Sprint(byteBound), func(t *testing.T) {
			limits := DiskLimits{MaxEntries: 3, MaxBytes: 8 << 20, MaxRecordBytes: 1 << 20, TempMaxAge: time.Minute}
			if byteBound {
				limits.MaxEntries = 64
				limits.MaxBytes = 2048
				limits.MaxRecordBytes = 2048
			}
			store, err := NewDiskStore(t.TempDir(), limits)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			batch := store.BeginBatch()
			defer batch.Close()
			for i := 1; i <= 12; i++ {
				if err := batch.Put(testRecord(t, Key{byte(i)}, strings.Repeat("x", 512))); err != nil {
					t.Fatal(err)
				}
			}
			if err := batch.Close(); err != nil {
				t.Fatal(err)
			}
			stats, err := store.Stats()
			if err != nil {
				t.Fatal(err)
			}
			if stats.Entries > limits.MaxEntries || stats.Bytes > limits.MaxBytes {
				t.Fatalf("retention exceeded: %+v", stats)
			}
		})
	}
}

func TestDiskBatchCollectionFailureRemainsObservable(t *testing.T) {
	store, err := NewDiskStore(t.TempDir(), DiskLimits{MaxEntries: 1, MaxBytes: 8 << 20, MaxRecordBytes: 1 << 20, TempMaxAge: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	failure := errors.New("collection failed")
	batch := store.BeginBatch()
	batch.sweep = func() error { return failure }
	if err := batch.Put(testRecord(t, Key{1}, "published")); !errors.Is(err, failure) {
		t.Fatalf("lost collection error: %v", err)
	}
	for range 2 {
		if err := batch.Close(); !errors.Is(err, failure) {
			t.Fatalf("lost close error: %v", err)
		}
	}
}
