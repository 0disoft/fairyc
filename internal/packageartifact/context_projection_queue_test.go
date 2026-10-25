package packageartifact

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestProjectedContextClosesTypeChainAndCycle(t *testing.T) {
	for _, cycle := range []bool{false, true} {
		t.Run(fmt.Sprintf("cycle=%v", cycle), func(t *testing.T) {
			artifact := typeChainProjectionArtifact(t, 64)
			if cycle {
				// Two edges to the root exercise both cycles and shared dependencies.
				last := &artifact.Context.StructTypes[63]
				last.Fields[0].Type = hir.RawPointer(artifact.Context.StructTypes[0].Type, false)
				last.Fields = append(last.Fields, mir.StructField{Name: "again", Type: last.Fields[0].Type, Span: last.Span})
				var err error
				artifact.StableContext, err = DescribeContext(artifact.Context, artifact.PackageCoordinates, artifactBindings(artifact))
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := Verify(artifact); err != nil {
				t.Fatal(err)
			}
			projected, _, err := projectArtifactContext(artifact)
			if err != nil {
				t.Fatal(err)
			}
			want := artifact.StableContext
			want.Structs = slices.DeleteFunc(slices.Clone(want.Structs), func(item ContextStruct) bool { return item.Name == "Unused" })
			if !reflect.DeepEqual(projected, want) {
				t.Fatal("projection must retain each reachable type exactly once and exclude the unused type")
			}
		})
	}
}

func BenchmarkProjectArtifactContextTypeChain(b *testing.B) {
	for _, count := range []int{256, 1024, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			artifact := typeChainProjectionArtifact(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				context, _, err := projectArtifactContext(artifact)
				if err != nil || len(context.Structs) != count {
					b.Fatalf("project %d reachable types: %v", count, err)
				}
			}
		})
	}
}

func typeChainProjectionArtifact(t testing.TB, count int) Artifact {
	t.Helper()
	artifact := typedProjectionArtifact(t, 1)
	root := artifact.Context.StructTypes[0]
	prefix := strings.TrimSuffix(root.Type.String(), root.Name)
	artifact.Context.StructTypes = make([]mir.StructType, count+1)
	for i := range count {
		name := fmt.Sprintf("Type_%d", i)
		field := hir.TypeI32
		if i+1 < count {
			field = hir.RawPointer(hir.ParseType(fmt.Sprintf("%sType_%d", prefix, i+1)), false)
		}
		artifact.Context.StructTypes[i] = mir.StructType{
			Type: hir.ParseType(prefix + name), Name: name, Span: root.Span,
			Fields: []mir.StructField{{Name: "value", Type: field, Span: root.Span}},
		}
	}
	artifact.Context.StructTypes[count] = mir.StructType{Type: hir.ParseType(prefix + "Unused"), Name: "Unused", Span: root.Span}
	var err error
	artifact.StableContext, err = DescribeContext(artifact.Context, artifact.PackageCoordinates, artifactBindings(artifact))
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}
