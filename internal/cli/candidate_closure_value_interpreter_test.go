package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

type closureValueInterpreterCase struct {
	name     string
	files    []string
	bits     int32
	trap     uint32
	category string
}

func candidateClosureValueInterpreterCases() []closureValueInterpreterCase {
	var cases []closureValueInterpreterCase
	for n := 0; n <= 4; n++ {
		locals, captures, expression := "", []string{}, "x"
		want := int32(42)
		for i := 0; i < n; i++ {
			name := string(rune('a' + i))
			locals += fmt.Sprintf("let %s=%d;", name, i+1)
			captures = append(captures, "i32 "+name)
			expression += "-" + name
			want -= int32(i + 1)
		}
		text := `module main;fn fn i32(i32) make(){` + locals + `let fn i32(i32) f=fn [` + strings.Join(captures, ",") + `] i32(i32 x){return ` + expression + `;};return f;}fn i32 main(){let f=make();return f(42);}`
		cases = append(cases, closureValueInterpreterCase{fmt.Sprintf("returned-captures-%d", n), []string{text}, want, 0, ""})
	}
	cases = append(cases,
		closureValueInterpreterCase{"different-helpers-same-parameter", []string{`module main;fn fn i32(i32) add(){let base=40;let fn i32(i32) f=fn [i32 base] i32(i32 x){return base+x;};return f;}fn fn i32(i32) multiply(){let fn i32(i32) f=fn [] i32(i32 x){return x*2;};return f;}fn i32 apply(fn i32(i32) f,i32 x){return f(x);}fn i32 main(){let a=add();let b=multiply();return apply(a,2)+apply(b,21);}`}, 84, 0, ""},
		closureValueInterpreterCase{"struct-return-copy-snapshot-once", []string{`module main;struct Holder{fn i32(i32) action;}fn Holder make(i32 seed){var i32 a=seed;let b=3;let c=2;let d=4;let fn i32(i32) f=fn [i32 a,i32 b,i32 c,i32 d] i32(i32 x){let reduced=a-b*x;return reduced+c-d;};a=999;return Holder{.action=f};}fn Holder keep(Holder h){let copy=h;return copy;}fn i32 main(){let first=make(50);let second=make(60);let copy=keep(first);let stored=copy;let f=stored.action;let other=second.action;var i32 count=0;let value=f({count=count+1;2});assert(count==1 && other(2)==52);return value;}`}, 42, 0, ""},
		closureValueInterpreterCase{"construction-copy-does-not-run-body", []string{`module main;struct Holder{fn i32(i32) f;}fn fn i32(i32) make(){let fn i32(i32) f=fn [] i32(i32 x){let unused=1/0;return x;};return f;}fn i32 main(){let f=make();let holder=Holder{.f=f};let copy=holder;discard copy;return 42;}`}, 42, 0, ""},
		closureValueInterpreterCase{"argument-before-body-trap", []string{`module main;fn i32 main(){let top=2147483647;let fn i32(i32) f=fn [i32 top] i32(i32 x){let unused=top+1;return x;};return f(1/0);}`}, 0, 3, "division-by-zero"},
		closureValueInterpreterCase{"body-local-trap-order", []string{`module main;fn i32 main(){let fn i32(i32) f=fn [] i32(i32 x){let unused=2147483647+1;return x/0;};return f(42);}`}, 0, 2, "integer-overflow"},
		closureValueInterpreterCase{"body-remainder-trap", []string{`module main;fn i32 main(){let divisor=-1;let fn i32(i32) f=fn [i32 divisor] i32(i32 x){return x%divisor;};return f(-2147483648);}`}, 0, 4, "signed-division-overflow"},
		closureValueInterpreterCase{"signed-capture-and-repeated-call", []string{`module main;fn i32 main(){let base=-2;let fn i32(i32) f=fn [i32 base] i32(i32 x){let y=x+base;return y;};assert(f(44)==42);return f(45);}`}, 43, 0, ""},
	)
	// Original modules with equal helper declaration numbers must resolve by
	// real creator identity, even when both pass through the same parameter.
	for _, tc := range candidateClosureProjectCCases() {
		trap := uint32(0)
		if tc.trap != "" {
			trap = 3
		}
		bits := int32(tc.exit)
		if trap != 0 {
			bits = 0
		}
		cases = append(cases, closureValueInterpreterCase{"project-" + tc.name, tc.files, bits, trap, tc.trap})
	}
	return cases
}

