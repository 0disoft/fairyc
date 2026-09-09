package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestFieldBorrowAssignmentRejectsOnlyExecutableLowering(t *testing.T) {
	const text = `module main;
struct Value { i32 number; }
struct Counter { Value value; }
fn void Counter::set(ref mut Counter this, Value value) { this.value = value; }
fn i32 main() { var Counter counter = Counter { .value = Value { .number = 0 } }; counter.set(Value { .number = 42 }); return 0; }`
	files := []semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)}
	analysis := semantic.Analyze(files)
	if analysis.CompletedThrough != semantic.PhaseFlow || len(analysis.Diagnostics) != 0 || len(analysis.Unsupported) != 0 {
		t.Fatalf("semantic admission: %+v", analysis)
	}
	for _, lower := range []func() (compiler.Result, error){
		func() (compiler.Result, error) { return compiler.Check(files) },
		func() (compiler.Result, error) { return compiler.LowerAnalyzed(compiler.Analysis{Semantic: analysis}) },
	} {
		result, err := lower()
		if err != nil || result.Verified || len(result.MIR.Functions) != 0 || len(result.Semantic.Unsupported) != 1 {
			t.Fatalf("executable admission: %+v %v", result, err)
		}
		finding := result.Semantic.Unsupported[0]
		if finding.Feature != "field assignment without executable field storage" || finding.Span.Source != "main.fy" || finding.Span.StartByte != strings.Index(text, "this.value") {
			t.Fatalf("source location: %+v", finding)
		}
	}
	if len(analysis.Unsupported) != 0 {
		t.Fatal("lowering changed its semantic input")
	}
}
