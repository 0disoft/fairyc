package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

type closureProjectCCase struct {
	name            string
	files           []string
	exit, made      int
	trap            string
	sameDeclaration bool
}

func candidateClosureProjectCCases() []closureProjectCCase {
	const factory = `module provider;pub fn fn i32(i32) make(i32 base){let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};return action;}`
	return []closureProjectCCase{
		{"factory-provider-parameter", []string{`module main;import provider as p;fn i32 apply(fn i32(i32) action,i32 value){return action(value);}fn i32 main(){let action=p::make(40);let value=apply(action,2);assert(value==42);return value;}`, factory}, 42, 1, "", false},
		{"same-declaration-different-helpers", []string{`module main;import left as a;import right as b;fn i32 apply(fn i32(i32) action,i32 value){return action(value);}fn i32 main(){let left=a::make(40);let right=b::make(2);let first=apply(left,2);let second=apply(right,21);assert(first==42 && second==42);return first+second;}`, strings.Replace(factory, "module provider;", "module left;", 1), strings.Replace(strings.Replace(factory, "module provider;", "module right;", 1), "base+x", "base*x", 1)}, 84, 2, "", true},
		{"struct-copy-snapshot", []string{`module main;import provider as p;fn i32 main(){let original=p::make(40);let copy=p::keep(original);let stored=copy;let extracted=stored.action;let value=extracted(2);assert(value==42);return value;}`, `module provider;pub struct Holder{fn i32(i32) action;}pub fn Holder make(i32 seed){var i32 base=seed;let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};base=0;return Holder{.action=action};}pub fn Holder keep(Holder value){let copy=value;return copy;}`}, 42, 1, "", false},
		{"three-module-factory-forward", []string{`module main;import relay as r;fn i32 main(){let original=r::forward(40);let copy=original;let value=copy(2);assert(value==42);return value;}`, `module relay;import provider as p;pub fn fn i32(i32) forward(i32 base){let original=p::make(base);let copy=original;return copy;}`, factory}, 42, 1, "", false},
		{"unused-trapping-helper", []string{`module main;import provider as p;fn i32 main(){let action=p::make(0);let copy=action;discard copy;return 42;}`, `module provider;pub fn fn i32(i32) make(i32 divisor){let fn i32(i32) action=fn [i32 divisor] i32(i32 x){return x/divisor;};return action;}`}, 42, 1, "", false},
		{"provider-body-division-trap", []string{`module main;import provider as p;fn i32 main(){let action=p::make(0);return action(42);}`, `module provider;pub fn fn i32(i32) make(i32 divisor){let fn i32(i32) action=fn [i32 divisor] i32(i32 x){return x/divisor;};return action;}`}, 82, 1, "division-by-zero", false},
		{"argument-trap-before-provider-body-overflow", []string{`module main;import provider as p;fn i32 main(){let action=p::make(2147483647);return action(42/0);}`, `module provider;pub fn fn i32(i32) make(i32 top){let fn i32(i32) action=fn [i32 top] i32(i32 x){let unused=top+1;return x;};return action;}`}, 82, 1, "division-by-zero", false},
	}
}

// Independent Go semantic/MIR/interpreter controls run original modules. This
// neither invokes Fairy Complete/C emission nor runs any native compiler.
func candidateClosureProjectCControls(t *testing.T) []closureProjectCCase {
	t.Helper()
	cases := candidateClosureProjectCCases()
	for _, tc := range cases {
		var files []semantic.File
		for _, raw := range tc.files {
			module := strings.TrimSpace(strings.TrimPrefix(raw[:strings.IndexByte(raw, ';')], "module "))
			path := strings.ReplaceAll(module, "::", "/") + ".fy"
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(raw)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				t.Fatalf("Go project C syntax %s/%s: %v %+v", tc.name, path, err, parsed.Tree.Diagnostics)
			}
			files = append(files, semantic.File{Package: hir.DefaultPackage, Path: path, Module: parsed.AST})
		}
		checked, err := compiler.CheckExplicitFiles(files)
		if err != nil || !checked.Verified {
			t.Fatalf("Go project C %s: %v diagnostics=%+v unsupported=%+v", tc.name, err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
		}
		out, err := interpreter.ExecuteWithOptions(checked.MIR, hir.QualifiedModulePath(hir.DefaultPackage, "main")+"::main", interpreter.Options{Context: t.Context()})
		if err != nil || len(out.Stdout) != 0 || len(out.Stderr) != 0 {
			t.Fatalf("Go project C interpreter %s: %+v %v", tc.name, out, err)
		}
		if tc.trap == "" {
			if out.Kind != interpreter.OutcomeReturn || int(out.Value) != tc.exit {
				t.Fatalf("Go project C %s returned %+v want %d", tc.name, out, tc.exit)
			}
		} else if out.Kind != interpreter.OutcomeTrap || out.TrapCategory != tc.trap {
			t.Fatalf("Go project C %s trap=%+v want=%s", tc.name, out, tc.trap)
		}
	}
	return cases
}

