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

func candidateClosureMIRFixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "mir_closure_values.fy"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestCandidateClosureValueMIRSyntax(t *testing.T) {
	raw := candidateClosureMIRFixture(t)
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/mir_closure_values.fy", raw), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("closure MIR fixture syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
}

func candidateClosureMIRWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyLexerExample(t, root)
	program := strings.Replace(string(candidateClosureMIRFixture(t)), "module tests::mir_closure_values;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test", "") + "\nfn i32 main(){closure_mir_checks_targets_captures_and_initialization();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "mir_model", "mir_state", "mir_verify", "mir_closure_values")
	if code, out, diagnostics := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostics != "" {
		t.Fatalf("focused closure MIR verifier source check: %d %q %s", code, out, diagnostics)
	} else {
		t.Log(strings.TrimSpace(out))
	}
	return root
}

func TestCandidateClosureValueMIRSourceCheck(t *testing.T) {
	candidateClosureMIRWorkspace(t)
}

func TestCandidateClosureValueMIR(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("focused Windows AMD64 MIR verifier proof; Linux deferred")
	}
	root := candidateClosureMIRWorkspace(t)
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		if code, out, diagnostics := runForTest(args, root); code != ExitSuccess || out != "" || diagnostics != "" {
			t.Fatalf("focused closure MIR verifier %v: %d %q %s", args, code, out, diagnostics)
		}
	}
}
