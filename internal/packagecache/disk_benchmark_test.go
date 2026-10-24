package packagecache

import (
	"bytes"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/moduleinterface"
)

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
