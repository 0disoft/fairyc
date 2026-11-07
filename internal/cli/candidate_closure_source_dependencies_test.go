package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func closureSourceDependenciesFixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_source_dependencies.fy"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// Read the original byte literals from the authored Fairy fixture. The Go
// oracle never replaces a candidate AST, signature, or dependency record.
func TestCandidateClosureSourceDependenciesOracle(t *testing.T) {
	raw := closureSourceDependenciesFixture(t)
	p, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_source_dependencies.fy", []byte(raw)), source.DefaultLimits())
	if err != nil || p.AST == nil || !p.Tree.ErrorFree() {
		t.Fatalf("fixture syntax: %v %+v", err, p.Tree.Diagnostics)
	}
	seen := map[string]bool{}
	acceptedSources := map[string]bool{
		"module main;fn fn i32(i32) keep(fn i32(i32) action){return action;}":                                                             true,
		"module main;fn fn i32(i32) make(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){return offset+x;};return action;}": true,
		"module main;struct Holder{fn i32(i32) action;}fn Holder keep(Holder value){return value;}":                                       true,
		"module main;fn i32 keep(i32 value){return value;}":                                                                               true,
	}
	accepted, refused := 0, 0
	for _, literal := range p.AST.Literals {
		if !strings.HasPrefix(literal.Raw, `b"module main;`) {
			continue
		}
		original, err := strconv.Unquote(strings.TrimPrefix(literal.Raw, "b"))
		if err != nil {
			t.Fatal(err)
		}
		if seen[original] {
			continue
		}
		seen[original] = true
		t.Run(strconv.Itoa(len(seen)), func(t *testing.T) {
			parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(original)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				t.Fatalf("original syntax: %v %+v", err, parsed.Tree.Diagnostics)
			}
			before, err := json.Marshal(parsed.AST)
			if err != nil {
				t.Fatal(err)
			}
			want := acceptedSources[original]
			analysis := semantic.Analyze([]semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: parsed.AST}})
			ok := len(analysis.Diagnostics) == 0 && len(analysis.Unsupported) == 0 && analysis.CompletedThrough == semantic.PhaseFlow
			if ok != want {
				t.Fatalf("Go semantic accepted=%v want=%v: %+v %+v", ok, want, analysis.Diagnostics, analysis.Unsupported)
			}
			if want {
				owner := genericplan.PackageID("package_coordinate_v1_" + strings.Repeat("1", 64))
				plan, err := genericsource.PreparePackages([]genericsource.PackageModule{{Module: parsed.AST, Owner: owner, Imports: map[string]genericplan.PackageID{}}}, genericplan.DefaultPlanLimits())
				if err != nil || len(plan.Plan.Roots) != 0 || len(plan.Plan.Specializations) != 0 {
					t.Fatalf("Go original source plan: %v %+v", err, plan.Plan)
				}
				accepted++
			} else {
				refused++
			}
			after, err := json.Marshal(parsed.AST)
			if err != nil || string(before) != string(after) {
				t.Fatal("Go oracle mutated original source AST")
			}
		})
	}
	if accepted != 4 || refused != 7 {
		t.Fatalf("original fixture classifications: accepted=%d refused=%d", accepted, refused)
	}
	for _, original := range []string{
		"module main;fn fn i32(i32 keep(){return 0;}",
		"module main;fn fn (i32) keep(){return 0;}",
		"module main;fn fn i32(,i32) keep(){return 0;}",
	} {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(original)), source.DefaultLimits())
		if err == nil && parsed.AST != nil && parsed.Tree.ErrorFree() {
			t.Fatalf("malformed original signature accepted: %s", original)
		}
	}
	t.Logf("original sources: %d accepted source/plan controls, %d semantic refusals, 3 malformed syntax refusals; AST bytes retained", accepted, refused)
}

func closureSourceDependenciesWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyLexerExample(t, root)
	program := strings.Replace(closureSourceDependenciesFixture(t), "module tests::closure_source_dependencies;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test ", "")
	program += "\nfn i32 main(){closure_dependencies_admit_original_sources();closure_dependencies_keep_source_signature_and_child_refusals();closure_dependencies_reject_malformed_owned_links();closure_dependencies_keep_shared_work_boundary();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "generic_source_requests", "module_signatures", "syntax", "scanner")
	return root
}

func TestCandidateClosureSourceDependenciesCheck(t *testing.T) {
	root := closureSourceDependenciesWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("authored source dependency probe check: %d %s %s", code, out, diagnostic)
	} else {
		t.Log(strings.TrimSpace(out))
	}
}

func TestCandidateClosureSourceDependenciesNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 probe; Linux remains batched")
	}
	root := closureSourceDependenciesWorkspace(t)
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		if code, out, diagnostic := runForTest(args, root); code != ExitSuccess || out != "" || diagnostic != "" {
			t.Fatalf("original dependency probe %v: %d %q %s", args, code, out, diagnostic)
		}
	}
}
