package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var lexerCandidateSourceFiles = []string{
	"fairy.toml", "src/scanner.fy", "src/candidate_limits.fy", "src/numbers.fy",
	"src/diagnostics.fy", "src/utf8.fy", "src/strings.fy", "src/budget.fy",
}

func copyLexerExample(t *testing.T, root string) {
	t.Helper()
	for _, name := range lexerCandidateSourceFiles {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, name, string(data))
	}
}

func TestLexerCandidateFixtureWorkspacesCheck(t *testing.T) {
	for _, fixture := range []struct {
		name string
		copy func(*testing.T, string)
	}{
		{"lexer", copyLexerExample},
		{"parser", copyLexerStreamExample},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			root := t.TempDir()
			fixture.copy(t, root)
			writeTestSource(t, root, "src/main.fy", "module main; import scanner as scanner; fn i32 main() { return 0; }\n")
			code, stdout, stderr := runForTest([]string{"check"}, root)
			if code != ExitSuccess || !strings.HasPrefix(stdout, "check ok:") || stderr != "" {
				t.Fatalf("candidate fixture check: code=%d stdout=%q stderr=%s", code, stdout, stderr)
			}
		})
	}
}
