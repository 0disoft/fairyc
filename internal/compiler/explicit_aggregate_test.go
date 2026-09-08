package compiler_test

import (
	"encoding/json"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestExplicitFilesSpecializeAggregatesWithoutGenericFunctions(t *testing.T) {
	for _, item := range []struct {
		name, source     string
		instances, drops int
	}{
		{"struct", `module main; struct Cell<T> { T value; }
fn i32 main() { let Cell<i32> cell = Cell<i32> { .value = 42 }; return cell.value; }`, 1, 0},
		{"resource", `module main;
fn i32 observe(i32 value) { return value; }
fn i32 check_drop() { return 1; }
resource Token<T> { T value; cleanup { let i32 mark = observe(65); } }
fn Token<i32> make(i32 value) { return Token<i32> { .value = value }; }
fn i32 consume(Token<i32> owned) { return owned.value; }
fn i32 main() { let Token<i32> first = make(41); let Token<i32> moved = first; return consume(moved) + check_drop(); }`, 1, 1},
		{"unused", `module main; struct Unused<T> { T value; } fn i32 main() { return 42; }`, 0, 0},
	} {
		t.Run(item.name, func(t *testing.T) {
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", item.source)
			before, err := json.Marshal(file.Module)
			if err != nil {
				t.Fatal(err)
			}
			result, err := compiler.CheckExplicitFiles([]semantic.File{file})
			if err != nil || !result.Verified {
				t.Fatalf("compile: %v %+v", err, result.Semantic)
			}
			if len(result.GenericProvenance.Declarations()) != 1 || len(result.GenericProvenance.Specializations()) != item.instances {
				t.Fatal("aggregate-only provenance missing")
			}
			outcome, err := interpreter.Execute(result.MIR, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || len(outcome.DropTrace) != item.drops {
				t.Fatalf("execute: %+v %v", outcome, err)
			}
			after, err := json.Marshal(file.Module)
			if err != nil || string(before) != string(after) {
				t.Fatal("explicit source mutated")
			}
			t.Run("native", func(t *testing.T) { runGenericAggregateNative(t, item.source) })
		})
	}
}

func TestExplicitAggregateOnlyRejectionPublishesNoProvenance(t *testing.T) {
	for _, text := range []string{
		`module main; struct Cell<T> { T value; } fn i32 main() { let Cell<Unknown> cell = Cell<Unknown> { .value = 42 }; return 0; }`,
		`module main; struct Cell<T> { T value; } fn i32 main() { let Cell<i32, bool> cell = Cell<i32, bool> { .value = 42 }; return 0; }`,
		`module main; resource Token<T> { T value; cleanup {} } fn i32 consume(Token<i32> value) { return value.value; } fn i32 main() { let Token<i32> first = Token<i32> { .value = 42 }; let Token<i32> moved = first; return consume(first); }`,
	} {
		file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
		result, err := compiler.CheckExplicitFiles([]semantic.File{file})
		if err != nil || result.Verified || len(result.GenericProvenance.Declarations()) != 0 || len(result.GenericProvenance.Specializations()) != 0 {
			t.Fatalf("invalid explicit aggregate published: %+v %v", result.Semantic, err)
		}
	}
}
