package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestInlineMutationVersionsAndNative(t *testing.T) {
	const declaration = `@inline pub fn i32 step(i32 input){var i32 current=input;let before=current;current=current+2;let after=current;current=before+after;return current;}`
	text := "module main;" + declaration + "fn i32 main(){return step(20);}"
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
	if err != nil || !result.Verified {
		t.Fatalf("mutation: %v %+v", err, result.Semantic)
	}
	value, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || value.Value != 42 {
		t.Fatalf("versions lost: %+v %v", value, err)
	}
	runExpandedLanguageNative(t, result, "main::main", "")
	wire, err := mircodec.Encode(result.MIR, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := mircodec.Decode(wire, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	value, err = interpreter.Execute(restored, "main::main")
	if err != nil || value.Value != 42 {
		t.Fatalf("restored versions: %+v %v", value, err)
	}
	changed := strings.Replace(text, "current=current+2", "current=current+3", 1)
	second, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", changed)})
	if err != nil || !second.Verified || inlineMutationMarker(second) == inlineMutationMarker(result) {
		t.Fatalf("assignment did not affect identity: %v", err)
	}
}

func TestInlineMutationUnusedAssignmentTraps(t *testing.T) {
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;@inline pub fn i32 step(){var i32 value=2147483647;value=value+1;return 42;}fn i32 main(){return step();}`)})
	if err != nil || !result.Verified {
		t.Fatalf("check: %v %+v", err, result.Semantic)
	}
	value, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || value.Kind != interpreter.OutcomeTrap || value.TrapCategory != "integer-overflow" {
		t.Fatalf("assignment trap lost: %+v %v", value, err)
	}
}

func inlineMutationMarker(value compiler.Result) string {
	for _, fn := range value.MIR.Functions {
		if fn.InlineBody != "" {
			return fn.InlineBody
		}
	}
	return ""
}
