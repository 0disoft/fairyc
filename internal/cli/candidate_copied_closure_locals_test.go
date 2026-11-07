package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
)

func copiedClosureLocalsProfiles(t *testing.T) []cdriver.Profile {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("copied closure locals native replay requires Windows AMD64; Linux deferred")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for bounded copied closure locals replay")
	}
	return []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile}
}

func copiedClosureLocalsWorkspace(t *testing.T, raw string) string {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.copied-closure-locals"
name = "copied-closure-locals"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", strings.Replace(raw, "module root;", "module main;", 1))
	return root
}

// Each initializer and returned expression spends the same 64-node allowance.
// A 31-operand sum has 61 nodes; (a) has two and b has one: exactly 64.
func copiedClosureLocalsBudgetBody(extraGroup bool) string {
	second := "(a)"
	if extraGroup {
		second = "((a))"
	}
	return "let i32 a=" + strings.Repeat("x+", 30) + "x;let b=" + second + ";return b;"
}

func copiedClosureLocalsCountBody(n int) string {
	var out strings.Builder
	for i := 0; i < n; i++ {
		value := "x"
		if i != 0 {
			value = fmt.Sprintf("l%d", i-1)
		}
		if i%2 == 0 {
			fmt.Fprintf(&out, "let i32 l%d=%s;", i, value)
		} else {
			fmt.Fprintf(&out, "let l%d=%s;", i, value)
		}
	}
	fmt.Fprintf(&out, "return l%d;", n-1)
	return out.String()
}

func copiedClosureLocalsWrap(body string) string {
	return "module root;fn i32 main(){let i32 base=40;let c=fn [i32 base] i32(i32 x){" + body + "};return c(2);}"
}

func copiedClosureLocalsProgram() string {
	return `module root;
fn T keep<T>(T value){let i32 base=40;let c=fn [i32 base] i32(i32 x){let i32 sum=base+x;let result=sum+2;return result;};assert(c(0)==42);return value;}
fn i32 limit16(){let c=fn [] i32(i32 x){` + copiedClosureLocalsCountBody(16) + `};return c(42);}
fn i32 limit64(){let c=fn [] i32(i32 x){` + copiedClosureLocalsBudgetBody(false) + `};return c(1);}
fn i32 main(){
var i32 base=40;let i32 sum=7;
let first=fn [i32 base] i32(i32 x){let i32 sum=base+x;let twice=sum*2;let final=twice/2;final};base=100;
assert(first(2)==42);assert(first(3)==43);assert(first(2)==42);assert(sum==7);
let second=fn [i32 base] i32(i32 x){let i32 sum=base+x;let twice=sum*2;let final=twice/2;return final;};base=200;
assert(second(2)==102);assert(second(3)==103);assert(first(2)==42);
let none=fn [] i32(){let value=40;let i32 two=2;let unused=1;return value+two;};assert(none()==42);
let i32 zero=0;let never_called=fn [i32 zero] i32(){let unused=1/zero;42};
let i32 a=1;let i32 b=2;let i32 c=3;let i32 d=4;
let full=fn [i32 a,i32 b,i32 c,i32 d] i32(i32 w,i32 x,i32 y,i32 z){let i32 sum=a+b+c+d;let rest=w+x+y+z;let final=sum+rest;return final;};assert(full(5,6,7,14)==42);
let arithmetic=fn [] i32(i32 x){let i32 negative=-x;let positive=+x;let divided=(positive*6)/3;let remainder=divided%5;return remainder-negative;};assert(arithmetic(2)==6);
assert(limit16()==42 && limit64()==31);assert(keep<i64>(14)==14);return keep<i32>(first(2));}`
}

type copiedClosureLocalsCase struct {
	name, raw string
	accepted  bool
}

