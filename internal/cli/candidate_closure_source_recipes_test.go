package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func candidateClosureSourceRecipesWorkspace(t *testing.T) (string, int) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_source_recipes.fy"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_source_recipes.fy", raw), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("closure source recipes syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
	main := strings.Replace(string(raw), "module tests::closure_source_recipes;", "module main;", 1)
	main += "\n@effects(console) fn i32 main(){scalar_control();detached_forgery();\n"
	count := 0
	for _, tc := range candidateClosureUnitValueControls(t) {
		if tc.accept {
			main += fmt.Sprintf("verify(b%s,b%s,%t);\n", strconv.Quote(tc.name), strconv.Quote(tc.raw), count == 0)
			count++
		}
	}
	// Same-width originals change source-owned semantics while remaining valid
	// closure Programs with the same table shapes and declaration coordinates.
	originals := []string{
		`module main;fn i32 main(){let a=3;let b=4;let fn i32(i32) f=fn [i32 a,i32 b] i32(i32 x){let n=a+x;return n+b;};return f(20);}`,
		`module main;fn i32 main(){let a=3;let b=4;let fn i32(i32) f=fn [i32 a,i32 b] i32(i32 x){let n=a-x;return n+b;};return f(20);}`,
		`module main;fn i32 main(){let a=3;let b=4;let fn i32(i32) f=fn [i32 b,i32 a] i32(i32 x){let n=a+x;return n+b;};return f(20);}`,
		`module main;fn i32 main(){let a=9;let b=4;let fn i32(i32) f=fn [i32 a,i32 b] i32(i32 x){let n=a+x;return n+b;};return f(20);}`,
		`module main;fn i32 main(){let a=3;let b=4;let fn i32(i32) f=fn [i32 a,i32 b] i32(i32 x){let n=a+x;return n+b;};return f(21);}`,
	}
	for i, text := range originals {
		main += fmt.Sprintf("verify(b%s,b%s,false);\n", strconv.Quote(fmt.Sprintf("ordered-source-%d", i)), strconv.Quote(text))
		if i > 0 {
			main += "mismatch(b" + strconv.Quote(originals[0]) + ",b" + strconv.Quote(text) + ");\n"
		}
		count++
	}
	main += "return 0;}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_project_artifact_calls", "module_project_artifact_recipes", "module_project_artifact_coverage", "module_unit_artifact_identities", "module_project_compile", "project_source_plan", "module_program_artifact_encode", "module_program_artifact_decode", "scanner", "syntax")
	writeTestSource(t, root, "src/main.fy", main)
	return root, count
}

// Source checking checks the actual matcher and authored assertions. It does
// not execute detached matching, coverage or the adversarial mutations.
func TestCandidateClosureSourceRecipesSource(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root, count := candidateClosureSourceRecipesWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure source recipes: %d %s %s", code, out, diagnostic)
	}
	t.Logf("%d accepted original sources plus scalar control; full calls/recipes/coverage, detached-valid mutations and exact-work assertions source checked, not executed", count)
}

// Parent schedules this sequentially after source freeze. No native build is
// part of the worker's Source check; existing host limits remain unchanged.
func TestCandidateClosureSourceRecipesNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("closure recipe native probe requires Windows AMD64; Linux is separately batched")
	}
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root, count := candidateClosureSourceRecipesWorkspace(t)
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	// Filtering low leaves the optimized subtest unselected for one coordinated
	// diagnostic run. A normal Native run still executes both profiles.
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		if !t.Run(profile.Name, func(t *testing.T) {
			t.Logf("%s: %d private fixture parts", profile.Name, len(build.Sources))
			image, err := cdriver.BuildWindowsMSVCCompilerHostPartsWithStackReserve(toolchain, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "closure-source-recipes-"+profile.Name+".exe")
			if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			code, stdout := candidateAtlasProcess(t, path, nil)
			if code != 0 || stdout != "" {
				t.Fatalf("closure source recipes %s: exit=%d stdout=%q", profile.Name, code, stdout)
			}
			t.Logf("%s: %d original closure controls, scalar gate, detached-valid forgery/mismatch and exact meter assertions executed", profile.Name, count)
		}) {
			return
		}
	}
}
