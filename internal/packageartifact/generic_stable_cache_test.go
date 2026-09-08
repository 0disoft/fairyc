package packageartifact

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
)

func TestGenericStableCacheIsolationAndQuarantine(t *testing.T) {
	testStableCacheIsolationAndQuarantine(t, false)
}

func TestStorageTransferStableCacheIsolationAndQuarantine(t *testing.T) {
	testStableCacheIsolationAndQuarantine(t, true)
}

func testStableCacheIsolationAndQuarantine(t *testing.T, transfers bool) {
	t.Helper()
	artifact, _, _ := cacheFixture(t)
	root := t.TempDir()
	limits := packagecache.DiskLimits{MaxEntries: 4, MaxBytes: 16 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute}
	createCache := NewGenericStableCache
	otherCache := NewStorageTransferStableCache
	if transfers {
		createCache, otherCache = otherCache, createCache
	}
	cache, err := createCache(root, limits, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cache.Close() })
	key, hash := packagecache.GenericKey{1}, moduleinterface.GenericHash{2}
	get := func() GenericStableLookup { return cache.Get(key, hash, artifact.Package, artifact.PackageCoordinate) }
	if get().Status != LookupMiss {
		t.Fatal("cold hit")
	}
	if err := cache.Put(key, hash, artifact); err != nil {
		t.Fatal(err)
	}
	if get().Status != LookupHit {
		t.Fatal("missing warm artifact")
	}
	ordinary, err := packagecache.NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer ordinary.Close()
	address := stableDiskAddress(cache.schema(), key)
	if _, found := ordinary.Get(address); found {
		t.Fatal("generic record leaked into ordinary directory")
	}
	other, err := otherCache(root, limits, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if other.Get(key, hash, artifact.Package, artifact.PackageCoordinate).Status != LookupMiss {
		t.Fatal("stable cache format namespaces overlap")
	}
	if address == stableDiskAddress(other.schema(), key) || stableDiskInterface(cache.schema(), hash) == stableDiskInterface(other.schema(), hash) {
		t.Fatal("stable cache storage domains overlap")
	}
	cache.maxArtifactBytes = 1
	if err := cache.Put(key, hash, artifact); !errors.Is(err, ErrContextArtifactTooLarge) {
		t.Fatalf("oversize: %v", err)
	}
	cache.maxArtifactBytes = 8 << 20
	if get().Status != LookupHit {
		t.Fatal("oversize write removed prior generation")
	}
	record, found := cache.store.Get(address)
	if !found {
		t.Fatal("missing original generation")
	}
	// Even a record copied to the other directory and relabeled at the outer
	// DiskStore layer cannot cross the inner cache/artifact schema boundary.
	foreignRecord := record
	foreignRecord.Key = stableDiskAddress(other.schema(), key)
	foreignRecord.PublicInterface = stableDiskInterface(other.schema(), hash)
	if err := other.store.Put(foreignRecord); err != nil {
		t.Fatal(err)
	}
	if other.Get(key, hash, artifact.Package, artifact.PackageCoordinate).Status != LookupRejected || get().Status != LookupHit {
		t.Fatal("cross-format rejection damaged the original namespace")
	}
	cache.maxArtifactBytes = uint64(len(record.Artifacts[0].Data) - 1)
	if err := cache.Put(key, hash, artifact); !errors.Is(err, ErrStableArtifactTooLarge) {
		t.Fatalf("envelope limit: %v", err)
	}
	cache.maxArtifactBytes = 8 << 20
	if get().Status != LookupHit {
		t.Fatal("oversize envelope disturbed prior generation")
	}
	if cache.Get(key, hash, artifact.Package, [32]byte{}).Status != LookupRejected || get().Status != LookupHit {
		t.Fatal("missing expectation disturbed a valid record")
	}
	for _, mutate := range []func(*packagecache.Record){
		func(r *packagecache.Record) { r.PublicInterface[0] ^= 1 },
		func(r *packagecache.Record) { r.Artifacts[0].Data = []byte(`{"schema":"unknown"}`) },
		func(r *packagecache.Record) { r.Artifacts[0].Data = append(r.Artifacts[0].Data, ' ') },
		func(r *packagecache.Record) { r.Artifacts[0].Kind = packagecache.ArtifactCompilation },
	} {
		if err := cache.Put(key, hash, artifact); err != nil {
			t.Fatal(err)
		}
		record, found := cache.store.Get(address)
		if !found {
			t.Fatal("missing stored record")
		}
		cache.store.DeleteIfMatch(record)
		mutate(&record)
		if err := cache.store.Put(record); err != nil {
			t.Fatal(err)
		}
		if get().Status != LookupRejected || get().Status != LookupMiss {
			t.Fatal("bad generation was not quarantined")
		}
	}
	if err := cache.Put(key, hash, artifact); err != nil {
		t.Fatal(err)
	}
	otherCoordinate := artifact.PackageCoordinate
	otherCoordinate[0] ^= 1
	if cache.Get(key, hash, artifact.Package, otherCoordinate).Status != LookupRejected || get().Status != LookupMiss {
		t.Fatal("foreign coordinate accepted")
	}
	stats, err := cache.Stats()
	if err != nil || stats.Rejected != 6 {
		t.Fatalf("stats: %+v %v", stats, err)
	}
}

func TestGenericStableCacheRequiresBoundsAndOpenedContainment(t *testing.T) {
	testStableCacheBoundsAndContainment(t, false)
}

func TestStorageTransferStableCacheRequiresBoundsAndOpenedContainment(t *testing.T) {
	testStableCacheBoundsAndContainment(t, true)
}

func testStableCacheBoundsAndContainment(t *testing.T, transfers bool) {
	t.Helper()
	createCache, createBoundCache := NewGenericStableCache, NewGenericStableCacheInRoot
	if transfers {
		createCache, createBoundCache = NewStorageTransferStableCache, NewStorageTransferStableCacheInRoot
	}
	limits := packagecache.DiskLimits{MaxEntries: 2, MaxBytes: 16 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute}
	if _, err := createCache("", limits, 8<<20); err == nil {
		t.Fatal("empty root accepted")
	}
	if _, err := createCache(t.TempDir(), limits, 0); err == nil {
		t.Fatal("missing bound accepted")
	}
	if _, err := createBoundCache(nil, ".", limits, 8<<20); err == nil {
		t.Fatal("nil boundary accepted")
	}
	boundary, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer boundary.Close()
	if _, err := createBoundCache(boundary, "../escape", limits, 8<<20); err == nil {
		t.Fatal("boundary escape accepted")
	}
	cache, err := createBoundCache(boundary, "cache", limits, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}
}