func copiedClosureLocalsCases() []copiedClosureLocalsCase {
	base := copiedClosureLocalsWrap("let i32 local=base+x;let next=local+1;return next;")
	cases := []copiedClosureLocalsCase{
		{"explicit-return", base, true},
		{"tail-and-transitive", copiedClosureLocalsWrap("let local=base+x;let next=local+1;let final=(next*2)/2;final"), true},
		{"sixteen-locals", copiedClosureLocalsWrap(copiedClosureLocalsCountBody(16)), true},
		{"shared-64-nodes", copiedClosureLocalsWrap(copiedClosureLocalsBudgetBody(false)), true},
		{"seventeen-locals", copiedClosureLocalsWrap(copiedClosureLocalsCountBody(17)), false},
		{"shared-65-nodes", copiedClosureLocalsWrap(copiedClosureLocalsBudgetBody(true)), false},
		{"snapshot-hygiene-generics", copiedClosureLocalsProgram(), true},
		// Equal-width edits bind both JSON and the generic definition digest to
		// changed local initializers, returned expressions and local names.
		{"initializer-edit", strings.Replace(base, "local=base+x", "local=base-x", 1), true},
		{"return-edit", strings.Replace(base, "return next;", "return base;", 1), true},
		{"local-name-edit", strings.ReplaceAll(base, "local", "other"), true},
	}
	for _, bad := range []struct{ name, body string }{
		{"forward", "let local=later+x;let later=base;return local;"},
		{"self", "let local=local+x;return local;"},
		{"duplicate", "let local=base;let local=x;return local;"},
		{"capture-collision", "let base=x;return base;"},
		{"parameter-collision", "let x=base;return x;"},
		{"reserved", "let _fy_closure_local=base;return x;"},
		{"mutable", "var i32 local=base+x;return local;"},
		{"wrong-explicit-type", "let i64 local=base;return x;"},
		{"unused-inferred-bool", "let unused=true;return x;"},
		{"unused-inferred-float", "let unused=1.0;return x;"},
		{"unused-inferred-string", "let unused=\"text\";return x;"},
		{"unused-inferred-byte-string", "let unused=b\"text\";return x;"},
		{"unused-inferred-too-large", "let unused=2147483648;return x;"},
		{"typed-bool-initializer", "let i32 unused=true;return x;"},
		{"call", "let local=helper(x);return local;"},
		{"early-return", "return x;let local=base;return local;"},
		{"missing-result", "let local=base+x;"},
		{"return-and-tail", "let local=base+x;return local;local"},
		{"mutation", "let local=base+x;local+=1;return local;"},
		{"control-flow", "let local=base+x;if(true){return local;}return local;"},
		{"unlisted-capture", "let local=outside+x;return local;"},
		{"comparison", "let unused=base==x;return x;"},
		{"nested-block", "let local={base+x};return local;"},
	} {
		raw := copiedClosureLocalsWrap(bad.body)
		if bad.name == "call" {
			raw = strings.Replace(raw, "module root;", "module root;fn i32 helper(i32 x){return x;}", 1)
		}
		if bad.name == "unlisted-capture" {
			raw = strings.Replace(raw, "let i32 base=40;", "let i32 base=40;let i32 outside=1;", 1)
		}
		cases = append(cases, copiedClosureLocalsCase{bad.name, raw, false})
	}
	for _, edit := range []struct{ name, before, after string }{
		{"escape-return", "return c(2);", "return c;"},
		{"escape-copy", "return c(2);", "let other=c;return other(2);"},
		{"mutable-closure", "let c=fn", "var c=fn"},
		{"capture-not-i32", "[i32 base]", "[i64 base]"},
		{"parameter-not-i32", "i32(i32 x)", "i32(i64 x)"},
		{"result-not-i32", "i32(i32 x)", "i64(i32 x)"},
	} {
		cases = append(cases, copiedClosureLocalsCase{edit.name, strings.Replace(base, edit.before, edit.after, 1), false})
	}
	return cases
}

