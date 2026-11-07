package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

type closureSourcePolicyCase struct {
	name, raw string
	admitted  bool
}

func closureSourcePolicyCases() []closureSourcePolicyCase {
	const literal = "let fn i32(i32) action=fn [] i32(i32 x){return x;};"
	return []closureSourcePolicyCase{
		{"closed-scalar-closure-forward", `module main;fn fn i32(i32) forward<T>(fn i32(i32) action,T marker){discard marker;return action;}fn i32 main(){` + literal + `let copy=forward<i32>(action,1);return copy(42);}`, false},
		{"closed-scalar-holder-forward", `module main;struct Holder{fn i32(i32) action;}fn Holder forward<T>(Holder value,T marker){discard marker;return value;}fn i32 main(){` + literal + `let holder=Holder{.action=action};let copy=forward<i32>(holder,1);let extracted=copy.action;return extracted(42);}`, false},
		{"closed-scalar-holder-parameter", `module main;struct Holder{fn i32(i32) action;}fn i32 invoke<T>(Holder value,T marker){discard marker;let extracted=value.action;return extracted(42);}fn i32 main(){` + literal + `let holder=Holder{.action=action};return invoke<i32>(holder,1);}`, false},
		{"holder-generic-identity", `module main;struct Holder{fn i32(i32) action;}fn T identity<T>(T value){return value;}fn i32 main(){` + literal + `let holder=Holder{.action=action};let copy=identity<Holder>(holder);let extracted=copy.action;return extracted(42);}`, false},
		{"unused-generic-closure-parameter", `module main;fn void unused<T>(fn i32(i32) action,T marker){discard marker;discard action;}fn i32 main(){return 42;}`, true},
		{"unused-generic-closure-result", `module main;fn fn i32(i32) unused<T>(fn i32(i32) action,T marker){discard marker;return action;}fn i32 main(){return 42;}`, true},
		{"unused-generic-holder-parameter", `module main;struct Holder{fn i32(i32) action;}fn void unused<T>(Holder value,T marker){discard marker;discard value;}fn i32 main(){return 42;}`, true},
		{"scalar-generic-near-closure", `module main;fn T identity<T>(T value){return value;}fn i32 main(){` + literal + `return identity<i32>(action(42));}`, true},
		{"unused-scalar-generic-near-closure", `module main;fn T identity<T>(T value){return value;}fn i32 main(){` + literal + `return action(42);}`, true},
		{"scalar-type-parameter-shadows-holder", `module main;struct Holder{fn i32(i32) action;}fn Holder identity<Holder>(Holder value){return value;}fn i32 main(){` + literal + `return identity<i32>(action(42));}`, true},
		{"scalar-struct-generic-near-closure", `module main;struct Scalar{i32 value;}fn T identity<T>(T value){return value;}fn i32 main(){` + literal + `let scalar=Scalar{.value=42};let copied=identity<Scalar>(scalar);return action(copied.value);}`, false},
	}
}
func closureSourcePolicyOracle(t *testing.T) []bool {
	t.Helper()
	var admitted []bool
	for _, tc := range closureSourcePolicyCases() {
		root := copiedClosureLocalsWorkspace(t, tc.raw)
		code, out, diagnostic := runForTest([]string{"check"}, root)
		accepted := code == ExitSuccess && diagnostic == "" && strings.HasPrefix(out, "check ok:")
		if !accepted && (code != ExitSourceDiagnostics || diagnostic == "") {
			t.Fatalf("Go reference %s: %d %q %s", tc.name, code, out, diagnostic)
		}
		if accepted != tc.admitted {
			t.Fatalf("Go canonical policy changed for %s: admitted=%t want=%t exit=%d diagnostic=%s", tc.name, accepted, tc.admitted, code, diagnostic)
		}
		t.Logf("Go CLI %s: admitted=%t exit=%d diagnostic=%s", tc.name, accepted, code, strings.TrimSpace(diagnostic))
		admitted = append(admitted, accepted)
	}
	return admitted
}
func closureSourcePolicyProgram(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_value_source_policy.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::closure_value_source_policy;", "module main;", 1)
	program += "\n@effects(console) fn i32 main(){"
	for _, tc := range closureSourcePolicyCases() {
		program += "probe(b" + strconv.Quote(tc.name) + ",b" + strconv.Quote(tc.raw) + ");\n"
	}
	return program + "return 0;}\n"
}
func closureSourcePolicyWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyLexerExample(t, root)
	writeTestSource(t, root, "src/main.fy", closureSourcePolicyProgram(t))
	copyCandidateParityModules(t, root, "scanner", "syntax", "name_resolution", "type_check", "hir_lower", "mir_lower", "mir_model", "mir_verify")
	return root
}
func TestCandidateClosureValueSourcePolicySource(t *testing.T) { closureSourcePolicyOracle(t) }
func TestCandidateClosureValueSourcePolicySyntax(t *testing.T) {
	for _, tc := range append(closureSourcePolicyCases(), closureSourcePolicyCase{"policy-probe", closureSourcePolicyProgram(t), true}) {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(tc.raw)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("%s syntax: %v %+v", tc.name, err, parsed.Tree.Diagnostics)
		}
	}
}
func TestCandidateClosureValueSourcePolicyCheck(t *testing.T) {
	root := closureSourcePolicyWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("bounded original-source policy probe check: %d %s %s", code, out, diagnostic)
	} else {
		t.Log(strings.TrimSpace(out))
	}
}
func TestCandidateClosureValueSourcePolicyNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 original-source pipeline probe; Linux batched")
	}
	oracle := closureSourcePolicyOracle(t)
	root := closureSourcePolicyWorkspace(t)
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		code, out, diagnostic := runForTest(args, root)
		if code != ExitSuccess || diagnostic != "" {
			t.Fatalf("Fairy source pipeline %v: %d %q %s", args, code, out, diagnostic)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != len(oracle) {
			t.Fatalf("Fairy source policy reports=%d want=%d: %q", len(lines), len(oracle), out)
		}
		for i, tc := range closureSourcePolicyCases() {
			var stage, status uint32
			var start, end uint64
			if _, err := fmt.Sscanf(lines[i], "%d %d %d %d", &stage, &status, &start, &end); err != nil {
				t.Fatalf("report %s %q: %v", tc.name, lines[i], err)
			}
			accepted := stage == 9 && status == 0
			t.Logf("Fairy %v %s: admitted=%t phase=%d code=%d span=%d..%d Go-admitted=%t", args, tc.name, accepted, stage, status, start, end, oracle[i])
			if accepted != oracle[i] {
				t.Errorf("original-source policy differs for %s: Fairy=%t phase=%d code=%d span=%d..%d Go=%t", tc.name, accepted, stage, status, start, end, oracle[i])
			}
		}
	}
}
