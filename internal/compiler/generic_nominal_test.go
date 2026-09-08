package compiler_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
)

const nominalGenericProgram = `module main;
fn i32 observe(i32 value) { return value; }
fn i32 check_drop() { return 1; }
@repr(C) struct Point { i32 value; }
struct Box<T> { T value; }
resource Token { i32 value; cleanup { let i32 observed = observe(65); } }
fn T pass<T>(T value) { return value; }
fn i32 consume(Token owned) { return owned.value; }
fn i32 main() {
    let Point point = pass<Point>(Point { .value = 41 });
    let Box<Point> box = Box<Point> { .value = point };
    let Token original = Token { .value = box.value.value };
    let Token moved = pass<Token>(original);
    return consume(moved) + check_drop();
}`

func TestPrivateNominalGenericOwnershipAndIdentity(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", nominalGenericProgram)
	before, _ := json.Marshal(file.Module)
	coordinate := genericCoordinate(t)
	limits := genericplan.DefaultPlanLimits()
	result, plan, err := compiler.CheckNominalGenericFunctions(file, coordinate, limits)
	if err != nil || !result.Verified || len(plan.Specializations) != 3 {
		t.Fatalf("nominal compile: %v %+v %+v", err, result.Semantic, plan)
	}
	outcome, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || len(outcome.DropTrace) != 1 {
		t.Fatalf("execution: %+v %v", outcome, err)
	}
	after, _ := json.Marshal(file.Module)
	if string(before) != string(after) {
		t.Fatal("nominal source mutated")
	}
	_, repeated, err := compiler.CheckNominalGenericFunctions(file, coordinate, limits)
	if err != nil || repeated.CatalogDigest != plan.CatalogDigest {
		t.Fatal("nominal catalog unstable")
	}
	for i, entry := range plan.Specializations {
		if repeated.Specializations[i].ID != entry.ID || !strings.HasPrefix(string(entry.Bindings[0].Type), "source_nominal_v0::package_coordinate_v1_") {
			t.Fatal("nominal identity lost owner or determinism")
		}
	}
	other := coordinate
	other[0] ^= 1
	_, owned, err := compiler.CheckNominalGenericFunctions(file, other, limits)
	if err != nil {
		t.Fatal(err)
	}
	changedFile := parseCompilerFile(t, hir.DefaultPackage, "main.fy", strings.Replace(nominalGenericProgram, "observe(65)", "observe(66)", 1))
	changed, changedPlan, err := compiler.CheckNominalGenericFunctions(changedFile, coordinate, limits)
	if err != nil || !changed.Verified {
		t.Fatalf("changed definition: %v", err)
	}
	for _, prior := range plan.Specializations {
		for _, candidate := range append(owned.Specializations, changedPlan.Specializations...) {
			if prior.ID == candidate.ID {
				t.Fatal("owner or definition change reused nominal specialization")
			}
		}
	}
	legacy, _, err := compiler.CheckGenericFunctions(file, limits)
	if err == nil && legacy.Verified {
		t.Fatal("legacy scalar lane admitted nominal arguments")
	}
	t.Run("native", func(t *testing.T) {
		for _, text := range []string{nominalGenericProgram, strings.Replace(nominalGenericProgram, "@repr(C) struct Point", "struct Point", 1)} {
			runGenericAggregateNativeWithCompiler(t, text, func(file semantic.File) (compiler.Result, error) {
				result, _, err := compiler.CheckNominalGenericFunctions(file, coordinate, limits)
				return result, err
			})
		}
	})
}

func TestPrivateNominalGenericRejectsOwnershipAndScope(t *testing.T) {
	for _, text := range []string{
		strings.Replace(nominalGenericProgram, "return consume(moved) + check_drop();", "let i32 used = consume(moved); return consume(original);", 1),
		strings.Replace(nominalGenericProgram, "pass<Point>", "pass<Missing>", 1),
		strings.Replace(nominalGenericProgram, "module main;", "module main; import other as other;", 1),
		strings.Replace(nominalGenericProgram, "fn T pass<T>(T value)", "fn Point pass<Point>(Point value)", 1),
	} {
		file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
		result, _, err := compiler.CheckNominalGenericFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
		if err == nil && result.Verified {
			t.Fatal("invalid nominal generic source verified")
		}
		if len(result.GenericProvenance.Declarations()) != 0 {
			t.Fatal("failed nominal source published provenance")
		}
	}
}