// Go owns an independent whole-source admission oracle. The native Fairy probe
// receives source bytes, never a Go-expanded AST or pre-lowered local tables.
func TestCandidateCopiedClosureLocalsSource(t *testing.T) {
	profiles := copiedClosureLocalsProfiles(t)
	root := t.TempDir()
	copyLexerExample(t, root)
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "copied_closure_locals_test.fy"))
	if err != nil {
		t.Fatal(err)
	}
	var calls, expected strings.Builder
	for _, tc := range copiedClosureLocalsCases() {
		oracleRoot := copiedClosureLocalsWorkspace(t, tc.raw)
		code, out, diagnostics := runForTest([]string{"check"}, oracleRoot)
		if tc.accepted {
			if code != ExitSuccess || diagnostics != "" || !strings.HasPrefix(out, "check ok:") {
				t.Fatalf("current Go oracle %s: %d %q %s", tc.name, code, out, diagnostics)
			}
		} else if code != ExitSourceDiagnostics || diagnostics == "" {
			t.Fatalf("current Go oracle must refuse %s: %d %q %s", tc.name, code, out, diagnostics)
		}
		fmt.Fprintf(&calls, "admission(b%s,b%s,%t);\n", strconv.Quote(tc.name), strconv.Quote(tc.raw), tc.accepted)
		if !tc.accepted {
			continue
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("root.fy", []byte(tc.raw)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("Go written source %s: %v", tc.name, err)
		}
		for _, fn := range parsed.AST.Functions {
			encoded, err := json.Marshal(fn)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(&expected, "%s\n", genericplan.DigestText("source-function-ast-v0", string(encoded)))
		}
		fmt.Fprintf(&calls, "source_case(b%s);\n", strconv.Quote(tc.raw))
		t.Logf("Go oracle accepted %s; written AST definition hash expected", tc.name)
	}
	program := strings.Replace(string(data), "module tests::copied_closure_locals_test;", "module main;", 1)
	program += "\n@effects(console) fn i32 main(){" + calls.String() + "return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "syntax", "scanner", "module_signatures", "module_unit_bind", "module_unit_call_closure", "module_unit_values", "generic_source_function_json")
	for _, profile := range profiles {
		args := []string{"run"}
		if profile.Name == cdriver.OptimizedProfile.Name {
			args = append(args, "--release", "--locked")
		}
		code, out, diagnostics := runForTest(args, root)
		if code != ExitSuccess || diagnostics != "" {
			t.Fatalf("native Fairy %s source: %d %s", profile.Name, code, diagnostics)
		}
		if out != expected.String() {
			t.Fatalf("native Fairy %s written AST definition hash differs from Go\nactual: %s\nexpected: %s", profile.Name, out, expected.String())
		}
		t.Logf("native Fairy %s: %d admission cases + written AST definition hash/copy/origin replay", profile.Name, len(copiedClosureLocalsCases()))
	}
}

func copiedClosureLocalsTrapProgram(divisor int) string {
	return fmt.Sprintf(`module root;fn i32 main(){
let c=fn [] i32(i32 a,i32 b){let unused=a/b;let i32 next=a+1;return 42;};
assert(c(40,2)==42);return c(2147483647,%d);}`, divisor)
}

