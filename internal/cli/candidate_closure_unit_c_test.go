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
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func candidateClosureUnitCFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_unit_c.fy")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_unit_c.fy", raw), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("unit C fixture syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
	return strings.Replace(string(raw), "module tests::closure_unit_c;", "module main;", 1)
}

// This prepares an executable probe for parent-owned native validation; Source
// below only checks it. Controls cover captures 0..4, escaped factories, copies,
// struct storage, mutable snapshots and different helpers through one parameter.
func candidateClosureUnitCWorkspace(t *testing.T) string {
	t.Helper()
	main := candidateClosureUnitCFixture(t) + "\n@effects(console) fn i32 main(){\n"
	for _, tc := range candidateClosureUnitValueControls(t) {
		if tc.accept && tc.name != "signature-only-void-consumer" {
			main += "verify(b" + strconv.Quote(tc.raw) + ",true);\n"
		}
	}
	main += `verify(b"module main;fn i32 main(){return 42;}",false);
verify(b"module main;fn fn i32(i32) make(){let i32 a=0;let fn i32(i32) f=fn [i32 a] i32(i32 x){return x/a;};return f;}fn i32 main(){let f=make();return f(1);}",true);
verify(b"module main;fn fn i32(i32) make(){let i32 a=0;let fn i32(i32) f=fn [i32 a] i32(i32 x){return x/a;};return f;}fn i32 main(){let f=make();return 0;}",true);
`
	cases := candidateClosureUnitCRuntimeCases()
	main += fmt.Sprintf("assert(io::write(\"FYUNITC1\\n\").is_ok && io::write_usize(%d).is_ok && io::write(\"\\n\").is_ok);\n", len(cases))
	for _, tc := range cases {
		main += "produce(b" + strconv.Quote(tc.name) + ",b" + strconv.Quote(tc.raw) + ");\n"
	}
	main += "return 0;}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	writeTestSource(t, root, "src/main.fy", main)
	copyCandidateParityModules(t, root, "module_unit_c_emit", "module_unit_c_closure", "module_unit_body_ir_lower", "module_unit_checked_bodies", "module_signatures", "syntax", "scanner")
	return root
}

func TestCandidateClosureUnitCControls(t *testing.T) {
	candidateClosureUnitValueControls(t)
	candidateClosureUnitCFixture(t)
	for _, tc := range candidateClosureUnitCRuntimeCases() {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(tc.raw)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("runtime original source %s: %v %+v", tc.name, err, parsed.Tree.Diagnostics)
		}

		checked, err := compiler.CheckExplicitFiles([]semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: parsed.AST}})
		if err != nil || !checked.Verified {
			t.Fatalf("runtime original Go control %s: %v diagnostics=%+v unsupported=%+v", tc.name, err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
		}
	}
}

// No run/build/native calls: parent retains the native compiler CPU reservation.
func TestCandidateClosureUnitCSource(t *testing.T) {
	root := candidateClosureUnitCWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure unit C source check: %d %s %s", code, out, diagnostic)
	}
}

// Expected results are independent of emitted C. Embedded source assertions
// verify snapshot preservation, copied struct contents and repeated invocation.
func candidateClosureUnitCRuntimeCases() []candidateClosureCCase {
	var cases []candidateClosureCCase
	for count := 0; count <= 4; count++ {
		var locals, captures []string
		body, total := "x", 0
		for i := 0; i < count; i++ {
			name := string(rune('a' + i))
			locals = append(locals, fmt.Sprintf("let i32 %s=%d;", name, i+1))
			captures = append(captures, "i32 "+name)
			body += "+" + name
			total += i + 1
		}
		raw := "module main;fn fn i32(i32) make(){" + strings.Join(locals, "") + "let fn i32(i32) f=fn [" + strings.Join(captures, ",") + "] i32(i32 x){return " + body + ";};return f;}fn i32 main(){let escaped=make();return escaped(" + strconv.Itoa(42-total) + ");}"
		cases = append(cases, candidateClosureCCase{fmt.Sprintf("returned-captures-%d", count), raw, 42})
	}
	for _, tc := range candidateClosureCCases() {
		switch tc.name {
		case "returned-four-captures-struct-snapshot", "different-targets-one-parameter", "unused-trapping-body-copy", "unused-local-traps-on-invocation", "argument-trap-precedes-body-overflow", "unused-body-overflow", "runtime-signed-divide-overflow", "signed-boundary-success", "argument-once-body-locals-repeat", "signature-only-no-literal":
			cases = append(cases, tc)
		}
	}
	cases = append(cases, candidateClosureCCase{"different-helpers-return-84", `module main;fn fn i32(i32) first(){let i32 base=40;let fn i32(i32) f=fn [i32 base] i32(i32 x){return base+x;};return f;}fn fn i32(i32) second(){let fn i32(i32) f=fn [] i32(i32 x){return x*2;};return f;}fn i32 apply(fn i32(i32) f,i32 x){return f(x);}fn i32 main(){let a=first();let b=second();return apply(a,2)+apply(b,21);}`, 84})
	cases = append(cases, candidateClosureCCase{"first-body-local-trap-precedes-second", `module main;fn i32 main(){let i32 top=2147483647;let fn i32(i32) f=fn [i32 top] i32(i32 x){let first=42/x;let second=top+1;return first+second;};return f(0);}`, 82})
	return cases
}

