package compiler_test

import (
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"testing"
)

func TestArgumentLabelsGenericExpansionRejects(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main; fn T id<T>(T value) { return value; } fn i32 main() { return id<i32>(.value = 42); }")
	result, _, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	if result.Verified {
		t.Fatal("generic expansion admitted labels")
	}
	for _, d := range result.Semantic.Diagnostics {
		if d.Code == "E1014" {
			return
		}
	}
	t.Fatalf("missing label diagnostic: %+v", result.Semantic)
}
