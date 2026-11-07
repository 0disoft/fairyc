package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

type closureUnitValueCase struct {
	name, raw                   string
	accept, conditional         bool
	constructs, calls, captures int
}

func candidateClosureUnitValueCases() []closureUnitValueCase {
	cases := []closureUnitValueCase{}
	for captures := 0; captures <= 4; captures++ {
		locals, names, result := "", []string{}, "x"
		for i := 0; i < captures; i++ {
			name := string(rune('a' + i))
			locals += fmt.Sprintf("let %s=%d;", name, i+1)
			names = append(names, "i32 "+name)
			result += "+" + name
		}
		raw := "module main;fn i32 main(){" + locals + "let fn i32(i32) f=fn [" + strings.Join(names, ",") + "] i32(i32 x){return " + result + ";};return f(2);}"
		cases = append(cases, closureUnitValueCase{fmt.Sprintf("captures-%d-inferred-snapshots", captures), raw, true, false, 1, 1, captures})
	}
	cases = append(cases,
		closureUnitValueCase{"factory-copy-direct-struct", `module main;struct Holder{fn i32(i32) f;}fn fn i32(i32) make(i32 base){let fn i32(i32) f=fn [i32 base] i32(i32 x){return base+x;};return f;}fn fn i32(i32) keep(fn i32(i32) f){let copied=f;return copied;}fn i32 main(){let original=make(40);let copy=keep(original);let holder=Holder{.f=copy};let stored=holder;let extracted=stored.f;return extracted(2);}`, true, false, 1, 1, 1},
		closureUnitValueCase{"different-helpers-one-parameter", `module main;fn fn i32(i32) first(){let base=40;let fn i32(i32) f=fn [i32 base] i32(i32 x){return base+x;};return f;}fn fn i32(i32) second(){let fn i32(i32) f=fn [] i32(i32 x){return x*2;};return f;}fn i32 apply(fn i32(i32) f,i32 x){return f(x);}fn i32 main(){let a=first();let b=second();return apply(a,2)+apply(b,21);}`, true, false, 2, 1, 1},
		closureUnitValueCase{"signature-only-void-consumer", `module main;fn void consume(fn i32(i32) f){discard f(2);return;}`, true, false, 0, 1, 0},
		closureUnitValueCase{"snapshot-from-mutable-i32", `module main;fn i32 main(){var i32 base=40;let fn i32(i32) f=fn [i32 base] i32(i32 x){return base+x;};base=0;return f(2);}`, true, false, 1, 1, 1},
		closureUnitValueCase{"shadow-unsafe-provider", `module main;unsafe fn i32 f(i32 x){return x;}fn i32 main(){let fn i32(i32) f=fn [] i32(i32 x){return x;};return f(2);}`, true, true, 1, 1, 0},
		closureUnitValueCase{"shadow-effectful-provider", `module main;@effects(console) fn i32 f(i32 x){return x;}fn i32 main(){let fn i32(i32) f=fn [] i32(i32 x){return x;};return f(2);}`, true, true, 1, 1, 0},
	)
	base := `module main;fn i32 main(){let fn i32(i32) f=fn [] i32(i32 x){return x;};`
	for _, rejected := range []struct{ name, raw string }{
		{"actual-effectful-provider", `module main;@effects(console) fn i32 f(i32 x){return x;}fn i32 main(){return f(2);}`},
		{"unhandled-must-use-provider", `module main;@must_use fn i32 f(i32 x){return x;}fn i32 main(){f(2);return 0;}`},
		{"wrong-callable", `module main;fn i32 main(){let f=1;return f(2);}`},
		{"zero-args", base + `return f();}`},
		{"two-args", base + `return f(1,2);}`},
		{"bool-arg", base + `return f(true);}`},
		{"wide-arg", base + `let i64 x=2;return f(x);}`},
		{"labeled-arg", base + `return f(.x=2);}`},
		{"mutable-closure", `module main;fn i32 main(){var fn i32(i32) f=fn [] i32(i32 x){return x;};return f(2);}`},
		{"mutable-copy", base + `var copy=f;return copy(2);}`},
		{"wrong-capture-type", `module main;fn i32 main(){let bool base=true;let fn i32(i32) f=fn [i32 base] i32(i32 x){return base+x;};return f(2);}`},
		{"unsupported-signature", `module main;fn i64 apply(fn i64(i64) f){return f(2);}`},
		{"option-wrapper", `module main;fn void consume(Option<fn i32(i32)> f){return;}`},
		{"result-wrapper", `module main;fn void consume(Result<fn i32(i32),u32> f){return;}`},
		{"array-wrapper", `module main;struct Holder{(fn i32(i32))[2] f;}`},
		{"nested-carrier", `module main;struct Holder{fn i32(i32) f;}struct Box{Holder h;}fn Box keep(Box value){return value;}`},
		{"repr-c", `module main;@repr(C) struct Holder{fn i32(i32) f;}`},
		{"ffi", `module main;unsafe extern C fn void consume(fn i32(i32) f);`},
	} {
		cases = append(cases, closureUnitValueCase{name: rejected.name, raw: rejected.raw})
	}
	return cases
}

