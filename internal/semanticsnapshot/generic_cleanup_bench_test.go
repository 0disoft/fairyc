package semanticsnapshot

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// Model the post-join boundary: one scalar resource specialization per package.
// Empty cleanups isolate owner preparation from executable-summary construction.
func BenchmarkGenericCleanupOwners(b *testing.B) {
	for _, count := range []int{32, 512, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			metadata := make(map[hir.PackageID]packageMetadata, count)
			implementations := make(map[DeclarationKey]SpecializationSummary, count)
			for i := range count {
				uid := fmt.Sprintf("cleanup.owner.%d", i)
				source, err := packagecontent.Compute([]packagecontent.File{{Path: "main.fy", Bytes: []byte(uid)}})
				if err != nil {
					b.Fatal(err)
				}
				identity := packagegraph.Identity{PackageUID: uid, SourceIdentity: source.String(), VersionIdentity: "1"}
				instance, err := packagegraph.ComputeInstanceID(identity)
				if err != nil {
					b.Fatal(err)
				}
				coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.CoordinateOf(identity))
				if err != nil {
					b.Fatal(err)
				}
				metadata[hir.PackageID(i+1)] = packageMetadata{instance: instance.String(), coord: coordinate.String()}
				original := DeclarationKey{PackageInstance: instance.String(), Module: "main", Kind: DeclarationResource, Name: "Token"}
				implementation := original
				implementation.Name = "_fy_generic_g0_" + fmt.Sprintf("%064x", i)
				implementations[implementation] = SpecializationSummary{Declaration: original, Aggregate: &Declaration{
					Key: original, Provenance: ProvenanceCompilerObserved, DropKind: "empty",
					Type: TypeRef{Kind: TypeNominal, Name: "main::Token", PackageCoordinate: coordinate.String()},
				}}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				summaries, borrows, resolved, external, err := buildGenericCleanupSummaries(compiler.Result{}, implementations, metadata, nil, nil, nil)
				if err != nil || len(summaries) != 0 || len(borrows) != 0 || resolved != 0 || external != 0 {
					b.Fatalf("empty-cleanup summaries changed: %v", err)
				}
			}
		})
	}
}
