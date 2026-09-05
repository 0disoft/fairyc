package compiler_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestGenericSyntaxCannotBypassSpecialization(t *testing.T) {
	for _, text := range []string{
		"module main; fn i32 value<T>() { return 42; } fn i32 main() { return 0; }",
		"module main; fn i32 value() { return 42; } fn i32 main() { return value<i32>(); }",
	} {
		file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
		result, err := compiler.Check([]semantic.File{file})
		if err == nil && result.Verified {
			t.Fatal("generic syntax silently erased")
		}
	}
}
