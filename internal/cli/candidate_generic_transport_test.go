package cli

import (
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
			program := strings.Replace(string(raw), "module tests::"+fixture.name+";", "module main;", 1)
			program = strings.ReplaceAll(program, "@test", "") + "\nfn i32 main(){" + fixture.calls + "return 0;}\n"
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
				code, out, diagnostics := runForTest(args, root)
				if code != ExitSuccess || out != "" || diagnostics != "" {
					t.Fatalf("%v: %d %q %s", args, code, out, diagnostics)
				}
			}
		})
	}
}
