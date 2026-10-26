package packageartifact

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/mir"
)

func TestProjectedContextLeavesLargeBodyUnchanged(t *testing.T) {
	artifact := contextVisitFixture(t, 256)
	before := cloneArtifactForTest(artifact)
	context, _, err := projectArtifactContext(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(context, artifact.StableContext) {
		t.Fatal("projection changed the scalar-only type context")
	}
	if !reflect.DeepEqual(cloneArtifactForTest(artifact), before) {
		t.Fatal("collecting types mutated the source artifact")
	}
}

func TestProjectedContextTypeErrorPreservesSource(t *testing.T) {
	artifact := typedProjectionArtifact(t, 1)
	delete(artifact.PackageCoordinates, artifact.PackageID)
	before := cloneArtifactForTest(artifact)
	_, _, err := projectArtifactContext(artifact)
	if err == nil || !strings.Contains(err.Error(), "collect stable function types") {
		t.Fatalf("missing type owner was not rejected during type collection: %v", err)
	}
	if !reflect.DeepEqual(cloneArtifactForTest(artifact), before) {
		t.Fatal("failed type collection mutated the source artifact")
	}
}

func BenchmarkProjectContextLargeBody(b *testing.B) {
	for _, blocks := range []int{256, 1024, 4096} {
		b.Run(fmt.Sprint(blocks), func(b *testing.B) {
			artifact := contextVisitFixture(b, blocks)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				context, _, err := projectArtifactContext(artifact)
				if err != nil || len(context.Structs) != 0 {
					b.Fatalf("project %d blocks: %v", blocks, err)
				}
			}
		})
	}
}

func contextVisitFixture(t testing.TB, blocks int) Artifact {
	t.Helper()
	program, identities, coordinates := extractionContextFixture(1, 1)
	function := &program.Functions[0]
	final := function.Blocks[0].Terminator
	function.Blocks = make([]mir.Block, blocks)
	for i := range blocks - 1 {
		function.Blocks[i] = mir.Block{ID: mir.BlockID(i), Terminator: mir.Terminator{
			Kind: mir.TerminatorGoto, UnwindSuccessor: mir.NoBlock,
			Successors: []mir.BlockID{mir.BlockID(i + 1)},
		}}
	}
	function.Blocks[blocks-1] = mir.Block{ID: mir.BlockID(blocks - 1), Terminator: final}
	artifact, err := Extract(program, function.Span.Package, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}
