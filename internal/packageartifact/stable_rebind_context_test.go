package packageartifact

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestStableRebindContextMatchesIndependentBodies(t *testing.T) {
	artifact := typedProjectionArtifact(t, 4)
	stable, err := DescribeStableArtifact(artifact, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	rebound, err := RebindStableArtifactsWithBindings([]StableArtifact{stable}, artifact.PackageCoordinates, 30000, 40000, 1)
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]mir.Function, len(rebound.MIR.Functions))
	for _, function := range rebound.MIR.Functions {
		byName[function.Name] = function
	}
	for i, body := range stable.Functions {
		independent, err := RebindStableLeafBody(body, stable.Context, artifact.PackageCoordinates, rebound.Functions[i].Symbol, 1, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(independent, byName[independent.Name]) {
			t.Fatalf("batch body %s differs from independent rebind", body.ID)
		}
	}
	encodedContext, err := EncodeContextDescriptor(stable.Context, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	_, err = RebindStableLeafBody(stable.Functions[0], stable.Context, artifact.PackageCoordinates, 30000, 1, uint64(len(encodedContext)-1))
	if !errors.Is(err, ErrContextArtifactTooLarge) {
		t.Fatalf("independent rebind ignored its context limit: %v", err)
	}
}

func BenchmarkRebindStableArtifactsTypedContext(b *testing.B) {
	for _, count := range []int{64, 256} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			artifact := typedProjectionArtifact(b, count)
			stable, err := DescribeStableArtifact(artifact, stablePackageArtifactLimit)
			if err != nil {
				b.Fatal(err)
			}
			artifacts := []StableArtifact{stable}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				rebound, err := RebindStableArtifactsWithBindings(artifacts, artifact.PackageCoordinates, hir.SymbolID(30000), hir.SymbolID(40000), mir.LocalID(1))
				if err != nil || len(rebound.MIR.Functions) != count {
					b.Fatalf("rebind %d functions and types: %v", count, err)
				}
			}
		})
	}
}
