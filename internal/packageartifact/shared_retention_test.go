package packageartifact

import (
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
)

func TestStableAdaptersShareRetentionWithoutSharingRecords(t *testing.T) {
	artifact, _, _ := cacheFixture(t)
	root := t.TempDir()
	limits := packagecache.DiskLimits{MaxEntries: 2, MaxBytes: 16 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute}
	ordinary, err := packagecache.NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer ordinary.Close()
	if err := ordinary.Put(packagecache.Record{Key: packagecache.Key{1}, PublicInterface: moduleinterface.Hash{2}, Artifacts: []packagecache.Artifact{{Kind: packagecache.ArtifactMIR, Data: []byte("ordinary")}}}); err != nil {
		t.Fatal(err)
	}
	key, hash := packagecache.GenericKey{1}, moduleinterface.GenericHash{2}
	for _, create := range []func(string, packagecache.DiskLimits, uint64) (*GenericStableCache, error){NewGenericStableCache, NewStorageTransferStableCache} {
		cache, err := create(root, limits, 8<<20)
		if err != nil {
			t.Fatal(err)
		}
		defer cache.Close()
		if err := cache.Put(key, hash, artifact); err != nil {
			t.Fatal(err)
		}
		if cache.Get(key, hash, artifact.Package, artifact.PackageCoordinate).Status != LookupHit {
			t.Fatal("fresh artifact unavailable")
		}
		if _, found := ordinary.Get(stableDiskAddress(cache.schema(), key)); found {
			t.Fatal("namespace leaked into ordinary lookup")
		}
	}
	stats, err := ordinary.Stats()
	if err != nil || stats.Entries != 2 || stats.Bytes > limits.MaxBytes {
		t.Fatalf("combined adapter retention: %+v %v", stats, err)
	}
}
