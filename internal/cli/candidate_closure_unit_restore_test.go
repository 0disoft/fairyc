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

func candidateClosureUnitRestoreWorkspace(t *testing.T) (string, int) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_unit_restore.fy"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_unit_restore.fy", raw), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("closure unit restore fixture syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
	program := strings.Replace(string(raw), "module tests::closure_unit_restore;", "module main;", 1)
	program += "\n@effects(console) fn i32 main(){scalar_closed();\n"
	count := 0
	for _, tc := range candidateClosureUnitValueControls(t) {
		if !tc.accept {
			continue
		}
		program += fmt.Sprintf("verify(b%s,b%s);\n", strconv.Quote(tc.name), strconv.Quote(tc.raw))
		count++
	}
	// Declaration handles deliberately exceed the small canonical type arena.
	// A kind36 target is a helper declaration, never an entry in the type map.
	declarations := "module main;"
	for i := 0; i < 12; i++ {
		declarations += fmt.Sprintf("fn i32 pad%d(){return %d;}", i, i)
	}
	declarations += "fn fn i32(i32) make(){let fn i32(i32) f=fn [] i32(i32 x){return x+1;};return f;}fn i32 main(){let f=make();return f(41);}"
	for _, tc := range []struct{ name, raw string }{
		{"declaration-not-type-id", declarations},
		{"return-only-copy", "module main;fn fn i32(i32) keep(fn i32(i32) f){let copied=f;return copied;}"},
	} {
		program += fmt.Sprintf("verify(b%s,b%s);\n", strconv.Quote(tc.name), strconv.Quote(tc.raw))
		count++
	}
	base := "module main;fn i32 main(){let base=40;let fn i32(i32) f=fn [i32 base] i32(i32 x){return base+x+0;};return f(2);}"
	helper := strings.Replace(base, "base+x+0", "base+x+1", 1)
	invocation := strings.Replace(base, "f(2)", "f(3)", 1)
	metadata := strings.Replace(base, "fn i32 main()", "@effects(console) fn i32 main()", 1)
	for _, changed := range []string{helper, invocation, metadata} {
		program += fmt.Sprintf("mismatch(b%s,b%s);stale(b%s,b%s);\n", strconv.Quote(base), strconv.Quote(changed), strconv.Quote(base), strconv.Quote(changed))
	}
	program += "return 0;}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_unit_artifact_admission", "module_unit_artifact_identities", "module_project_artifact_bodies", "module_program_artifact_encode", "module_program_artifact_decode", "project_source_plan", "scanner", "syntax")
	writeTestSource(t, root, "src/main.fy", program)
	return root, count
}

func TestCandidateClosureUnitRestoreSource(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root, count := candidateClosureUnitRestoreWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure unit restore source: %d %s %s", code, out, diagnostic)
	}
	t.Logf("%d original-source closure controls, scalar refusal, three detached-valid source/helper/metadata mismatches and stale-input fixture source checked, not executed", count)
}

func TestCandidateClosureUnitRestoreNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 source-bound unit fixture; Linux remains separate")
	}
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root, count := candidateClosureUnitRestoreWorkspace(t)
	candidateClosureArtifactNativeFixture(t, root, count, "source-bound units, source/helper/metadata refusals")
}
