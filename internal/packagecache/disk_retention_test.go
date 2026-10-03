package packagecache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiskRetentionSharesBoundsAcrossNamespaces(t *testing.T) {
	for _, bound := range []string{"entries", "bytes"} {
		t.Run(bound, func(t *testing.T) {
			root := t.TempDir()
			limits := DiskLimits{MaxEntries: 10, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute}
			namespaces := []string{".", "generic-v0", "storage-transfer-v0"}
			key := Key{1}
			var stores []*DiskStore
			var encodedBytes uint64
			for i, namespace := range namespaces {
				store, err := NewNamespacedDiskStore(root, namespace, limits)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { store.Close() })
				stores = append(stores, store)
				record := testRecord(t, key, strings.Repeat(string(rune('a'+i)), 32))
				encoded, err := encodeDiskRecord(record, limits.MaxRecordBytes)
				if err != nil {
					t.Fatal(err)
				}
				encodedBytes = uint64(len(encoded))
				if err := store.Put(record); err != nil {
					t.Fatal(err)
				}
				entries, err := readCacheDirectory(store.root, store.keyDirectoryName(key))
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range entries {
					when := time.Unix(1000+int64(i), 0)
					if err := os.Chtimes(filepath.Join(store.keyDirectory(key), entry.Name()), when, when); err != nil {
						t.Fatal(err)
					}
				}
			}
			stats, err := stores[0].Stats()
			if err != nil || stats.Entries != 3 || stats.Bytes != 3*encodedBytes {
				t.Fatalf("aggregate stats: %+v %v", stats, err)
			}
			// Lookups stay isolated even though collection and accounting are shared.
			for i, store := range stores {
				record, found := store.Get(key)
				if !found || string(record.Artifacts[0].Data) != strings.Repeat(string(rune('a'+i)), 32) {
					t.Fatal("namespace lookup crossed formats")
				}
			}
			if bound == "entries" {
				limits.MaxEntries = 2
			} else {
				limits.MaxBytes = 2 * encodedBytes
				limits.MaxRecordBytes = encodedBytes
			}
			boundary, err := os.OpenRoot(root)
			if err != nil {
				t.Fatal(err)
			}
			defer boundary.Close()
			collector, err := NewNamespacedDiskStoreInRoot(boundary, ".", "storage-transfer-v0", limits)
			if err != nil {
				t.Fatal(err)
			}
			defer collector.Close()
			if err := collector.Sweep(); err != nil {
				t.Fatal(err)
			}
			for _, store := range stores {
				stats, err := store.Stats()
				if err != nil || stats.Entries != 2 || stats.Bytes != 2*encodedBytes {
					t.Fatalf("bounded stats: %+v %v", stats, err)
				}
			}
			if _, found := stores[0].Get(key); found {
				t.Fatal("oldest ordinary record survived collection from transfer namespace")
			}
			if _, found := stores[2].Get(key); !found {
				t.Fatal("newest record was evicted")
			}
		})
	}
}

func TestDiskRetentionPreservesUnownedFilesAndLiveTemps(t *testing.T) {
	root := t.TempDir()
	limits := DiskLimits{MaxEntries: 2, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute}
	store, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	keyDir := strings.Repeat("a", 64)
	old := filepath.Join(root, "generic-v0", "v1", keyDir, ".tmp-expired")
	active := filepath.Join(root, "storage-transfer-v0", "v1", keyDir, ".tmp-active")
	unknown := filepath.Join(root, "unknown-v0", "v1", keyDir, strings.Repeat("b", 64)+".json")
	unrelated := filepath.Join(root, "generic-v0", "v1", keyDir, "notes.txt")
	for _, path := range []string{old, active, unknown, unrelated} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("preserve"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	when := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(old, when, when); err != nil {
		t.Fatal(err)
	}
	if err := store.Sweep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("expired sibling temp: %v", err)
	}
	for _, path := range []string{active, unknown, unrelated} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "preserve" {
			t.Fatalf("unowned/live file changed: %s %v", path, err)
		}
	}
	if _, err := NewNamespacedDiskStore(root, "../escape", limits); err == nil {
		t.Fatal("unowned namespace accepted")
	}
}

func TestDiskRetentionDoesNotEraseLargerSiblingRecords(t *testing.T) {
	root := t.TempDir()
	limits := DiskLimits{MaxEntries: 2, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute}
	sibling, err := NewNamespacedDiskStore(root, "generic-v0", limits)
	if err != nil {
		t.Fatal(err)
	}
	defer sibling.Close()
	key := Key{1}
	if err := sibling.Put(testRecord(t, key, strings.Repeat("x", 1024))); err != nil {
		t.Fatal(err)
	}
	limits.MaxRecordBytes = 100
	smallReader, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer smallReader.Close()
	stats, err := smallReader.Stats()
	if err != nil || stats.Entries != 1 || stats.Bytes <= limits.MaxRecordBytes {
		t.Fatalf("larger sibling not counted: %+v %v", stats, err)
	}
	if _, found := sibling.Get(key); !found {
		t.Fatal("small reader erased valid larger sibling")
	}
}
