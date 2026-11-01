package packageartifact

import (
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
)

func TestStableLookupDiscardPreservesOtherGeneration(t *testing.T) {
	artifact, key, hash := cacheFixture(t)
	body := stableCacheStateBody(t, artifact)
	cache, err := NewStableCache(t.TempDir(), packagecache.DiskLimits{MaxEntries: 4, MaxBytes: 16 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute}, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	if err := cache.Put(key, hash, artifact, body); err != nil {
		t.Fatal(err)
	}
	lookup := cache.Get(key, hash, artifact.Package, body.Hash)
	if lookup.Status != LookupHit {
		t.Fatal("missing observed generation")
	}
	// Caller-owned decoded payload mutations must not alter the rejection receipt.
	lookup.Compilation.MIR.Functions[0].Function.Entry = 999
	assertDiscardPreservesOtherGeneration(t, cache.store, key, lookup.Discard)
	if (StableLookup{}).Discard() {
		t.Fatal("empty lookup discarded a generation")
	}
}

func TestGenericStableLookupDiscardPreservesOtherGeneration(t *testing.T) {
	for _, transfers := range []bool{false, true} {
		name := "generic"
		if transfers {
			name = "storage-transfer"
		}
		t.Run(name, func(t *testing.T) {
			artifact, _, _ := cacheFixture(t)
			create := NewGenericStableCache
			if transfers {
				create = NewStorageTransferStableCache
			}
			cache, err := create(t.TempDir(), packagecache.DiskLimits{MaxEntries: 4, MaxBytes: 16 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute}, 8<<20)
			if err != nil {
				t.Fatal(err)
			}
			defer cache.Close()
			key, hash := packagecache.GenericKey{1}, moduleinterface.GenericHash{2}
			if err := cache.Put(key, hash, artifact); err != nil {
				t.Fatal(err)
			}
			lookup := cache.Get(key, hash, artifact.Package, artifact.PackageCoordinate)
			if lookup.Status != LookupHit {
				t.Fatal("missing observed generation")
			}
			lookup.Artifact.Functions[0].Function.Entry = 999
			assertDiscardPreservesOtherGeneration(t, cache.store, stableDiskAddress(cache.schema(), key), lookup.Discard)
			if (GenericStableLookup{}).Discard() {
				t.Fatal("empty lookup discarded a generation")
			}
		})
	}
}

func assertDiscardPreservesOtherGeneration(t *testing.T, store *packagecache.DiskStore, key packagecache.Key, discard func() bool) {
	t.Helper()
	different, found := store.Get(key)
	if !found {
		t.Fatal("observed record missing")
	}
	different.PublicInterface[0] ^= 0x80
	// A distinct immutable generation arrives after the old lookup, before rejection.
	if err := store.Put(different); err != nil {
		t.Fatal(err)
	}
	if !discard() || discard() {
		t.Fatal("observed generation was not discarded exactly once")
	}
	remaining, found := store.Get(key)
	if !found || remaining.PublicInterface != different.PublicInterface {
		t.Fatal("rejection removed a different generation")
	}
}
