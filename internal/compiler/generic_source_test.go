package compiler_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestGenericSourcePlansSubstitutesAndExecutes(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", genericProgram(t))
	before, _ := json.Marshal(file.Module)
	result, plan, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v, semantic=%+v", err, result.Semantic)
	}
	if len(plan.Specializations) != 3 || len(result.MIR.Functions) != 4 {
		t.Fatalf("deduplication failed: %+v, functions=%d", plan, len(result.MIR.Functions))
	}
	outcome, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("execute: %+v %v", outcome, err)
	}
	after, _ := json.Marshal(file.Module)
	if string(before) != string(after) {
		t.Fatal("input AST mutated")
	}
	_, repeated, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
	if err != nil || repeated.CatalogDigest != plan.CatalogDigest {
		t.Fatalf("repeat: %v", err)
	}
	for i := range plan.Specializations {
		if repeated.Specializations[i].ID != plan.Specializations[i].ID {
			t.Fatal("unstable specialization identity")
		}
	}
	file.Package = hir.PackageID(7)
	_, otherOwner, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	if otherOwner.Specializations[0].ID == plan.Specializations[0].ID {
		t.Fatal("package owner missing from specialization identity")
	}
}

func TestGenericSourceRejectsUnadmittedShapes(t *testing.T) {
	cases := map[string]string{
		"reserved-intrinsic":       `fn T source_location<T>(T x) { return x; } fn i32 main() { return source_location<i32>(42); }`,
		"builtin-container-shadow": `fn Option<i32> id<Option>() { return .some(42); } fn i32 main() { return 0; }`,
		"constant-collision":       `const i32 id = 7; fn T id<T>(T x) { return x; } fn i32 main() { return id<i32>(42); }`,
		"qualified":                `fn T id<T>(T x) { return x; } fn i32 main() { return main::id<i32>(42); }`,
		"implicit":                 `fn T id<T>(T x) { return x; } fn i32 main() { return id(42); }`,
		"arity":                    `fn T id<T>(T x) { return x; } fn i32 main() { return id<i32, bool>(42); }`,
		"cycle":                    `fn T id<T>(T x) { return id<T>(x); } fn i32 main() { return id<i32>(42); }`,
		"unknown":                  `fn i32 main() { return missing<i32>(42); }`,
		"nongeneric":               `fn i32 id(i32 x) { return x; } fn i32 main() { return id<i32>(42); }`,
		"public":                   `pub fn T id<T>(T x) { return x; } fn i32 main() { return id<i32>(42); }`,
		"nominal":                  `struct X { i32 value; } fn T id<T>(T x) { return x; } fn i32 main() { return id<X>(42); }`,
		"shadow":                   `fn T id<T>(T x) { return x; } fn i32 main() { let i32 id = 7; return id<i32>(42); }`,
		"parameter-shadow":         `fn T id<T>(T x) { return x; } fn i32 other(i32 id) { return id<i32>(42); } fn i32 main() { return 0; }`,
		"builtin-shadow":           `fn i32 id<i32>(i32 x) { return x; } fn i32 main() { return id<i32>(42); }`,
		"duplicate-parameter":      `fn T id<T, T>(T x) { return x; } fn i32 main() { return id<i32, i32>(42); }`,
		"unresolved-parameter":     `fn T id<T>(T x) { return x; } fn i32 main() { return id<T>(42); }`,
		"type-error":               `fn T id<T>(T x) { return false; } fn i32 main() { return id<i32>(42); }`,
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main; "+text)
			result, _, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
			if err == nil && result.Verified {
				t.Fatal("unsupported source was verified")
			}
		})
	}
}

func TestGenericSourceUsesPlannerBudgets(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", genericProgram(t))
	for _, limits := range []genericplan.PlanLimits{
		{MaxSpecializations: 1, MaxSpecializationsPerDeclaration: 5, MaxExpansionDepth: 5},
		{MaxSpecializations: 5, MaxSpecializationsPerDeclaration: 1, MaxExpansionDepth: 5},
		{MaxSpecializations: 5, MaxSpecializationsPerDeclaration: 5, MaxExpansionDepth: 1},
	} {
		result, _, err := compiler.CheckGenericFunctions(file, limits)
		if result.Verified || genericplan.ErrorCode(err) != genericplan.CodePlanningBudgetExceeded {
			t.Fatalf("budget not enforced: %v", err)
		}
	}
}

func TestGenericSourceBodyChangeInvalidatesDependentIdentity(t *testing.T) {
	compile := func(text string) genericplan.Plan {
		file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
		result, plan, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
		if err != nil || !result.Verified {
			t.Fatalf("compile: %v %+v", err, result.Semantic)
		}
		return plan
	}
	before := compile(genericProgram(t))
	after := compile(strings.Replace(genericProgram(t), "return value;", "let T copy = value; return copy;", 1))
	for i := range before.Specializations {
		if before.Specializations[i].ID == after.Specializations[i].ID {
			t.Fatal("template change did not invalidate transitive specialization")
		}
	}
}
