package semanticsnapshot_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/source"
)

func TestGenericDeclarationsMatchExactPackageFileAfterCanonicalization(t *testing.T) {
	compiled, input := genericFilesFixture(t, 3)
	provenance := compiled.GenericProvenance
	want, err := semanticsnapshot.GenericDeclarations(provenance, []semanticsnapshot.PackageInput{input})
	if err != nil || len(want) != 3 {
		t.Fatalf("source declarations: count=%d, error=%v", len(want), err)
	}
	for i, declaration := range want {
		if declaration.Key.Module != fmt.Sprintf("m%06d", i) || declaration.Key.Name != "id" {
			t.Fatalf("lost canonical source declaration: %+v", declaration.Key)
		}
	}
	input.Files = slices.Clone(input.Files)
	slices.Reverse(input.Files)
	before := slices.Clone(input.Files)
	got, err := semanticsnapshot.GenericDeclarations(provenance, []semanticsnapshot.PackageInput{input})
	if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(input.Files, before) {
		t.Fatalf("file canonicalization changed output or caller-owned files: %v", err)
	}
	for _, missing := range []int{0, 1, 2} {
		t.Run(fmt.Sprintf("missing-%d", missing), func(t *testing.T) {
			candidate := input
			candidate.Files = slices.Delete(slices.Clone(input.Files), missing, missing+1)
			// The same relative path in another package cannot authorize this source.
			other := input
			other.BuildID = hir.PackageID(99)
			other.Instance[0] ^= 0xff
			other.Coordinate[0] ^= 0xff
			_, err := semanticsnapshot.GenericDeclarations(provenance, []semanticsnapshot.PackageInput{candidate, other})
			if err == nil || err.Error() != "generic declaration has no exact source file identity" {
				t.Fatalf("missing package-local file accepted or error changed: %v", err)
			}
		})
	}
}

func TestBuildCompilerKeepsGenericInputsDetached(t *testing.T) {
	compiled, input := genericFilesFixture(t, 3)
	packages := []semanticsnapshot.PackageInput{input}
	snapshot, err := semanticsnapshot.BuildCompiler(compiled, packages)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(input.Files)
	filesBefore := slices.Clone(input.Files)
	originsBefore := compiled.GenericProvenance.Declarations()
	again, err := semanticsnapshot.BuildCompiler(compiled, packages)
	if err != nil || !reflect.DeepEqual(snapshot, again) {
		t.Fatalf("reordering caller-owned files changed the snapshot: %v", err)
	}
	if !reflect.DeepEqual(input.Files, filesBefore) || !reflect.DeepEqual(compiled.GenericProvenance.Declarations(), originsBefore) {
		t.Fatal("compiler snapshot mutated caller-owned files or compiler provenance")
	}
	declarationsBefore, snapshotFilesBefore := snapshot.Declarations(), snapshot.Files()
	// Neither compiler accessors nor package inputs may alias the published snapshot.
	originsBefore[0].Parameters[0] = "corrupted"
	input.Files[0].Path = "corrupted.fy"
	if !reflect.DeepEqual(snapshot.Declarations(), declarationsBefore) || !reflect.DeepEqual(snapshot.Files(), snapshotFilesBefore) {
		t.Fatal("compiler or package input mutation changed the published snapshot")
	}
}

// Parse and compile every file, then derive its real content and package identities.
// Fixture setup is outside the timed source-declaration adapter calls.
func genericFilesFixture(tb testing.TB, count int) (compiler.Result, semanticsnapshot.PackageInput) {
	tb.Helper()
	files := make([]semantic.File, 0, count)
	contents := make([]packagecontent.File, 0, count)
	input := semanticsnapshot.PackageInput{
		BuildID: hir.DefaultPackage, UID: "example.generic.files", Name: "generic-files", Version: "1.0.0",
		Language: "fairyc-language/20", SourceKind: "local", Role: "root",
	}
	for i := range count {
		module := fmt.Sprintf("m%06d", i)
		path := module + ".fy"
		text := []byte("module " + module + "; pub fn T id<T>(T value) { return value; }")
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, text), source.DefaultLimits())
		if err != nil || len(parsed.Tree.Diagnostics) != 0 || parsed.AST == nil {
			tb.Fatalf("parse %s: %v %+v", path, err, parsed.Tree.Diagnostics)
		}
		files = append(files, semantic.File{Package: hir.DefaultPackage, Path: path, Module: parsed.AST})
		contents = append(contents, packagecontent.File{Path: path, Bytes: text})
		digest, lineMap := packagecontent.ComputeFileIdentities(text)
		input.Files = append(input.Files, semanticsnapshot.FileInput{Path: path, Digest: digest, LineMap: lineMap, Kind: semanticsnapshot.FileKindSource})
	}
	result, err := compiler.CheckExplicitFiles(files)
	if err != nil || !result.Verified {
		tb.Fatalf("compile %d generic source files: verified=%t, error=%v", count, result.Verified, err)
	}
	input.Source, err = packagecontent.Compute(contents)
	if err != nil {
		tb.Fatal(err)
	}
	identity := packagegraph.Identity{PackageUID: input.UID, SourceIdentity: input.Source.String(), VersionIdentity: input.Version}
	input.Instance, err = packagegraph.ComputeInstanceID(identity)
	if err != nil {
		tb.Fatal(err)
	}
	input.Coordinate, err = packagegraph.ComputeCoordinateID(packagegraph.CoordinateOf(identity))
	if err != nil {
		tb.Fatal(err)
	}
	return result, input
}

func BenchmarkGenericDeclarationFiles(b *testing.B) {
	for _, count := range []int{32, 1024, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			compiled, input := genericFilesFixture(b, count)
			provenance := compiled.GenericProvenance
			packages := []semanticsnapshot.PackageInput{input}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				declarations, err := semanticsnapshot.GenericDeclarations(provenance, packages)
				if err != nil || len(declarations) != count {
					b.Fatalf("source declarations: count=%d, error=%v", len(declarations), err)
				}
			}
		})
	}
}

func BenchmarkBuildCompilerGenericFiles(b *testing.B) {
	for _, count := range []int{32, 1024, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			compiled, input := genericFilesFixture(b, count)
			packages := []semanticsnapshot.PackageInput{input}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if _, err := semanticsnapshot.BuildCompiler(compiled, packages); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
