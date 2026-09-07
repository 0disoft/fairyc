package compiler_test

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
)

const genericAggregateProgram = `module main;
fn i32 observe(i32 value) { return value; }
fn i32 check_drop() { return 1; }
struct Cell<T> { T value; }
resource Token<T> {
    Cell<T> payload;
    cleanup { let i32 observed = observe(65); }
}
fn Token<T> make<T>(T value) {
    return Token<T> { .payload = Cell<T> { .value = value } };
}
fn i32 consume(Token<i32> owned) { return owned.payload.value; }
fn i32 main() {
    let Token<i32> original = make<i32>(41);
    let Token<i32> moved = original;
    let i32 value = consume(moved);
    return value + check_drop();
}
`

func TestGenericAggregateExpansionMovesAndDrops(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", genericAggregateProgram)
	before, _ := json.Marshal(file.Module)
	result, plan, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v %+v", err, result.Semantic)
	}
	if len(plan.Specializations) != 3 || len(result.MIR.ResourceTypes) != 1 || len(result.MIR.StructTypes) != 1 {
		t.Fatalf("unexpected expansion: %+v", plan)
	}
	outcome, err := interpreter.ExecuteWithOptions(result.MIR, "main::main", interpreter.Options{
		TestIntrinsics: map[string]interpreter.TestIntrinsicKind{"main::observe": interpreter.TestIntrinsicWriteStdoutByte}, OutputLimit: 10,
	})
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || len(outcome.DropTrace) != 1 || string(outcome.Stdout) != "A" {
		t.Fatalf("execute: %+v %v", outcome, err)
	}
	after, _ := json.Marshal(file.Module)
	if string(before) != string(after) {
		t.Fatal("aggregate source AST mutated")
	}
	_, repeated, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
	if err != nil || repeated.CatalogDigest != plan.CatalogDigest {
		t.Fatalf("repeat: %v", err)
	}
	for i := range plan.Specializations {
		if repeated.Specializations[i].ID != plan.Specializations[i].ID {
			t.Fatal("unstable aggregate identity")
		}
	}
	limits := genericplan.DefaultPlanLimits()
	limits.MaxSpecializations = 2
	limited, _, err := compiler.CheckGenericFunctions(file, limits)
	if limited.Verified || genericplan.ErrorCode(err) != genericplan.CodePlanningBudgetExceeded {
		t.Fatalf("combined budget: %v", err)
	}
	changed := parseCompilerFile(t, hir.DefaultPackage, "main.fy", strings.Replace(genericAggregateProgram, "observe(65)", "observe(66)", 1))
	updated, changedPlan, err := compiler.CheckGenericFunctions(changed, genericplan.DefaultPlanLimits())
	if err != nil || !updated.Verified {
		t.Fatalf("changed compile: %v", err)
	}
	for _, before := range plan.Specializations {
		if strings.HasSuffix(string(before.Declaration), "::make") || strings.HasSuffix(string(before.Declaration), "::Token") {
			for _, after := range changedPlan.Specializations {
				if before.Declaration == after.Declaration && before.ID == after.ID {
					t.Fatal("cleanup change did not invalidate aggregate or factory identity")
				}
			}
		}
	}
}

func TestGenericAggregateExpansionRejectsInvalidOwnershipAndShapes(t *testing.T) {
	cases := map[string]string{
		"builtin-name":        genericAggregateProgram + `struct Option<T> { T value; }`,
		"use-after-move":      strings.Replace(genericAggregateProgram, "return value + check_drop();", "return consume(original);", 1),
		"missing-arguments":   `module main; resource R<T> { i32 value; } fn i32 main() { let R v = R { .value = 42 }; return v.value; }`,
		"cycle":               `module main; struct R<T> { R<T> value; } fn i32 main() { let R<i32> r = R<i32> { .value = 42 }; return 0; }`,
		"public":              strings.Replace(genericAggregateProgram, "resource Token", "pub resource Token", 1),
		"duplicate":           strings.Replace(genericAggregateProgram, "struct Cell<T>", "struct Cell { i32 value; } struct Cell<T>", 1),
		"duplicate-parameter": genericAggregateProgram + `struct Wrong<T, T> { T value; }`,
		"unknown-argument":    `module main; struct R<T> { T value; } fn i32 main() { let R<Missing> r = R<Missing> { .value = 42 }; return 0; }`,
		"field-mismatch":      strings.Replace(genericAggregateProgram, ".value = value", ".value = false", 1),
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
			result, _, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
			if err == nil && result.Verified {
				t.Fatal("invalid aggregate verified")
			}
		})
	}
}

func TestGenericAggregateCleanupRunsExactlyOnceMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native proof requires Windows amd64")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(genericAggregateProgram, "fn i32 observe(i32 value) { return value; }", `
@link_name("fairyc_fixture_generic_mark") unsafe extern C fn i32 mark_raw(i32 value);
@link_name("fairyc_fixture_generic_check") unsafe extern C fn i32 check_raw();
fn i32 observe(i32 value) { return unsafe { mark_raw(value) }; }`, 1)
	text = strings.Replace(text, "fn i32 check_drop() { return 1; }", "fn i32 check_drop() { return unsafe { check_raw() }; }", 1)
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	result, _, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("native compile: %v %+v", err, result.Semantic)
	}
	unit, err := cbackend.Emit(result.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	scalarRuntime, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	scalarRuntime += `
static int32_t probe_calls = 0;
static int32_t probe_value = 0;
int32_t fairyc_fixture_generic_mark(int32_t value) { probe_calls++; probe_value = value; return value; }
int32_t fairyc_fixture_generic_check(void) { return probe_calls == 1 && probe_value == 65 ? 1 : 0; }
`
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalarRuntime)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsHost(artifact)
			if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("native execution: %+v %v", outcome, err)
			}
		})
	}
}
