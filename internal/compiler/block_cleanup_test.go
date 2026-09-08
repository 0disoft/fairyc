package compiler_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestBlockResultPrecedesLexicalCleanup(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;
fn i32 observe(i32 value) { return value; }
resource Mark { i32 id; cleanup { let i32 marked = observe(this.id); } }
fn Mark make(i32 id) { return Mark { .id = id }; }
fn Mark transfer() {
    return { let Mark kept = make(67); let Mark discarded = make(66); kept };
}
fn i32 main() {
    let i32 saved = { let Mark value = make(65); value.id };
    assert(saved == 65);
    let Mark kept = transfer();
    let i32 marked = observe(68);
    return 42;
}`)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("block cleanup compile: %v %+v", err, result.Semantic)
	}
	outcome, err := interpreter.ExecuteWithOptions(result.MIR, "main::main", interpreter.Options{
		TestIntrinsics: map[string]interpreter.TestIntrinsicKind{"main::observe": interpreter.TestIntrinsicWriteStdoutByte}, OutputLimit: 10,
	})
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || string(outcome.Stdout) != "ABDC" || len(outcome.DropTrace) != 3 {
		t.Fatalf("block result/cleanup order: %+v %v", outcome, err)
	}
}
