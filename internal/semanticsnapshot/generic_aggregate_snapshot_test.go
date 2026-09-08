package semanticsnapshot_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const aggregateSnapshotSource = `module main;
/// Payload.
struct Cell<T> { T value; }
/// Owned payload.
resource Token<T> { Cell<T> payload; cleanup { let T copy = this.payload.value; let ref T view = ref copy; let T observed = finish<T>(*view); } }
fn T finish<T>(T value) { let ref T view = ref value; return *view; }
fn Token<T> make<T>(T value) { return Token<T> { .payload = Cell<T> { .value = value } }; }
/// Never instantiated.
struct Unused<T> { T value; }
fn i32 main() { let Token<i32> first = make<i32>(42); let Token<u32> second = make<u32>(7); return first.payload.value; }`

func TestOrdinaryCleanupBorrowKeepsPrivateOwner(t *testing.T) {
	text := `module main;
resource Token { i32 value; cleanup { let i32 copy = this.value; let ref i32 view = ref copy; assert(*view == 42); } }
fn i32 main() { let Token value = Token { .value = 42 }; return 0; }`
	file := semantic.File{Package: hir.DefaultPackage, Path: "main.fy", Module: analyzeSyntax(t, "main.fy", text)}
	compiled, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !compiled.Verified || len(compiled.Semantic.Borrows) != 1 {
		t.Fatalf("cleanup borrow analysis: %v %+v", err, compiled.Semantic)
	}
	packages := []semanticsnapshot.PackageInput{packageInput(t, hir.DefaultPackage, "example.cleanup", "cleanup", "1", "main.fy", text, "root")}
	snapshot, err := semanticsnapshot.BuildCompiler(compiled, packages)
	if err != nil || len(snapshot.Functions()) != 1 || len(snapshot.Functions()[0].Borrows) != 0 {
		t.Fatalf("private cleanup leaked into source function: %v %+v", err, snapshot.Functions())
	}
	for _, function := range compiled.MIR.Functions {
		if function.DropGlueFor == "" {
			compiled.Semantic.Borrows[0].Borrow = function.Span
			break
		}
	}
	if _, err := semanticsnapshot.BuildCompiler(compiled, packages); err == nil {
		t.Fatal("cleanup borrow with unrelated span accepted")
	}
}

func TestAggregateSnapshotJoinsTypesCleanupCallsAndDocumentation(t *testing.T) {
	file := semantic.File{Package: hir.DefaultPackage, Path: "main.fy", Module: analyzeSyntax(t, "main.fy", aggregateSnapshotSource)}
	compiled, _, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
	if err != nil || !compiled.Verified {
		t.Fatalf("compile: %v %+v", err, compiled.Semantic)
	}
	packages := []semanticsnapshot.PackageInput{packageInput(t, hir.DefaultPackage, "example.aggregate.snapshot", "aggregate", "1", "main.fy", aggregateSnapshotSource, "root")}
	snapshot, err := semanticsnapshot.BuildCompiler(compiled, packages)
	if err != nil {
		t.Fatal(err)
	}
	declarations, specializations, docs := snapshot.Declarations(), snapshot.Specializations(), snapshot.DocumentationSpans()
	if len(declarations) != 6 || len(specializations) != 8 || len(docs) != 3 || len(snapshot.Functions()) != 1 {
		t.Fatalf("counts: %d/%d/%d/%d", len(declarations), len(specializations), len(docs), len(snapshot.Functions()))
	}
	if strings.Contains(fmt.Sprint(declarations, specializations, docs), "_fy_generic_") {
		t.Fatal("generated identity leaked")
	}
	byID := map[string]semanticsnapshot.SpecializationSummary{}
	for _, value := range specializations {
		byID[value.ID] = value
	}
	cleanup, aggregates, borrows := 0, 0, 0
	for _, value := range specializations {
		if value.Aggregate == nil {
			borrows += len(value.Function.Borrows)
			continue
		}
		aggregates++
		if value.Function.Function != (semanticsnapshot.DeclarationKey{}) || value.Aggregate.Key != value.Declaration || value.Aggregate.Provenance != semanticsnapshot.ProvenanceCompilerObserved {
			t.Fatalf("aggregate confused with function: %+v", value)
		}
		if value.Declaration.Kind == semanticsnapshot.DeclarationResource {
			if value.Cleanup == nil || value.Cleanup.Function != value.Declaration || len(value.Cleanup.Calls) != 1 || len(value.Cleanup.Borrows) != 1 || value.Aggregate.DropKind != "non-empty" {
				t.Fatalf("cleanup lost: %+v", value)
			}
			cleanup++
			call := value.Cleanup.Calls[0]
			helper, found := byID[call.Specialization]
			if !found || helper.Declaration != call.Target || helper.Declaration.Name != "finish" || fmt.Sprint(helper.TypeArguments) != fmt.Sprint(value.TypeArguments) {
				t.Fatalf("cleanup called wrong specialization: %+v", call)
			}
			field := value.Aggregate.Fields[0].Type
			if field.Name != "main::Cell" || len(field.Arguments) != 1 || field.Arguments[0].Name != value.TypeArguments[0].Name {
				t.Fatalf("nested aggregate type lost: %+v", field)
			}
			value.Aggregate.Fields[0].Type.Arguments[0].Name = "corrupt"
			value.Cleanup.Calls[0].Specialization = "corrupt"
		}
	}
	if aggregates != 4 || cleanup != 2 || borrows != 2 {
		t.Fatalf("observations: %d/%d/%d", aggregates, cleanup, borrows)
	}
	if strings.Contains(fmt.Sprint(snapshot.Specializations()), "corrupt") {
		t.Fatal("snapshot aggregate or cleanup getter aliases storage")
	}
	if _, err := atlasmap.BuildPlanV2(snapshot, atlasmap.Input{}); err == nil || !strings.Contains(err.Error(), "aggregate specialization transport") {
		t.Fatalf("old map accepted aggregate facts: %v", err)
	}
	compiled.Semantic.HIR.Modules[0].Structs[0].Fields[0].Type = hir.TypeBool
	if _, err := semanticsnapshot.BuildCompiler(compiled, packages); err == nil {
		t.Fatal("contradicting concrete aggregate field accepted")
	}
}
