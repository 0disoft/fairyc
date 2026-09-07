package semanticsnapshot_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

const genericSnapshotSource = `module main;
/// Copy the input.
fn T id<T>(T value) { let ref T view = ref value; return *view; }
fn T pass<T>(T value) { return id<T>(value); }
/// An unused template.
fn T unused<T>(T value) { return value; }
fn i32 main() { assert(pass<i32>(42) == 42); assert(pass<u32>(7) == 7); return 0; }
`

func compiledGenericSnapshot(t *testing.T) (compiler.Result, []semanticsnapshot.PackageInput) {
	t.Helper()
	result, err := compiler.CheckExplicitFiles([]semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: analyzeSyntax(t, "main.fy", genericSnapshotSource)}})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %+v %v", result.Semantic, err)
	}
	return result, []semanticsnapshot.PackageInput{packageInput(t, hir.DefaultPackage, "example.generic.snapshot", "generic", "1", "main.fy", genericSnapshotSource, "root")}
}

func TestCompilerSnapshotSeparatesOriginalsSpecializationsAndUnknownBodies(t *testing.T) {
	result, packages := compiledGenericSnapshot(t)
	snapshot, err := semanticsnapshot.BuildCompiler(result, packages)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := atlasmap.BuildPlan(snapshot, atlasmap.Input{}); err == nil || !strings.Contains(err.Error(), "cannot preserve generic specializations") {
		t.Fatalf("v1 map silently admitted specialization data: %v", err)
	}
	declarations, functions, specializations, docs := snapshot.Declarations(), snapshot.Functions(), snapshot.Specializations(), snapshot.DocumentationSpans()
	if len(declarations) != 4 || len(functions) != 1 || len(specializations) != 4 || len(docs) != 2 {
		t.Fatalf("snapshot cardinalities: declarations=%d functions=%d specializations=%d docs=%d", len(declarations), len(functions), len(specializations), len(docs))
	}
	if strings.Contains(fmt.Sprint(declarations, functions, specializations, docs), "_fy_generic_") {
		t.Fatal("generated implementation name escaped as source identity")
	}
	byID := map[string]semanticsnapshot.SpecializationSummary{}
	borrows := 0
	for _, specialization := range specializations {
		if specialization.ID == "" || len(specialization.TypeArguments) != 1 || specialization.Function.Function != specialization.Declaration {
			t.Fatalf("invalid specialization: %+v", specialization)
		}
		byID[specialization.ID] = specialization
		borrows += len(specialization.Function.Borrows)
		if specialization.Declaration.Name == "id" && len(specialization.Function.Borrows) != 1 {
			t.Fatal("borrow facts did not stay specialization-specific")
		}
	}
	if len(byID) != 4 || borrows != 2 {
		t.Fatalf("implementation identities/borrows collapsed: %d %d", len(byID), borrows)
	}
	checkCalls := func(function semanticsnapshot.FunctionSummary) {
		for _, call := range function.Calls {
			target, ok := byID[call.Specialization]
			if !ok || target.Declaration != call.Target {
				t.Fatalf("call lost exact specialization: %+v", call)
			}
		}
	}
	checkCalls(functions[0])
	for _, specialization := range specializations {
		checkCalls(specialization.Function)
	}
	if len(functions[0].Calls) != 2 {
		t.Fatal("ordinary caller lost its generic calls")
	}
	unknown := findCompleteness(t, snapshot.Completeness(), "generic-template-bodies")
	if unknown.Status != semanticsnapshot.CompletenessUnavailable || unknown.Unknown != 3 {
		t.Fatalf("uninstantiated body validity overclaimed: %+v", unknown)
	}
	if exact := findCompleteness(t, snapshot.Completeness(), "generic-specializations"); exact.Status != semanticsnapshot.CompletenessExact || exact.Resolved != 4 {
		t.Fatalf("specialization completeness: %+v", exact)
	}
	symbols, err := symbolidentity.FromSnapshot(snapshot)
	if err != nil || len(symbols) != 4 {
		t.Fatalf("source symbol identities: %+v %v", symbols, err)
	}
	before := snapshot.Specializations()
	specializations[0].TypeArguments[0].Name = "corrupted"
	specializations[0].Function.Calls = nil
	functions[0].Calls[0].Specialization = "corrupted"
	if !reflect.DeepEqual(before, snapshot.Specializations()) || snapshot.Functions()[0].Calls[0].Specialization == "corrupted" {
		t.Fatal("specialization or call accessor leaked mutable storage")
	}
}

func TestCompilerSnapshotRejectsMissingOrContradictoryGenericFacts(t *testing.T) {
	result, packages := compiledGenericSnapshot(t)
	if _, err := semanticsnapshot.Build(result.Semantic, result.MIR, packages); err == nil {
		t.Fatal("legacy path silently interpreted generated functions as source declarations")
	}
	missing := result
	missing.GenericProvenance = compiler.GenericProvenance{}
	if _, err := semanticsnapshot.BuildCompiler(missing, packages); err == nil {
		t.Fatal("missing compiler provenance accepted")
	}
	for i := range result.Semantic.HIR.Modules[0].Functions {
		function := &result.Semantic.HIR.Modules[0].Functions[i]
		if strings.HasPrefix(function.Name, "_fy_generic_") {
			function.ResultType = hir.TypeBool
			break
		}
	}
	if _, err := semanticsnapshot.BuildCompiler(result, packages); err == nil {
		t.Fatal("typed implementation/source signature contradiction accepted")
	}
}

func TestCompilerSnapshotPreservesOrdinarySnapshot(t *testing.T) {
	text := "module main; fn i32 _fy_generic_helper() { return 42; } fn i32 main() { return _fy_generic_helper(); }"
	result := compileSinglePackage(t, hir.DefaultPackage, "main.fy", text)
	packages := []semanticsnapshot.PackageInput{packageInput(t, hir.DefaultPackage, "example.ordinary", "ordinary", "1", "main.fy", text, "root")}
	before, err := semanticsnapshot.Build(result.Semantic, result.MIR, packages)
	if err != nil {
		t.Fatal(err)
	}
	after, err := semanticsnapshot.BuildCompiler(result, packages)
	if err != nil {
		t.Fatal(err)
	}
	assertSnapshotsEqual(t, before, after)
	if len(after.Specializations()) != 0 {
		t.Fatal("ordinary code acquired generic implementation records")
	}
}

func TestUnusedGenericSourceStillRequiresProvenance(t *testing.T) {
	text := "module main; fn T unused<T>(T value) { return value; } fn i32 main() { return 0; }"
	result, err := compiler.CheckExplicitFiles([]semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: analyzeSyntax(t, "main.fy", text)}})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %+v %v", result.Semantic, err)
	}
	packages := []semanticsnapshot.PackageInput{packageInput(t, hir.DefaultPackage, "example.unused", "unused", "1", "main.fy", text, "root")}
	snapshot, err := semanticsnapshot.BuildCompiler(result, packages)
	if err != nil || len(snapshot.Declarations()) != 2 || len(snapshot.Specializations()) != 0 {
		t.Fatalf("unused generic snapshot: %+v %v", snapshot.Declarations(), err)
	}
	result.GenericProvenance = compiler.GenericProvenance{}
	if _, err := semanticsnapshot.BuildCompiler(result, packages); err == nil {
		t.Fatal("unused generic declaration was silently dropped")
	}
}
