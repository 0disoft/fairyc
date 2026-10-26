package packageartifact

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestReassembleSharedBindingsPreservesCallsAndInputs(t *testing.T) {
	artifacts := reassemblyBindingsFixture(t, 32)
	original := cloneArtifactsForTest(artifacts)
	for _, tc := range []struct {
		name string
		run  func([]Artifact) (mir.Program, error)
	}{
		{"ordinary", Reassemble},
		{"storage transfers", ReassembleStorageTransfers},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program, err := tc.run(artifacts)
			if err != nil {
				t.Fatal(err)
			}
			for i := range 31 {
				outcome, err := interpreter.Execute(program, fmt.Sprintf("main::bench_%d", i))
				if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
					t.Fatalf("caller %d: %+v, %v", i, outcome, err)
				}
			}
			if !reflect.DeepEqual(cloneArtifactsForTest(artifacts), original) {
				t.Fatal("reassembly changed its input artifacts")
			}
		})
	}
}

func BenchmarkReassemblePackageBindings(b *testing.B) {
	for _, count := range []int{64, 256, 1024} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			artifacts := reassemblyBindingsFixture(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				program, err := Reassemble(artifacts)
				if err != nil || len(program.Functions) != count {
					b.Fatalf("reassemble %d functions: %v", count, err)
				}
			}
		})
	}
}

func reassemblyBindingsFixture(t testing.TB, count int) []Artifact {
	t.Helper()
	var text strings.Builder
	text.WriteString("module main;\nfn i32 leaf(){return 42;}\n")
	for i := range count - 1 {
		fmt.Fprintf(&text, "fn i32 bench_%d(){return leaf();}\n", i)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text.String())), source.DefaultLimits())
	if err != nil || parsed.AST == nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", err, checked.Semantic)
	}
	identities := map[hir.PackageID]packagegraph.InstanceID{hir.DefaultPackage: {1}}
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: {2}}
	artifacts, err := ExtractAll(checked.MIR, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	return artifacts
}
