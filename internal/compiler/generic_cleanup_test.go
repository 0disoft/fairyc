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

func genericCleanupProgram() string {
	text := strings.Replace(genericAggregateProgram, "cleanup { let i32 observed = observe(65); }", "cleanup { let T observed = finish<T>(this.payload.value); }", 1)
	return strings.Replace(text, "struct Cell<T>", "fn T finish<T>(T value) { let i32 marked = observe(65); return value; }\nstruct Cell<T>", 1)
}

func TestGenericCleanupParticipatesInPlanningAndOwnership(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", genericCleanupProgram())
	before, _ := json.Marshal(file.Module)
	result, plan, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v %+v", err, result.Semantic)
	}
	if len(plan.Specializations) != 4 {
		t.Fatalf("cleanup specialization missing or duplicated: %+v", plan)
	}
	outcome, err := interpreter.ExecuteWithOptions(result.MIR, "main::main", interpreter.Options{
		TestIntrinsics: map[string]interpreter.TestIntrinsicKind{"main::observe": interpreter.TestIntrinsicWriteStdoutByte}, OutputLimit: 10,
	})
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || len(outcome.DropTrace) != 1 || string(outcome.Stdout) != "A" {
		t.Fatalf("cleanup: %+v %v", outcome, err)
	}
	after, _ := json.Marshal(file.Module)
	if string(before) != string(after) {
		t.Fatal("cleanup template source mutated")
	}
	limits := genericplan.DefaultPlanLimits()
	limits.MaxSpecializations = 3
	limited, _, err := compiler.CheckGenericFunctions(file, limits)
	if limited.Verified || genericplan.ErrorCode(err) != genericplan.CodePlanningBudgetExceeded {
		t.Fatalf("cleanup bypassed budget: %v", err)
	}
	changed := parseCompilerFile(t, hir.DefaultPackage, "main.fy", strings.Replace(genericCleanupProgram(), "observe(65)", "observe(66)", 1))
	updated, next, err := compiler.CheckGenericFunctions(changed, genericplan.DefaultPlanLimits())
	if err != nil || !updated.Verified {
		t.Fatalf("changed helper: %v", err)
	}
	for _, original := range plan.Specializations {
		if strings.HasSuffix(string(original.Declaration), "::Cell") {
			continue
		}
		for _, current := range next.Specializations {
			if original.Declaration == current.Declaration && original.ID == current.ID {
				t.Fatalf("helper change did not invalidate %s", original.Declaration)
			}
		}
	}
}

func TestGenericCleanupRejectsInvalidCalls(t *testing.T) {
	for name, text := range map[string]string{
		"shadow":           strings.Replace(genericCleanupProgram(), "cleanup {", "cleanup { let i32 finish = 0;", 1),
		"arity":            strings.Replace(genericCleanupProgram(), "finish<T>(this.payload.value)", "finish<T, T>(this.payload.value)", 1),
		"unbound":          strings.Replace(genericCleanupProgram(), "finish<T>(this.payload.value)", "finish<U>(this.payload.value)", 1),
		"unknown":          strings.Replace(genericCleanupProgram(), "finish<T>(this.payload.value)", "missing<T>(this.payload.value)", 1),
		"recursive-helper": strings.Replace(genericCleanupProgram(), "return value; }\nstruct Cell<T>", "return finish<T>(value); }\nstruct Cell<T>", 1),
	} {
		t.Run(name, func(t *testing.T) {
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
			result, _, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
			if err == nil && result.Verified {
				t.Fatal("invalid cleanup call verified")
			}
		})
	}
}