func candidateClosureProjectCWorkspace(t *testing.T) (string, []closureProjectCCase) {
	t.Helper()
	cases := candidateClosureProjectCControls(t)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_project_c.fy"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_project_c.fy", raw), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("project C fixture syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
	main := strings.Replace(string(raw), "module tests::closure_project_c;", "module main;", 1)
	main += fmt.Sprintf("\n@effects(console) fn i32 main(){assert(io::write(\"FYPROJECTC1\\n%d\\n\").is_ok);\n", len(cases))
	for _, tc := range cases {
		texts := [3]string{}
		copy(texts[:], tc.files)
		main += fmt.Sprintf("produce(b%s,b%s,b%s,b%s,%d,%t);\n", strconv.Quote(tc.name), strconv.Quote(texts[0]), strconv.Quote(texts[1]), strconv.Quote(texts[2]), tc.made, tc.sameDeclaration)
	}
	main += "return 0;}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	writeTestSource(t, root, "src/main.fy", main)
	copyCandidateParityModules(t, root, "project_compiler_core", "module_project_identified_ir", "module_unit_c_emit", "module_unit_c_names", "module_unit_body_ir_closures", "sha256_stream", "syntax", "scanner")
	return root, cases
}

func TestCandidateClosureProjectCControls(t *testing.T) {
	cases := candidateClosureProjectCControls(t)
	t.Logf("%d original-source Go checks and interpreter outcomes; no native execution", len(cases))
}

func TestCandidateClosureProjectCSource(t *testing.T) {
	root, _ := candidateClosureProjectCWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("project Complete/C producer source check: %d %s %s", code, out, diagnostic)
	}
}

// Parent-only. Reuse the unit C exact-byte packet reader, split producer builder
// and the same target runtime/consumer builders. Only seven consumer cases run.
func TestCandidateClosureProjectCNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("project C native consumers require Windows AMD64; Linux remains batched")
	}
	previous := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	root, cases := candidateClosureProjectCWorkspace(t)
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	producer, err := cdriver.BuildWindowsMSVCCompilerHostPartsWithStackReserve(toolchain, cdriver.LowProfile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	producerPath := filepath.Join(root, "closure-project-c-producer.exe")
	if err := os.WriteFile(producerPath, producer.Bytes, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	command := exec.CommandContext(ctx, producerPath)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	cancel()
	if err != nil || stderr.Len() != 0 {
		t.Fatalf("Fairy original-source Complete/project C producer: %v stderr=%s", err, &stderr)
	}
	reader := bufio.NewReader(bytes.NewReader(output))
	marker, err := reader.ReadString('\n')
	if err != nil || marker != "FYPROJECTC1\n" {
		t.Fatalf("project C marker %q: %v", marker, err)
	}
	count, err := reader.ReadString('\n')
	if err != nil || count != strconv.Itoa(len(cases))+"\n" {
		t.Fatalf("project C case count %q: %v", count, err)
	}
	targetRuntime, err := cruntime.ScalarSource(80)
	if err != nil {
		t.Fatal(err)
	}
	for i, tc := range cases {
		if name := string(candidateClosureUnitCMember(t, reader)); name != tc.name {
			t.Fatalf("project C ordered case %q want %q", name, tc.name)
		}
		single := candidateClosureUnitCMember(t, reader)
		packet, err := candidatebuild.Decode(candidateClosureUnitCMember(t, reader))
		if err != nil {
			t.Fatalf("decode project C %s: %v", tc.name, err)
		}
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			for _, form := range []string{"single", "partitioned"} {
				passed := t.Run(tc.name+"/"+profile.Name+"/"+form, func(t *testing.T) {
					var image []byte
					if form == "single" {
						artifact, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, string(single), targetRuntime)
						if err != nil {
							t.Fatal(err)
						}
						image = artifact.Bytes
					} else {
						artifact, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, profile, packet.Header, packet.Literals, packet.Units, targetRuntime)
						if err != nil {
							t.Fatal(err)
						}
						image = artifact.Bytes
					}
					path := filepath.Join(root, fmt.Sprintf("closure-project-c-%d-%s-%s.exe", i, profile.Name, form))
					if err := os.WriteFile(path, image, 0700); err != nil {
						t.Fatal(err)
					}
					exit, out := candidateAtlasProcess(t, path, nil)
					if exit != tc.exit || out != "" {
						t.Fatalf("project C consumer: exit=%d want=%d stdout/stderr=%q", exit, tc.exit, out)
					}
				})
				if !passed {
					return
				}
			}
		}
	}
	if _, err := reader.ReadByte(); err != io.EOF {
		t.Fatalf("trailing project C producer bytes: %v", err)
	}
}
