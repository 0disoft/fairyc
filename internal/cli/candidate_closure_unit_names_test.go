package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const closureUnitNamesReceipt = "unit closure names: original captures 0..4, ordered snapshots, local/provider shadowing, helper/capture and source-owner refusals\n"

func closureUnitNamesProgram(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_unit_names.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::closure_unit_names;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test ", "")
	program = strings.Replace(program, "import std::bytes as bytes;", "import std::io as io;\nimport std::bytes as bytes;", 1)
	program += "\n@effects(console) fn i32 main(){closure_unit_names_original_captures_and_provider_shadowing();closure_unit_names_refuse_forged_helper_and_capture_operands();closure_unit_names_keep_owner_refusals();assert(io::write(\"" + strings.TrimSuffix(closureUnitNamesReceipt, "\n") + "\\n\").is_ok);return 0;}\n"
	return program
}

func TestCandidateClosureUnitNamesSyntax(t *testing.T) {
	for name, raw := range map[string][]byte{"main.fy": []byte(closureUnitNamesProgram(t)), "module_unit_name_walk.fy": nil} {
		if raw == nil {
			var err error
			raw, err = os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name))
			if err != nil {
				t.Fatal(err)
			}
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(name, raw), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("%s syntax: %v %+v", name, err, parsed.Tree.Diagnostics)
		}
	}
}

// These controls classify authored original source with Go. They do not claim
// execution of the candidate name walk; that receipt belongs to Native below.
func TestCandidateClosureUnitNamesOracle(t *testing.T) {
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(closureUnitNamesProgram(t))), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("probe syntax: %v", err)
	}
	provider, err := frontend.AnalyzeSyntax(source.NewFile("provider.fy", []byte("module provider;pub fn i32 action(i32 x){return x;}")), source.DefaultLimits())
	if err != nil || provider.AST == nil || !provider.Tree.ErrorFree() {
		t.Fatalf("provider syntax: %v", err)
	}
	accepted, refused := 0, 0
	for _, literal := range parsed.AST.Literals {
		if !strings.HasPrefix(literal.Raw, `b"module main;`) {
			continue
		}
		raw, err := strconv.Unquote(strings.TrimPrefix(literal.Raw, "b"))
		if err != nil {
			t.Fatal(err)
		}
		want := strings.Contains(raw, "import provider") || strings.Contains(raw, "fn i32 ordinary")
		original, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(raw)), source.DefaultLimits())
		ok := err == nil && original.AST != nil && original.Tree.ErrorFree()
		if ok {
			analysis := semantic.Analyze([]semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: original.AST}, {Package: hir.DefaultPackage, Path: "provider.fy", Module: provider.AST}})
			ok = len(analysis.Diagnostics) == 0 && len(analysis.Unsupported) == 0 && analysis.CompletedThrough == semantic.PhaseFlow
			if ok != want {
				t.Fatalf("Go source control accepted=%v want=%v: %s: %+v %+v", ok, want, raw, analysis.Diagnostics, analysis.Unsupported)
			}
		} else if want {
			t.Fatalf("valid original syntax: %s: %v", raw, err)
		}
		if want {
			accepted++
		} else {
			refused++
		}
	}
	if accepted != 6 || refused != 8 {
		t.Fatalf("original source controls: accepted=%d refused=%d", accepted, refused)
	}
	t.Logf("Go original-source controls: %d accepted, %d refused", accepted, refused)
}

func closureUnitNamesWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyLexerExample(t, root)
	writeTestSource(t, root, "src/main.fy", closureUnitNamesProgram(t))
	copyCandidateParityModules(t, root, "module_unit_ordinary_names", "module_unit_name_walk", "module_unit_name_model", "module_unit_body_ast", "module_unit_bind", "module_unit_interfaces", "module_signatures", "copied_closure_tables", "syntax", "scanner", "body_state")
	return root
}

func TestCandidateClosureUnitNamesCheck(t *testing.T) {
	root := closureUnitNamesWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("original unit-name probe check: %d %s %s", code, out, diagnostic)
	} else {
		t.Log(strings.TrimSpace(out))
	}
}

func TestCandidateClosureUnitNamesNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 focused probe; Linux remains batched")
	}
	root := closureUnitNamesWorkspace(t)
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		if code, out, diagnostic := runForTest(args, root); code != ExitSuccess || out != closureUnitNamesReceipt || diagnostic != "" {
			t.Fatalf("original unit-name probe %v: %d %q %s", args, code, out, diagnostic)
		}
	}
	t.Log(strings.TrimSpace(closureUnitNamesReceipt))
}
