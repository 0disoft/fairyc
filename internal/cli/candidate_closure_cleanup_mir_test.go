package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func candidateClosureCleanupMIRSource(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_cleanup_mir.fy")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_cleanup_mir.fy", raw), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("closure cleanup fixture syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
	program := strings.Replace(string(raw), "module tests::closure_cleanup_mir;", "module main;", 1)
	return strings.ReplaceAll(program, "@test", "") + `
fn i32 main(){
 closure_cleanup_leaf_is_explicit_trivial_and_modes_are_independent();
 closure_cleanup_rejects_payloads_duplicates_and_wrappers();
 closure_cleanup_has_no_drop_or_resource_obligation();
 closure_cleanup_keeps_old_artifact_decoders_closed();
 closure_cleanup_preserves_work_and_record_bounds();
 return 0;
}
`
}

func candidateClosureCleanupMIRWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyLexerExample(t, root)
	writeTestSource(t, root, "src/main.fy", candidateClosureCleanupMIRSource(t))
	copyCandidateParityModules(t, root, "module_unit_cleanup_mir_model", "module_unit_cleanup_mir_verify", "module_cleanup_artifact_decode")
	return root
}

// Syntax/import and real source checking only: safe to run while a parent owns
// native compiler CPU. This never builds or executes a native Fairy program.
func TestCandidateClosureCleanupMIRSource(t *testing.T) {
	root := candidateClosureCleanupMIRWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure cleanup source check: %d %s %s", code, out, diagnostic)
	}
}

// Run separately after parent CPU coordination. The old graph executable_code
// regression remains unchanged; this fixture selects the new private family.
func TestCandidateClosureCleanupMIRNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("small native cleanup probe requires Windows AMD64; Linux remains batched")
	}
	root := candidateClosureCleanupMIRWorkspace(t)
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		if code, out, diagnostic := runForTest(args, root); code != ExitSuccess || out != "" || diagnostic != "" {
			t.Fatalf("closure cleanup native %v: %d %q %s", args, code, out, diagnostic)
		}
		t.Logf("closure cleanup native %v: exit 0, empty stdout/stderr", args)
	}
}
