package packageartifact

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStableCachePutKeepsValidationAndCanonicalBytes(t *testing.T) {
	artifact, key, outputInterface := cacheFixture(t)
	body := stableCacheStateBody(t, artifact)
	compilation, err := DescribeStableCompilationArtifact(artifact, body, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	want, err := EncodeStableCompilationArtifact(compilation, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	limits := packagecache.DiskLimits{MaxEntries: 4, MaxBytes: 4 * stablePackageArtifactLimit, MaxRecordBytes: stablePackageArtifactLimit, TempMaxAge: time.Minute}
	cache, err := NewStableCache(t.TempDir(), limits, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cache.Close() })
	for _, mode := range []string{"MIR", "body hash", "coordinate", "size"} {
		t.Run(mode, func(t *testing.T) {
			current, state := cloneArtifactForTest(artifact), body
			wantError := ""
			switch mode {
			case "MIR":
				current.Functions[0].Body.Entry = 999
				wantError = "verify process-local artifact"
			case "body hash":
				state.Hash[0] ^= 0xff
				wantError = "hash does not match"
			case "coordinate":
				other := artifact
				other.PackageCoordinate[0] ^= 0xff
				state = stableCacheStateBody(t, other)
				wantError = "different package coordinates"
			case "size":
				cache.maxArtifactBytes = uint64(len(want) - 1)
			}
			if wantError != "" {
				if _, err := DescribeStableCompilationArtifact(current, state, cache.maxArtifactBytes); err == nil || !strings.Contains(err.Error(), wantError) {
					t.Fatalf("describe validation = %v, want %q", err, wantError)
				}
			}
			err := cache.Put(key, outputInterface, current, state)
			if mode == "size" {
				if !errors.Is(err, ErrStableCompilationTooLarge) {
					t.Fatalf("size validation = %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), wantError) {
				t.Fatalf("put validation = %v, want %q", err, wantError)
			}
			if _, found := cache.store.Get(key); found {
				t.Fatal("invalid compilation was published")
			}
		})
	}
	cache.maxArtifactBytes = stablePackageArtifactLimit
	if err := cache.Put(key, outputInterface, artifact, body); err != nil {
		t.Fatal(err)
	}
	record, found := cache.store.Get(key)
	if !found || len(record.Artifacts) != 1 || !bytes.Equal(record.Artifacts[0].Data, want) {
		t.Fatal("stored bytes differ from the independently validated public encoder")
	}
}

func BenchmarkStableCachePutTypedContext(b *testing.B) {
	for _, count := range []int{1, 64, 256} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			artifact := typedProjectionArtifact(b, count)
			bodies, err := contractstatemir.DescribeStableHIRStateBodies(hir.Package{}, []hir.PackageID{artifact.PackageID}, map[hir.PackageID]packagegraph.CoordinateID{artifact.PackageID: artifact.PackageCoordinate})
			if err != nil {
				b.Fatal(err)
			}
			limits := packagecache.DiskLimits{MaxEntries: 4, MaxBytes: 4 * stablePackageArtifactLimit, MaxRecordBytes: stablePackageArtifactLimit, TempMaxAge: time.Minute}
			cache, err := NewStableCache(b.TempDir(), limits, stablePackageArtifactLimit)
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() { _ = cache.Close() })
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := cache.Put(packagecache.Key{1}, moduleinterface.Hash{1}, artifact, bodies[artifact.PackageID]); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