func candidateClosureValueInterpreterControls(t *testing.T, cases []closureValueInterpreterCase) {
	t.Helper()
	for _, tc := range cases {
		var files []semantic.File
		for _, text := range tc.files {
			module := strings.TrimSpace(strings.TrimPrefix(text[:strings.IndexByte(text, ';')], "module "))
			path := strings.ReplaceAll(module, "::", "/") + ".fy"
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(text)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				t.Fatalf("Go syntax %s: %v %+v", tc.name, err, parsed.Tree.Diagnostics)
			}
			files = append(files, semantic.File{Package: hir.DefaultPackage, Path: path, Module: parsed.AST})
		}
		checked, err := compiler.CheckExplicitFiles(files)
		if err != nil || !checked.Verified {
			t.Fatalf("Go source %s: %v diagnostics=%+v unsupported=%+v", tc.name, err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
		}
		out, err := interpreter.ExecuteWithOptions(checked.MIR, hir.QualifiedModulePath(hir.DefaultPackage, "main")+"::main", interpreter.Options{Context: t.Context()})
		if err != nil || len(out.Stdout) != 0 || len(out.Stderr) != 0 {
			t.Fatalf("Go interpreter %s: %+v %v", tc.name, out, err)
		}
		if tc.category == "" {
			if out.Kind != interpreter.OutcomeReturn || out.Value != tc.bits {
				t.Fatalf("Go %s: %+v want=%d", tc.name, out, tc.bits)
			}
		} else if out.Kind != interpreter.OutcomeTrap || out.TrapCategory != tc.category {
			t.Fatalf("Go %s: %+v want trap=%s", tc.name, out, tc.category)
		}
	}
}

func candidateClosureValueInterpreterWorkspace(t *testing.T) (string, string) {
	t.Helper()
	cases := candidateClosureValueInterpreterCases()
	candidateClosureValueInterpreterControls(t, cases)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_value_interpreter.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::closure_value_interpreter;", "module main;", 1)
	main += "\n@effects(console) fn i32 main(){\n"
	var expected strings.Builder
	for i, tc := range cases {
		texts := [3]string{}
		copy(texts[:], tc.files)
		main += fmt.Sprintf("exercise(b%s,b%s,b%s,b%s,%d,%d,%t);\n", strconv.Quote(tc.name), strconv.Quote(texts[0]), strconv.Quote(texts[1]), strconv.Quote(texts[2]), uint32(tc.bits), tc.trap, i == 0)
		fmt.Fprintf(&expected, "%s\n0/%d/%d\n", tc.name, tc.trap, uint32(tc.bits))
	}
	main += "return 0;}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	writeTestSource(t, root, "src/main.fy", main)
	copyCandidateParityModules(t, root, "project_compiler_core", "module_unit_interpreter", "module_unit_interpreter_closures", "module_public_type_text", "module_catalog", "sha256_stream", "syntax", "scanner")
	return root, expected.String()
}

func TestCandidateClosureValueInterpreterSource(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root, _ := candidateClosureValueInterpreterWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure interpreter source: %d %s %s", code, out, diagnostic)
	}
	t.Logf("%d independent Go interpreter outcomes and authored Fairy Complete/semantic-interpreter fixture source checked; no native execution", len(candidateClosureValueInterpreterCases()))
}

// Parent runs this only after all concurrent source/cache writers freeze.
// The native executable runs the Fairy interpreter on original source; it does
// not emit or execute a generated C consumer as a semantic substitute.
func TestCandidateClosureValueInterpreterNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows Fairy interpreter replay; Linux remains batched")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("parent: set FAIRYC_RUN_PARITY_INTEGRATION=1 after source freeze")
	}
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root, expected := candidateClosureValueInterpreterWorkspace(t)
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		if !t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCCompilerHostPartsWithStackReserve(toolchain, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "closure-value-interpreter-"+profile.Name+".exe")
			if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, path)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			out, err := cmd.Output()
			if err != nil || stderr.Len() != 0 || string(out) != expected {
				t.Fatalf("Fairy semantic interpreter %s: %v stderr=%s actual=%q expected=%q", profile.Name, err, &stderr, out, expected)
			}
		}) {
			return
		} // Stop after the first native profile failure.
	}
}