func TestCandidateCopiedClosureLocalsPipeline(t *testing.T) {
	profiles := copiedClosureLocalsProfiles(t)
	root := t.TempDir()
	copyLexerExample(t, root)
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "copied_closure_pipeline.fy"))
	if err != nil {
		t.Fatal(err)
	}
	sdk, err := os.ReadFile(filepath.Join("..", "sdkstdlib", "sources", "optional", "std", "vec.fy"))
	if err != nil {
		t.Fatal(err)
	}
	sources := []string{copiedClosureLocalsProgram(), copiedClosureLocalsTrapProgram(0), copiedClosureLocalsTrapProgram(1)}
	for scenario, raw := range sources {
		oracleRoot := copiedClosureLocalsWorkspace(t, raw)
		code, out, diagnostics := runForTest([]string{"run"}, oracleRoot)
		if scenario == 0 {
			if code != 42 || out != "" || diagnostics != "" {
				t.Fatalf("Go success oracle: %d %q %s", code, out, diagnostics)
			}
		} else {
			category := []string{"", "division-by-zero", "integer-overflow"}[scenario]
			if code != ExitProgramTrap || out != "" {
				t.Fatalf("Go unused-initializer trap %d: %d %q %s", scenario, code, out, diagnostics)
			}
			expression := []string{"", "a/b", "a+1"}[scenario]
			start := strings.Index(raw, expression)
			if start < 0 {
				t.Fatal("missing Go trap oracle expression")
			}
			line := strings.Count(raw[:start], "\n") + 1
			column := start - strings.LastIndex(raw[:start], "\n")
			want := fmt.Sprintf("trap: %s\n  at src/main.fy:%d:%d\n", category, line, column)
			if diagnostics != want {
				t.Fatalf("Go initializer trap: got %q want %q", diagnostics, want)
			}
		}
	}
	program := strings.Replace(string(data), "module tests::copied_closure_pipeline;", "module main;", 1)
	var calls strings.Builder
	for _, raw := range sources {
		// The existing pipeline harness's second declaration authenticates the
		// real SDK import. It roundtrips/restores checked artifacts before C emit.
		raw = strings.Replace(raw, "module root;", "module root;import std::vec as v;", 1)
		fmt.Fprintf(&calls, "emit_closures(b%s,b%s);\n", strconv.Quote(raw), strconv.Quote(string(sdk)))
	}
	program += "\n@effects(console,filesystem) fn i32 main(){" + calls.String() + "return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "project_compiler_core", "module_project_identified_ir", "module_public_type_text", "module_catalog", "module_unit_c_emit", "module_project_complete_c", "module_unit_body_ir_model", "module_unit_cleanup_mir_model", "module_project_artifact_encode", "module_project_artifact_decode", "candidate_artifact_wire", "static_operation_declarations", "c_text", "syntax", "scanner")
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	vectorRuntime, err := cstdlib.WindowsMSVCCandidateSourceForSymbols(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	scalarRuntime, err := cruntime.ScalarSource(80)
	if err != nil {
		t.Fatal(err)
	}
	for _, producer := range profiles {
		t.Logf("native Fairy %s producer: %d parts", producer.Name, len(build.Sources))
		image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, producer, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		imagePath := filepath.Join(root, "locals-producer-"+producer.Name+".exe")
		if err := os.WriteFile(imagePath, image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
		var output, diagnostics bytes.Buffer
		command := exec.CommandContext(ctx, imagePath)
		command.Stdout, command.Stderr = &output, &diagnostics
		err = command.Run()
		cancel()
		if err != nil || diagnostics.Len() != 0 {
			t.Fatalf("native Fairy %s producer: %v stderr=%s", producer.Name, err, &diagnostics)
		}
		packets := bytes.Split(output.Bytes(), []byte("FYCPARTS1\n"))
		if len(packets) != len(sources)+1 || len(packets[0]) != 0 {
			t.Fatalf("expected %d target packets, got %d", len(sources), len(packets)-1)
		}
		for scenario := range sources {
			packet, err := candidatebuild.Decode(append([]byte("FYCPARTS1\n"), packets[scenario+1]...))
			if err != nil {
				t.Fatal(err)
			}
			targetRuntime := scalarRuntime + "\n" + vectorRuntime
			expectedExit := 42
			if scenario != 0 {
				// A missing trap returns the body's literal 42. The observer's
				// distinct sentinel prevents that optimization from passing.
				expectedExit = 73
				// Observe exact original root byte ranges independently of either
				// compiler's returned spans; unused initializers must run in order.
				raw := strings.Replace(sources[scenario], "module root;", "module root;import std::vec as v;", 1)
				expression, category := "a/b", 2
				if scenario == 2 {
					expression, category = "a+1", 1
				}
				start := strings.Index(raw, expression)
				if start < 0 {
					t.Fatal("missing independently observed initializer")
				}
				targetRuntime = fmt.Sprintf(`#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
_Noreturn void fy_runtime_trap(uint32_t category){(void)category;_Exit(90);}
_Noreturn void fy_runtime_trap_at_v0(uint32_t category,const char *source,uint32_t length,uint64_t start,uint64_t end){if(source && length==10 && memcmp(source,"p1/root.fy",10)==0 && category==%d && start==%d && end==%d)_Exit(73);fprintf(stderr,"trap observation: category=%%u source=%%.*s start=%%llu end=%%llu",category,(int)length,source,(unsigned long long)start,(unsigned long long)end);_Exit(91);}
`, category, start, start+len(expression)) + "\n" + vectorRuntime
			}
			for _, consumer := range profiles {
				artifact, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, consumer, packet.Header, packet.Literals, packet.Units, targetRuntime)
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(root, fmt.Sprintf("locals-%s-%s-%d.exe", producer.Name, consumer.Name, scenario))
				if err := os.WriteFile(path, artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
				output, err := exec.CommandContext(ctx, path).CombinedOutput()
				cancel()
				exit, ok := err.(*exec.ExitError)
				if !ok || exit.ExitCode() != expectedExit || len(output) != 0 {
					t.Fatalf("generated %s/%s scenario %d: %v %q", producer.Name, consumer.Name, scenario, err, output)
				}
				if scenario == 0 {
					t.Logf("generated %s/%s scenario %d: exit=42, empty output, snapshot/repeated-call/limits/generic behavior", producer.Name, consumer.Name, scenario)
				} else {
					t.Logf("generated %s/%s scenario %d: exit=73, empty output, exact initializer origin/category", producer.Name, consumer.Name, scenario)
				}
			}
		}
	}
}
