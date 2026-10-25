package packagecache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiskStoreCountsEveryConflictAndPreservesUnrelatedFiles(t *testing.T) {
	for _, count := range []int{0, 1, 2, 16} {
		t.Run(fmt.Sprintf("generations_%d", count), func(t *testing.T) {
			limits := DiskLimits{MaxEntries: 64, MaxBytes: 16 << 20, MaxRecordBytes: 1 << 20, TempMaxAge: time.Hour}
			store, err := NewDiskStore(t.TempDir(), limits)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { store.Close() })
			record := diskOwnershipRecord()
			record.Artifacts = []Artifact{{Kind: ArtifactObject, Data: bytes.Repeat([]byte{'x'}, 64<<10)}}
			directory := store.keyDirectory(record.Key)
			if err := os.MkdirAll(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			for index := range count {
				record.Artifacts[0].Data[0] = byte(index)
				encoded, err := encodeDiskRecord(record, limits.MaxRecordBytes)
				if err != nil {
					t.Fatal(err)
				}
				digest := sha256.Sum256(encoded)
				if err := os.WriteFile(filepath.Join(directory, hex.EncodeToString(digest[:])+".json"), encoded, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range []string{strings.Repeat("0", 64) + ".json", strings.Repeat("f", 64) + ".json", "keep.txt", ".tmp-live"} {
				if err := os.WriteFile(filepath.Join(directory, name), []byte("not a cache record"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, found := store.Get(record.Key)
			if found != (count == 1) || found && !recordsEqual(got, record) {
				t.Fatalf("Get found=%v for %d generations", found, count)
			}
			stats, err := store.Stats()
			if err != nil {
				t.Fatal(err)
			}
			wantRejected := uint64(2)
			if count > 1 {
				wantRejected += uint64(count)
			}
			if stats.Rejected != wantRejected || count == 1 && (stats.Entries != 1 || stats.Hits != 1 || stats.Misses != 0) ||
				count != 1 && (stats.Entries != 0 || stats.Hits != 0 || stats.Misses != 1) {
				t.Fatalf("Stats=%+v, generations=%d, want rejected=%d", stats, count, wantRejected)
			}
			for _, name := range []string{"keep.txt", ".tmp-live"} {
				if _, err := os.Stat(filepath.Join(directory, name)); err != nil {
					t.Fatalf("unrelated %s was removed: %v", name, err)
				}
			}
			// Conflict cleanup must not prevent a subsequent correct publication.
			if err := store.Put(record); err != nil {
				t.Fatal(err)
			}
			if got, found := store.Get(record.Key); !found || !recordsEqual(got, record) {
				t.Fatal("cache did not recover after conflict cleanup")
			}
		})
	}
}
