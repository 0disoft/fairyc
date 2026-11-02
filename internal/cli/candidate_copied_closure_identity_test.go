package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCandidateCopiedClosureIdentity(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native proof; Linux deferred")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	oracle := candidateCopiedClosureOracle(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "copied_closure_plan_fixture.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::copied_closure_plan_fixture;", "module main;", 1) + "\n@effects(console,filesystem) fn i32 main(){verify();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "project_source_plan", "project_diagnostic_codes", "generic_source_plan", "generic_source_plan_input", "generic_concrete_types", "generic_package_context", "generic_source_requests", "generic_source_dependencies", "generic_source_closure", "generic_source_dependency_model", "generic_source_dependency_json", "generic_source_catalog", "generic_package_bindings", "module_catalog", "module_public_type_text", "module_public_templates", "module_generic_interface", "syntax", "scanner")
	for _, args := range [][]string{{"run", "--", oracle}, {"run", "--release", "--locked", "--", oracle}} {
		code, out, diagnostics := runForTest(args, root)
		if code != ExitSuccess || out != "" || diagnostics != "" {
			t.Fatalf("%v: %d %q %s", args, code, out, diagnostics)
		}
	}
}
