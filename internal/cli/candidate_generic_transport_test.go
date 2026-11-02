package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestCandidateGenericTransportAdmission(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64; Linux native checks deferred")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for native Fairy parity checks")
	}

	for _, fixture := range []struct {
		name, calls string
		modules     []string
	}{
		{"generic_scalar_arguments", `descriptors_preserve_names_and_materialize_direct_value_types();forwarded_arguments_keep_identity_and_value_substitution();invalid_argument_domains_stay_closed();composite_substitution_rejects_without_publishing_partial_types();`, []string{"generic_scalar_arguments", "generic_transport_scalars", "module_types", "module_catalog", "syntax", "scanner"}},
		{"generic_concrete_types", `retained_type_properties_and_admission_boundaries();`, []string{"generic_concrete_types", "generic_transport_scalars", "scanner", "syntax"}},
		{"generic_signature_diagnostics", `signature_reports_reject_stale_sources_and_exhausted_work();`, []string{"generic_signature_diagnostics", "project_source_plan", "module_public_type_text", "module_catalog", "syntax", "scanner"}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", fixture.name+".fy"))
			if err != nil {
				t.Fatal(err)
			}
			var input string
			if fixture.name == "generic_signature_diagnostics" {
				repo, err := filepath.Abs(filepath.Join("..", ".."))
				if err != nil {
					t.Fatal(err)
				}
				compiler := filepath.Join(root, "go-reference.exe")
				command := exec.CommandContext(t.Context(), "go", "build", "-mod=readonly", "-o", compiler, "./cmd/fairyc")
				command.Dir = repo
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("build Go oracle: %v %s", err, output)
				}
				directory := filepath.Join(root, "signature-oracle")
				command = exec.CommandContext(t.Context(), "python", "-B", "tools/verify-candidate-unsupported-signatures.py", "--compiler", compiler, "--output", directory)
				command.Dir = repo
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("signature oracle: %v %s", err, output)
				}
				input = filepath.Join(directory, "fixture.bin")
				fixture.calls += "signature_reports_match_go_original_spans_and_features();"
			}
			program := strings.Replace(string(raw), "module tests::"+fixture.name+";", "module main;", 1)
			effects := ""
			if input != "" {
				effects = "@effects(filesystem) "
			}
			program = strings.ReplaceAll(program, "@test", "") + "\n" + effects + "fn i32 main(){" + fixture.calls + "return 0;}\n"
			writeTestSource(t, root, "src/main.fy", program)
			parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(program)), source.DefaultLimits())
			if err != nil || parsed.AST == nil {
				t.Fatalf("parse parity fixture: %v", err)
			}
			for _, imported := range parsed.AST.Imports {
				fixture.modules = append(fixture.modules, imported.Path)
			}
			copyCandidateParityModules(t, root, fixture.modules...)
			for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
				if input != "" {
					args = append(args, "--", input)
				}
				code, out, diagnostics := runForTest(args, root)
				if code != ExitSuccess || out != "" || diagnostics != "" {
					t.Fatalf("%v: %d %q %s", args, code, out, diagnostics)
				}
			}
		})
	}
}
