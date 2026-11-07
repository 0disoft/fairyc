package compiler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

func checkClosureLocals(t *testing.T, text string) compiler.Result {
	t.Helper()
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	before, err := json.Marshal(file.Module)
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("closure locals: %v diagnostics=%+v unsupported=%+v", err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
	after, err := json.Marshal(file.Module)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("compiler mutated caller closure AST")
	}
	return result
}

func TestCopiedClosureLocalsExecution(t *testing.T) {
	const text = `module main;fn i32 main(){
var i32 base=40;
let c=fn [i32 base] i32(i32 x){let i32 snapshot=base;let twice=x*2;let answer=snapshot+twice;return answer;};
base=100;assert(c(1)==42);assert(c(2)==44);
let tail=fn [] i32(i32 x){let first=x+1;let i32 second=first*2;second};assert(tail(20)==42);
return c(1);}`
	result := checkClosureLocals(t, text)
	out, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 42 {
		t.Fatalf("snapshot/per-call locals: %+v %v", out, err)
	}
	// The normal verified helper path must preserve the same semantics natively.
	runExpandedLanguageNative(t, result, "main::main", "")
}

func TestCopiedClosureLocalsBounds(t *testing.T) {
	var locals strings.Builder
	locals.WriteString("let i32 a0=x;")
	for i := 1; i < 16; i++ {
		fmt.Fprintf(&locals, "let a%d=a%d+1;", i, i-1)
	}
	for _, body := range []string{
		locals.String() + "return a15;",
		"let unused=1;return " + strings.Repeat("x+", 31) + "x;",
		"let unused=1;return ((" + strings.Repeat("x+", 30) + "x));",
	} {
		checkClosureLocals(t, "module main;fn i32 main(){let c=fn [] i32(i32 x){"+body+"};return c(1);}")
	}
	for name, body := range map[string]string{
		"seventeen":    locals.String() + "let a16=a15;return a16;",
		"sixty five":   "let unused=1;let other=1;return " + strings.Repeat("x+", 31) + "x;",
		"group budget": "let unused=1;return (" + strings.Repeat("x+", 31) + "x);",
		"bool":         "let unused=true;return x;",
		"float":        "let unused=1.5;return x;",
		"string":       "let unused=\"text\";return x;",
	} {
		t.Run(name, func(t *testing.T) {
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;fn i32 main(){let c=fn [] i32(i32 x){"+body+"};return c(1);}")
			span := file.Module.Functions[0].Body.Statements[0].Span
			result, err := compiler.CheckExplicitFiles([]semantic.File{file})
			if err != nil || result.Verified || len(result.Semantic.Unsupported) != 1 || len(result.Semantic.Diagnostics) != 0 {
				t.Fatalf("bounded admission refusal: %v %+v", err, result.Semantic)
			}
			got := result.Semantic.Unsupported[0].Span
			if got.Source != "main.fy" || got.StartByte != span.StartByte || got.EndByte != span.EndByte {
				t.Fatalf("original binding span: %v want %v", got, span)
			}
		})
	}
}

func TestCopiedClosureLocalsUnusedTraps(t *testing.T) {
	for _, item := range []struct{ name, body, calls, trap string }{
		{"creation only", "let unused=10/x;return 42;", "return 42;", ""},
		{"later call", "let unused=10/x;return 42;", "assert(c(1)==42);return c(0);", "division-by-zero"},
		{"overflow first", "let first=2147483647+x;let second=10/0;return 42;", "return c(1);", "integer-overflow"},
		{"division first", "let first=10/0;let second=2147483647+x;return 42;", "return c(1);", "division-by-zero"},
	} {
		t.Run(item.name, func(t *testing.T) {
			result := checkClosureLocals(t, "module main;fn i32 main(){let c=fn [] i32(i32 x){"+item.body+"};"+item.calls+"}")
			out, err := interpreter.Execute(result.MIR, "main::main")
			if err != nil || item.trap == "" && (out.Kind != interpreter.OutcomeReturn || out.Value != 42) || item.trap != "" && (out.Kind != interpreter.OutcomeTrap || out.TrapCategory != item.trap) {
				t.Fatalf("initializer timing/order: %+v %v", out, err)
			}
		})
	}
}

func TestCopiedClosureLocalsIntegerRangeStaysTyped(t *testing.T) {
	for _, literal := range []string{"2147483648", strings.Repeat("9", 80)} {
		text := "module main;fn i32 main(){let c=fn [] i32(){let unused=" + literal + ";return 42;};return c();}"
		result, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
		if err != nil || result.Verified || len(result.Semantic.Diagnostics) == 0 || len(result.Semantic.Unsupported) != 0 {
			t.Fatalf("ordinary contextual i32 range check: %v %+v", err, result.Semantic)
		}
	}
}

func TestCopiedClosureLocalsGenericIdentity(t *testing.T) {
	var prior []compiler.GenericAnalysis
	for _, body := range []string{
		"let first=1;let second=2;return x;",
		"let first=2;let second=1;return x;",
		"let first=1;let second=3;return x;",
	} {
		text := "module main;pub fn T keep<T>(T value){let c=fn [] i32(i32 x){" + body + "};assert(c(2)==2);return value;}fn i32 main(){return keep<i32>(42);}"
		file := parseCompilerFile(t, 1, "main.fy", text)
		before, _ := json.Marshal(file.Module)
		analysis, err := compiler.AnalyzeGenericPackages(semantic.PackageInput{Files: []semantic.File{file}}, map[hir.PackageID]packagegraph.CoordinateID{1: genericCoordinate(t)}, genericplan.DefaultPlanLimits())
		if err != nil || len(analysis.Interfaces) != 1 || len(analysis.Plan.Specializations) != 1 {
			t.Fatalf("generic local identity: %v %+v", err, analysis)
		}
		if len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
			t.Fatalf("generic local analysis: %+v", analysis.Analysis.Semantic)
		}
		for _, old := range prior {
			if analysis.Interfaces[1] == old.Interfaces[1] || analysis.Plan.Specializations[0].ID == old.Plan.Specializations[0].ID {
				t.Fatal("unused initializer value/order missing from generic identity")
			}
		}
		after, _ := json.Marshal(file.Module)
		if !bytes.Equal(before, after) {
			t.Fatal("generic expansion mutated original closure locals")
		}
		checkClosureLocals(t, text)
		prior = append(prior, analysis)
	}
}