// Framing reads exact byte counts, independent of CRLF or newlines inside C.
func candidateClosureUnitCMember(t *testing.T, reader *bufio.Reader) []byte {
	t.Helper()
	line, err := reader.ReadString('\n')
	if err != nil || len(line) > 9 {
		t.Fatalf("unit C member length %q: %v", line, err)
	}
	n, err := strconv.Atoi(strings.TrimSuffix(line, "\n"))
	if err != nil || n < 1 || n > 2<<20 || line != strconv.Itoa(n)+"\n" {
		t.Fatalf("noncanonical unit C member length %q", line)
	}
	data := make([]byte, n)
	if _, err = io.ReadFull(reader, data); err != nil {
		t.Fatal(err)
	}
	return data
}

// Parent explicitly owns native execution scheduling. Source does not call this
// test, compile an MSVC producer, run the Fairy producer, or execute generated C.
func TestCandidateClosureUnitCNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("unit C native consumer requires Windows AMD64; Linux remains batched")
	}
	previous := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	root := candidateClosureUnitCWorkspace(t)
	cases := candidateClosureUnitCRuntimeCases()
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	// The verified private compiler input uses the existing two-CL host bound.
	// One low-profile Fairy producer limits build cost. Both consumer profiles
	// independently compile and execute every single-buffer and split output.
	producer, err := cdriver.BuildWindowsMSVCCompilerHostPartsWithStackReserve(toolchain, cdriver.LowProfile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	producerPath := filepath.Join(root, "closure-unit-c-producer.exe")
	if err = os.WriteFile(producerPath, producer.Bytes, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	command := exec.CommandContext(ctx, producerPath)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	cancel()
	if err != nil || stderr.Len() != 0 {
		t.Fatalf("Fairy unit C producer: %v stderr=%s", err, &stderr)
	}
	reader := bufio.NewReader(bytes.NewReader(output))
	marker, err := reader.ReadString('\n')
	if err != nil || marker != "FYUNITC1\n" {
		t.Fatalf("unit C producer marker %q: %v", marker, err)
	}
	count, err := reader.ReadString('\n')
	if err != nil || count != strconv.Itoa(len(cases))+"\n" {
		t.Fatalf("unit C producer case count %q: %v", count, err)
	}
	targetRuntime, err := cruntime.ScalarSource(80)
	if err != nil {
		t.Fatal(err)
	}
	for i, tc := range cases {
		single := candidateClosureUnitCMember(t, reader)
		parts := candidateClosureUnitCMember(t, reader)
		packet, err := candidatebuild.Decode(parts)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.name, err)
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
					path := filepath.Join(root, fmt.Sprintf("closure-unit-c-%d-%s-%s.exe", i, profile.Name, form))
					if err := os.WriteFile(path, image, 0700); err != nil {
						t.Fatal(err)
					}
					exit, stdout := candidateAtlasProcess(t, path, nil)
					if exit != tc.exit || stdout != "" {
						t.Fatalf("generated C execution: exit=%d want=%d stdout=%q", exit, tc.exit, stdout)
					}
					t.Logf("generated C executed: exit=%d empty stdout/stderr", exit)
				})
				if !passed {
					return
				}
			}
		}
	}
	if _, err := reader.ReadByte(); err != io.EOF {
		t.Fatalf("trailing unit C producer output: %v", err)
	}
}
