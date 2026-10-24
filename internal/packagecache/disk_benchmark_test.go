package packagecache

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/moduleinterface"
)

func BenchmarkDiskEncodeRejectOversize(b *testing.B) {
	for _, size := range []int{128 << 10, 4 << 20, 16 << 20} {
		b.Run(fmt.Sprintf("payload_%d", size), func(b *testing.B) {
			record := diskOwnershipRecord()
			record.Artifacts = []Artifact{{Kind: ArtifactObject, Data: bytes.Repeat([]byte{'x'}, size)}}
			b.ReportAllocs()
			for b.Loop() {
				encoded, err := encodeDiskRecord(record, 64<<10)
				if encoded != nil || !errors.Is(err, ErrRecordTooLarge) {
					b.Fatalf("got %d bytes, %v; want oversized rejection", len(encoded), err)
				}
			}
		})
	}
}

func BenchmarkDiskStoreGet4MiB(b *testing.B) {
	const size = 4 << 20
	store, err := NewDiskStore(b.TempDir(), DiskLimits{
		MaxEntries: 4, MaxBytes: 32 << 20, MaxRecordBytes: 16 << 20, TempMaxAge: time.Hour,
	})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { store.Close() })
	key := Key(sha256.Sum256([]byte("disk-get-benchmark")))
	record := Record{Key: key, PublicInterface: moduleinterface.Hash(key),
		Artifacts: []Artifact{{Kind: ArtifactObject, Data: bytes.Repeat([]byte{'x'}, size)}}}
	if err := store.Put(record); err != nil {
		b.Fatal(err)
	}
	if _, found := store.Get(key); !found {
		b.Fatal("warm-up cache miss")
	}
	b.ReportAllocs()
	b.SetBytes(size)
	b.ResetTimer()
	for range b.N {
		got, found := store.Get(key)
		if !found || len(got.Artifacts) != 1 || len(got.Artifacts[0].Data) != size {
			b.Fatal("cache miss or wrong artifact size")
		}
	}
}
