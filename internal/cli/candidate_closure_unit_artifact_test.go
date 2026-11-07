package cli

import (
	"bytes"
	"encoding/binary"
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

// Independent FYBODY00 oracle for the existing scalar return-42 sample.
// Record words are explicit; neither codec supplies these expected bytes.
func closureArtifactScalarGolden() []byte {
	b := bytes.NewBufferString("FYBODY00")
	words := func(values ...uint64) {
		for _, value := range values {
			_ = binary.Write(b, binary.BigEndian, value)
		}
	}
	words(1, 3, 10, 0, 10, 1)
	words(0, 0, 1, 0, 1, 3, 5, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0,
		1, 1, 1, 1, 0,
		1, 3, 5, 1, 0, 0,
		3, 0, 2, 2, 2)
	words(3,
		1, 5, 2, 6, 1, 0, 0, 0, 0, 0, 2, 2,
		2, 7, 2, 6, 0, 0, 1, 0, 0, 0, 2, 2,
		3, 1, 0, 10, 0, 0, 2, 1, 0, 0, 2, 2)
	words(0, 0, 0, 0, 0, 0, 2)
	b.WriteString("42")
	return b.Bytes()
}

func candidateClosureUnitArtifactWorkspace(t *testing.T) (string, int) {
	t.Helper()
	base := filepath.Join("..", "..", "examples", "lexer_candidate", "tests")
	raw, err := os.ReadFile(filepath.Join(base, "closure_unit_artifact.fy"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_unit_artifact.fy", raw), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("closure artifact syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
	program := strings.Replace(string(raw), "module tests::closure_unit_artifact;", "module main;", 1)
	program = strings.ReplaceAll(program, "tests::module_body_artifact_sample", "artifact_scalar_sample")
	var golden strings.Builder
	for _, octet := range closureArtifactScalarGolden() {
		fmt.Fprintf(&golden, "\\x%02x", octet)
	}
	if len(closureArtifactScalarGolden()) != 754 {
		t.Fatal("scalar oracle width changed")
	}
	program += "\n@effects(console) fn i32 main(){families();scalar_golden(b\"" + golden.String() + "\");\n"
	count := 0
	for _, tc := range candidateClosureUnitValueControls(t) {
		if !tc.accept {
			continue
		}
		program += fmt.Sprintf("verify(b%s,b%s,%d,%d,%d);\n", strconv.Quote(tc.name), strconv.Quote(tc.raw), tc.constructs, tc.calls, tc.captures)
		count++
	}
	for _, tc := range []closureUnitValueCase{
		{name: "return-only-factory", raw: `module main;fn fn i32(i32) make(){let fn i32(i32) f=fn [] i32(i32 x){return x;};return f;}`, constructs: 1},
		{name: "return-only-leaf-copy", raw: `module main;fn fn i32(i32) keep(fn i32(i32) f){let copied=f;return copied;}`},
	} {
		program += fmt.Sprintf("verify(b%s,b%s,%d,0,0);\n", strconv.Quote(tc.name), strconv.Quote(tc.raw), tc.constructs)
		count++
	}
	program += "verify(b\"scalar\",b\"module main;fn i32 main(){return 42;}\",0,0,0);return 0;}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_unit_body_ir_lower", "module_body_artifact_encode", "module_body_artifact_decode", "module_cleanup_artifact_encode", "module_cleanup_artifact_decode", "scanner", "syntax")
	for _, fixture := range []struct{ file, module string }{
		{"module_body_artifact_cleanup.fy", "artifact_scalar_cleanup"},
		{"module_body_artifact_sample.fy", "artifact_scalar_sample"},
	} {
		data, err := os.ReadFile(filepath.Join(base, fixture.file))
		if err != nil {
			t.Fatal(err)
		}
		text := strings.Replace(string(data), "module tests::"+strings.TrimSuffix(fixture.file, ".fy")+";", "module "+fixture.module+";", 1)
		text = strings.ReplaceAll(text, "tests::module_body_artifact_cleanup", "artifact_scalar_cleanup")
		writeTestSource(t, root, "src/"+fixture.module+".fy", text)
	}
	writeTestSource(t, root, "src/main.fy", program)
	return root, count
}

func TestCandidateClosureUnitArtifactSource(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root, count := candidateClosureUnitArtifactWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure artifact source: %d %s %s", code, out, diagnostic)
	}
	t.Logf("%d original-source closure controls plus scalar control; component roundtrip/forgery/boundary fixture source checked (not executed)", count)
}

func TestCandidateClosureUnitArtifactNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 component probe; Linux remains separate")
	}
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root, count := candidateClosureUnitArtifactWorkspace(t)
	candidateClosureArtifactNativeFixture(t, root, count, "component bytes, scalar, forged records")
}

// These are compiler fixtures. Use the existing bounded partitioned host
// builder rather than making monolithic compiler optimization an acceptance gate.
func candidateClosureArtifactNativeFixture(t *testing.T, root string, count int, description string) {
	t.Helper()
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		image, err := cdriver.BuildWindowsMSVCCompilerHostPartsWithStackReserve(toolchain, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "closure-artifact-"+profile.Name+".exe")
		if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		code, out := candidateAtlasProcess(t, path, nil)
		if code != ExitSuccess || out != "" {
			t.Fatalf("%s %s: exit=%d output=%q", description, profile.Name, code, out)
		}
		t.Logf("%s: %d parts, %d controls, %s; exit 0 and empty stdout/stderr", profile.Name, len(build.Sources), count, description)
	}
}
