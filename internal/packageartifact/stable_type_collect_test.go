package packageartifact

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestCollectStableNominalTypesKeepsOrderAndAdmission(t *testing.T) {
	a := "pkg[" + (packagegraph.CoordinateID{1}).String() + "]::main::A"
	b := "pkg[" + (packagegraph.CoordinateID{2}).String() + "]::main::B"
	for _, test := range []struct {
		name, value string
		want        []string
	}{
		{"empty", "", nil},
		{"scalar", "Option<Result<i32, bool>>", nil},
		{"nominal", a, []string{a}},
		{"success-before-failure", "Result<Option<" + b + ">, " + a + ">", []string{b, a}},
		{"duplicates", "Result<" + a + ", " + a + ">", []string{a, a}},
		{"stable-pointer", "ptr mut Option<" + a + ">", []string{a}},
		{"nested-stable-pointer", "Option<ptr mut " + a + ">", []string{a}},
		{"stable-borrow", "ref Result<" + b + ", " + a + ">", []string{b, a}},
		{"stable-array", "Option<" + a + ">[4]", []string{a}},
		{"nested-stable-array", "Result<" + a + "[4], " + b + ">", []string{a, b}},
		{"stable-slice", a + "[]", []string{a}},
		{"malformed-array-compatibility", a + "[invalid]", []string{a}},
		{"hir-pointer-is-not-stable", "*Option<" + a + ">", nil},
		{"bad-result", "Result<" + a + ", " + b + ", i32>", nil},
		{"bad-coordinate", "pkg[invalid]::main::A", nil},
		{"opaque-depth-boundary", strings.Repeat("Option<", 257) + a + strings.Repeat(">", 257), []string{a}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var got []string
			collectStableNominalTypes(test.value, func(value string) { got = append(got, value) })
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("collected %v, want %v", got, test.want)
			}
		})
	}
}

func TestProjectedContextRetainsDeepCompoundDependency(t *testing.T) {
	artifact := compoundProjectionArtifact(t, 64)
	if err := Verify(artifact); err != nil {
		t.Fatal(err)
	}
	projected, _, err := projectArtifactContext(artifact)
	if err != nil {
		t.Fatal(err)
	}
	want := artifact.StableContext
	want.Structs = nil
	for _, item := range artifact.StableContext.Structs {
		if item.Name != "Unused" {
			want.Structs = append(want.Structs, item)
		}
	}
	if !reflect.DeepEqual(projected, want) {
		t.Fatal("projection lost a nested dependency or retained an unrelated type")
	}
}

func BenchmarkCollectStableNominalTypesNestedOption(b *testing.B) {
	leaf := "pkg[" + (packagegraph.CoordinateID{1}).String() + "]::main::A"
	for _, depth := range []int{16, 64, 256} {
		b.Run(fmt.Sprint(depth), func(b *testing.B) {
			value := strings.Repeat("Option<", depth) + leaf + strings.Repeat(">", depth)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				count := 0
				collectStableNominalTypes(value, func(got string) {
					if got != leaf {
						b.Fatal("collected an unexpected nominal type")
					}
					count++
				})
				if count != 1 {
					b.Fatalf("collected %d types, want one", count)
				}
			}
		})
	}
}

func BenchmarkProjectArtifactContextNestedOption(b *testing.B) {
	for _, depth := range []int{64, 256} {
		b.Run(fmt.Sprint(depth), func(b *testing.B) {
			artifact := compoundProjectionArtifact(b, depth)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				projected, _, err := projectArtifactContext(artifact)
				if err != nil || len(projected.Structs) != 2 {
					b.Fatalf("project nested type context: %v", err)
				}
			}
		})
	}
}

func compoundProjectionArtifact(t testing.TB, depth int) Artifact {
	t.Helper()
	artifact := typeChainProjectionArtifact(t, 2)
	typ := artifact.Context.StructTypes[1].Type
	for range depth {
		typ = hir.Option(typ)
	}
	artifact.Context.StructTypes[0].Fields[0].Type = typ
	var err error
	artifact.StableContext, err = DescribeContext(artifact.Context, artifact.PackageCoordinates, artifactBindings(artifact))
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}
