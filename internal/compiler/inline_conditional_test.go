package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/inlinebody"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestInlineConditionalIdentityAndExecution(t *testing.T) {
	const declaration = `@inline pub fn i32 choose(i32 x, i32 divisor) {
    return if (x < 0) { 42 / divisor } else if (x == 0) { 41 } else { x + 1 };
}`
	const main = `fn i32 main() {
    assert(choose(-1, 2) == 21);
    assert(choose(0, 0) == 41);
    return choose(41, 0);
}`
	check := func(source string) compiler.Result {
		t.Helper()
		return checkInlineCalls(t, "module main;"+source+main)
	}
	result := check(declaration)
	marker := inlineMutationMarker(result)
	if marker == "" || inlineMutationMarker(check(strings.ReplaceAll(declaration, "x", "input"))) != marker {
		t.Fatal("parameter rename changed the inline identity")
	}
	for _, edit := range [][2]string{{"x < 0", "x <= -1"}, {"42 / divisor", "44 / divisor"}, {"{ 41 }", "{ 40 }"}, {"x + 1", "x + 2"}} {
		if inlineMutationMarker(check(strings.Replace(declaration, edit[0], edit[1], 1))) == marker {
			t.Fatalf("conditional edit %v kept the old inline identity", edit)
		}
	}
	wire, err := mircodec.Encode(result.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := mircodec.Decode(wire, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := inlinebody.VerifyProgram(result.Semantic.HIR, restored); err != nil {
		t.Fatal(err)
	}
	value, err := interpreter.Execute(restored, "main::main")
	if err != nil || value.Kind != interpreter.OutcomeReturn || value.Value != 42 {
		t.Fatalf("restored conditional: %+v %v", value, err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		if _, err := cbackend.Emit(restored, "main::main", profile); err != nil {
			t.Fatal(err)
		}
	}
	runExpandedLanguageNative(t, result, "main::main", "")
}

func TestInlineConditionalAdmissionBoundaries(t *testing.T) {
	for _, body := range []string{
		"return if ((x == 0) == true) { 42 } else { 0 };",
		"return if (x == 0) { var i32 local = 42; local } else { 0 };",
		"var i32 y = 0; return if (x == 0) { y = 42; y } else { 0 };",
		"if (x == 0) { while (true) { return 42; } } return 0;",
	} {
		t.Run(body, func(t *testing.T) {
			input := "module main; @inline pub fn i32 choose(i32 x) {" + body + "} fn i32 main() { return choose(0); }"
			result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", input)})
			if err != nil || result.Verified || len(result.Semantic.Unsupported) == 0 {
				t.Fatalf("expected unsupported inline body: %v %+v", err, result.Semantic)
			}
		})
	}
}

func TestInlineStatementBranches(t *testing.T) {
	const body = `var i32 result = 0;
    if (x == 0) { return 40; }
    if (x < 0) { result = 2; }
    else if (x == 1) { let i32 local = 1; result = local + 1; }
    else { result = 10 / x; }
    return result;`
	const source = `
@inline pub fn i32 finish(i32 x) {
    if (x == 0) { return choose(0); } else { return choose(-1); };
}
fn i32 main() { assert(choose(1) == 2); return finish(0) + finish(2); }`
	check := func(body string) compiler.Result {
		t.Helper()
		return checkInlineCalls(t, `module main; @inline pub fn i32 choose(i32 x) {`+body+`}`+source)
	}
	result := check(body)
	marker := inlineMutationMarker(result)
	for _, edit := range [][2]string{{"x < 0", "x < -1"}, {"result = 2", "result = 3"}, {"return 40", "return 39"}} {
		if inlineMutationMarker(check(strings.Replace(body, edit[0], edit[1], 1))) == marker {
			t.Fatalf("statement branch edit %v did not change its identity", edit)
		}
	}
	if marker == "" || inlineMutationMarker(check(strings.ReplaceAll(body, "result", "renamed"))) != marker {
		t.Fatal("renaming a lexical slot changed its identity")
	}
	wire, err := mircodec.Encode(result.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := mircodec.Decode(wire, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := inlinebody.VerifyProgram(result.Semantic.HIR, restored); err != nil {
		t.Fatal(err)
	}
	value, err := interpreter.Execute(restored, "main::main")
	if err != nil || value.Kind != interpreter.OutcomeReturn || value.Value != 42 {
		t.Fatalf("restored statement branches: %+v %v", value, err)
	}
	runExpandedLanguageNative(t, result, "main::main", "")
}

func TestInlineConditionalArmBindings(t *testing.T) {
	const body = `let i32 seed = 41;
    let i32 selected = if (x == 0) {
        let i32 branch_value = seed + 1;
        if (branch_value == 42) { let i32 nested = branch_value; nested } else { 0 }
    } else { let i32 branch_value = 10 / x; branch_value };
    return selected;`
	check := func(body string) compiler.Result {
		t.Helper()
		return checkInlineCalls(t, `module main; @inline pub fn i32 choose(i32 x) {`+body+`}
fn i32 main() { assert(choose(2) == 5); return choose(0); }`)
	}
	result := check(body)
	marker := inlineMutationMarker(result)
	if marker == "" || inlineMutationMarker(check(strings.ReplaceAll(body, "branch_value", "renamed"))) != marker {
		t.Fatal("renaming arm locals changed the inline identity")
	}
	if inlineMutationMarker(check(strings.Replace(body, "10 / x", "12 / x", 1))) == marker {
		t.Fatal("editing the alternate initializer kept the old inline identity")
	}
	value, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || value.Kind != interpreter.OutcomeReturn || value.Value != 42 {
		t.Fatalf("arm binding evaluation: %+v %v", value, err)
	}
	runExpandedLanguageNative(t, result, "main::main", "")
}

func TestInlineConditionalShortCircuit(t *testing.T) {
	const body = `return if (x != 0 && 42 / x == 21) { 7 }
        else if (!(x != 0) || 42 / x == 1) { 42 } else { 0 };`
	const main = `fn i32 main() { assert(choose(2) == 7); return choose(0); }`
	check := func(expression string) compiler.Result {
		t.Helper()
		return checkInlineCalls(t, "module main; @inline pub fn i32 choose(i32 x) {"+expression+"}"+main)
	}
	result := check(body)
	for _, edit := range [][2]string{{"&&", "||"}, {"||", "&&"}, {"!(x != 0)", "(x != 0)"}} {
		if inlineMutationMarker(check(strings.Replace(body, edit[0], edit[1], 1))) == inlineMutationMarker(result) {
			t.Fatalf("logical edit %v kept the old inline identity", edit)
		}
	}
	value, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || value.Kind != interpreter.OutcomeReturn || value.Value != 42 {
		t.Fatalf("short-circuit evaluation: %+v %v", value, err)
	}
	runExpandedLanguageNative(t, result, "main::main", "")
}