// These controls parse original source and run the Go frontend/compiler only.
// The Fairy producer below independently stops at unit values, before MIR.
func candidateClosureUnitValueControls(t *testing.T) []closureUnitValueCase {
	t.Helper()
	var cases []closureUnitValueCase
	for _, tc := range candidateClosureUnitValueCases() {
		p, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(tc.raw)), source.DefaultLimits())
		if err != nil || p.AST == nil || !p.Tree.ErrorFree() {
			t.Fatalf("Go original syntax %s: %v %+v", tc.name, err, p.Tree.Diagnostics)
		}
		checked, err := compiler.CheckExplicitFiles([]semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: p.AST}})
		accepted := err == nil && checked.Verified
		if tc.conditional && !accepted {
			t.Logf("Go does not admit optional %s: err=%v diagnostics=%+v unsupported=%+v", tc.name, err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			continue
		}
		if accepted != tc.accept {
			t.Errorf("Go original %s accepted=%v want=%v err=%v diagnostics=%+v unsupported=%+v", tc.name, accepted, tc.accept, err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
		}
		cases = append(cases, tc)
	}
	if t.Failed() {
		t.Fatal("original-source Go controls failed; refusing a partial Fairy fixture")
	}
	return cases
}

func candidateClosureUnitValuesFixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_unit_values.fy"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_unit_values.fy", raw), source.DefaultLimits())
	if err != nil || p.AST == nil || !p.Tree.ErrorFree() {
		t.Fatalf("unit values fixture syntax: %v %+v", err, p.Tree.Diagnostics)
	}
	return strings.Replace(string(raw), "module tests::closure_unit_values;", "module main;", 1)
}

func candidateClosureUnitValuesWorkspace(t *testing.T) (string, []closureUnitValueCase) {
	t.Helper()
	cases := candidateClosureUnitValueControls(t)
	main := candidateClosureUnitValuesFixture(t) + "\n@effects(console) fn i32 main(){\n"
	for _, tc := range cases {
		main += fmt.Sprintf("verify(b%s,b%s,%t,%d,%d,%d);\n", strconv.Quote(tc.name), strconv.Quote(tc.raw), tc.accept, tc.constructs, tc.calls, tc.captures)
	}
	main += "return 0;}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	writeTestSource(t, root, "src/main.fy", main)
	copyCandidateParityModules(t, root, "module_unit_values", "module_unit_call_closure", "module_unit_bind", "module_signatures", "copied_closure_tables", "syntax", "scanner")
	return root, cases
}

func TestCandidateClosureUnitValuesControls(t *testing.T) {
	cases := candidateClosureUnitValueControls(t)
	candidateClosureUnitValuesFixture(t)
	t.Logf("%d original-source Go controls and Fairy fixture syntax", len(cases))
}

func TestCandidateClosureUnitValuesSource(t *testing.T) {
	root, _ := candidateClosureUnitValuesWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure unit values source check: %d %s %s", code, out, diagnostic)
	}
}

// Native runs require parent CPU coordination; no temporary environment gate.
// This executes the source-to-values fixture, not the user's closure program.
func TestCandidateClosureUnitValuesNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("closure unit values native probe requires Windows AMD64; Linux remains batched")
	}
	root, cases := candidateClosureUnitValuesWorkspace(t)
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		code, out, diagnostic := runForTest(args, root)
		if code != ExitSuccess || diagnostic != "" {
			t.Fatalf("closure unit values native %v: %d\nstdout: %s\nstderr: %s", args, code, out, diagnostic)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != len(cases) {
			t.Fatalf("native receipts=%d want=%d: %s", len(lines), len(cases), out)
		}
		for i, tc := range cases {
			if !strings.HasPrefix(lines[i], "closure-unit-values case="+tc.name+" ") || !strings.Contains(lines[i], " args=") || !strings.Contains(lines[i], " helpers=") {
				t.Fatalf("missing ordered receipt for %s: %q", tc.name, lines[i])
			}
		}
	}
}
