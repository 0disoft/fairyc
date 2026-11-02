package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCandidateStaticOperationDeclarations(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native proof; Linux deferred")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "static_operation_declarations.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::static_operation_declarations;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test", "") + "\nfn i32 main(){exact_providers_and_missing_tuple();malformed_operation_and_provider_reject();all_scalar_and_parameter_boundaries();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "static_operation_declarations", "scanner", "syntax")
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		code, out, diagnostics := runForTest(args, root)
		if code != ExitSuccess || out != "" || diagnostics != "" {
			t.Fatalf("%v: %d %q %s", args, code, out, diagnostics)
		}
	}
}
