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

func TestCandidateStaticOperationCalls(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native proof; Linux deferred")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "static_operation_calls.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::static_operation_calls;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test", "") + "\n@effects(console) fn i32 main(){selected_ordinary_calls_keep_exact_types();provider_bodies_keep_ordinary_type_checks();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "module_signatures", "module_unit_bind", "module_unit_call_closure", "module_unit_values", "module_unit_value_model", "scanner", "syntax")
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		code, out, diagnostics := runForTest(args, root)
		if code != ExitSuccess || out != "" || diagnostics != "" {
			t.Fatalf("%v: %d %q %s", args, code, out, diagnostics)
		}
	}
}

func TestCandidateStaticOperationIdentity(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native proof; Linux deferred")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	oracle := candidateStaticOperationOracle(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "static_operation_plan_fixture.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::static_operation_plan_fixture;", "module main;", 1) + "\n@effects(console,filesystem) fn i32 main(){verify();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "project_source_plan", "project_diagnostic_codes", "generic_source_plan", "generic_source_plan_input", "generic_concrete_types", "generic_package_context", "generic_source_requests", "generic_source_dependencies", "generic_source_closure", "generic_source_dependency_model", "generic_source_dependency_json", "generic_source_catalog", "generic_package_bindings", "module_catalog", "module_public_type_text", "module_public_templates", "module_generic_interface", "syntax", "scanner")
	for _, args := range [][]string{{"run", "--", oracle}, {"run", "--release", "--locked", "--", oracle}} {
		code, out, diagnostics := runForTest(args, root)
		if code != ExitSuccess || out != "" || diagnostics != "" {
			t.Fatalf("%v: %d %q %s", args, code, out, diagnostics)
		}
	}
}
