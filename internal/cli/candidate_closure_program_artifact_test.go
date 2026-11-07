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

// The full old-family oracle is assembled from explicit record words. Its body
// uses the existing independent component oracle; no encoder supplies bytes.
func closureProgramScalarGolden() []byte {
	words := func(marker string, values ...uint64) []byte {
		b := bytes.NewBufferString(marker)
		for _, n := range values {
			_ = binary.Write(b, binary.BigEndian, n)
		}
		return b.Bytes()
	}
	cleaned := words("FYCLEAN0", 0, 2,
		1, 1, 0, 0, 0, 0, 1, 7, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		1, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 0)
	b := bytes.NewBufferString("FYPROG00")
	member := func(data []byte) { _ = binary.Write(b, binary.BigEndian, uint64(len(data))); b.Write(data) }
	member(cleaned)
	member(words("FYSDK000", 0, 0, 0))
	member(words("FYMETA00", 0, 2, 0, 0, 0, 0, 0))
	member(words("FYSTATE0", 0, 0, 0))
	_ = binary.Write(b, binary.BigEndian, uint64(1))
	member(closureArtifactScalarGolden())
	return b.Bytes()
}

func candidateClosureProgramArtifactWorkspace(t *testing.T) (string, int) {
	t.Helper()
	base := filepath.Join("..", "..", "examples", "lexer_candidate", "tests")
	raw, err := os.ReadFile(filepath.Join(base, "closure_program_artifact.fy"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_program_artifact.fy", raw), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("program artifact fixture syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
	main := strings.Replace(string(raw), "module tests::closure_program_artifact;", "module main;", 1)
	var golden strings.Builder
	for _, b := range closureProgramScalarGolden() {
		fmt.Fprintf(&golden, "\\x%02x", b)
	}
	main += "\nfn i32 main(){scalar_golden(b\"" + golden.String() + "\");\n"
	count := 0
	for _, tc := range candidateClosureUnitValueControls(t) {
		if tc.accept {
			main += "verify(b" + strconv.Quote(tc.raw) + ");\n"
			count++
		}
	}
	main += "return 0;}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_program_artifact_encode", "module_program_artifact_decode", "module_metadata_artifact_model", "module_unit_checked_bodies", "module_unit_body_ir_lower", "scanner", "syntax")
	for _, f := range []struct{ file, module string }{
		{"module_body_artifact_cleanup.fy", "closure_program_scalar_cleanup"},
		{"module_body_artifact_sample.fy", "closure_program_scalar_sample"},
	} {
		data, err := os.ReadFile(filepath.Join(base, f.file))
		if err != nil {
			t.Fatal(err)
		}
		text := strings.Replace(string(data), "module tests::"+strings.TrimSuffix(f.file, ".fy")+";", "module "+f.module+";", 1)
		text = strings.ReplaceAll(text, "tests::module_body_artifact_cleanup", "closure_program_scalar_cleanup")
		writeTestSource(t, root, "src/"+f.module+".fy", text)
	}
	writeTestSource(t, root, "src/main.fy", main)
	return root, count
}

func TestCandidateClosureProgramArtifactGolden(t *testing.T) {
	raw := closureProgramScalarGolden()
	if len(raw) != 1266 || !bytes.HasPrefix(raw, []byte("FYPROG00")) {
		t.Fatalf("independent old Program oracle width=%d", len(raw))
	}
}

// Compiles/checks the authored fixture only. No native producer or restored
// closure program executes; source-bound Complete and cache are separate owners.
func TestCandidateClosureProgramArtifactSource(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root, count := candidateClosureProgramArtifactWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure Program artifact source: %d %s %s", code, out, diagnostic)
	}
	t.Logf("%d closure controls; exact FYPROG00 oracle plus FYPROG02 roundtrip, version/target/callee mutation and work-bound fixture source checked, not executed", count)
}

// Run only after parent Native CPU coordination. This executes every authored
// fixture assertion, rather than treating Source checking as roundtrip evidence.
func TestCandidateClosureProgramArtifactNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("closure Program artifact native probe requires Windows AMD64; Linux remains separate")
	}
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root, count := candidateClosureProgramArtifactWorkspace(t)
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Logf("%s private compiler fixture: %d parts, existing two-CL bound", profile.Name, len(build.Sources))
		image, err := cdriver.BuildWindowsMSVCCompilerHostPartsWithStackReserve(toolchain, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "closure-program-artifact-"+profile.Name+".exe")
		if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		code, stdout := candidateAtlasProcess(t, path, nil)
		if code != 0 || stdout != "" {
			t.Fatalf("closure Program artifact native %s: exit=%d stdout=%q", profile.Name, code, stdout)
		}
		t.Logf("%s executed: %d closure controls, complete FYPROG00 golden, FYPROG02 roundtrip/forge/budget assertions; exit 0, empty stdout/stderr", profile.Name, count)
	}
}
