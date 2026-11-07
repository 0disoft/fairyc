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

func candidateClosureUnitBodyWorkspace(t *testing.T) (string, int) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_unit_body.fy"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_unit_body.fy", raw), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("closure unit body fixture syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
	program := strings.Replace(string(raw), "module tests::closure_unit_body;", "module main;", 1)
	program += "\n@effects(console) fn i32 main(){\n"
	count := 0
	for _, tc := range candidateClosureUnitValueControls(t) {
		if !tc.accept {
			continue
		}
		program += fmt.Sprintf("verify(b%s,b%s,%d,%d,%d);\n", strconv.Quote(tc.name), strconv.Quote(tc.raw), tc.constructs, tc.calls, tc.captures)
		count++
	}
	program += "return 0;}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "module_unit_body_ir_lower", "module_unit_body_ir_closures", "module_unit_cfg_lower", "module_body_artifact_encode", "module_cleanup_artifact_encode", "scanner", "syntax")
	return root, count
}

func TestCandidateClosureUnitBodySource(t *testing.T) {
	root, count := candidateClosureUnitBodyWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure unit body source: %d %s %s", code, out, diagnostic)
	}
	t.Logf("%d original-source closure controls, body/CFG and forged-record fixture checked", count)
}

func TestCandidateClosureUnitBodyNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("closure unit body native probe requires Windows AMD64; Linux remains batched")
	}
	root, count := candidateClosureUnitBodyWorkspace(t)
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		if code, out, diagnostic := runForTest(args, root); code != ExitSuccess || out != "" || diagnostic != "" {
			t.Fatalf("closure unit body native %v: %d %q %s", args, code, out, diagnostic)
		}
		t.Logf("%v: %d original-source controls, body/CFG proof and applicable forged-record refusals", args, count)
	}
}
