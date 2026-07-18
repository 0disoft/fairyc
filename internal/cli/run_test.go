package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestExperimentalParseEmitsDeterministicContentSafeJSON(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\n// SENSITIVE_SENTINEL\nfn main() -> i32 { return 0; }\n")
	firstCode, firstOut, firstErr := runForTest([]string{"experimental", "parse", "main.fy"}, root)
	secondCode, secondOut, secondErr := runForTest([]string{"experimental", "parse", "main.fy"}, root)
	if firstCode != ExitSuccess || secondCode != ExitSuccess {
		t.Fatalf("Run() codes = %d, %d; stderr = %q, %q", firstCode, secondCode, firstErr, secondErr)
	}
	if firstOut != secondOut {
		t.Fatalf("parse output is nondeterministic:\nfirst=%s\nsecond=%s", firstOut, secondOut)
	}
	if strings.Contains(firstOut, "SENSITIVE_SENTINEL") || strings.Contains(firstOut, filepath.ToSlash(root)) {
		t.Fatalf("parse output leaked source content or absolute root: %s", firstOut)
	}
	var report parseReport
	if err := json.Unmarshal([]byte(firstOut), &report); err != nil {
		t.Fatalf("parse output is not JSON: %v\n%s", err, firstOut)
	}
	if report.SchemaVersion != parseReportSchema || report.SourcePath != "main.fy" || report.Status != "ok" {
		t.Fatalf("parse report identity = %+v", report)
	}
	if report.Module != "main" || report.Root < 0 || !report.ASTLowered {
		t.Fatalf("parse report frontend state = %+v", report)
	}
	if len(report.Tokens) == 0 || len(report.Nodes) == 0 || report.Diagnostics == nil {
		t.Fatalf("parse report collections = %+v", report)
	}
	for _, node := range report.Nodes {
		for _, child := range node.Children {
			if child < 0 || child >= len(report.Nodes) {
				t.Fatalf("node %d contains invalid child %d", node.ID, child)
			}
		}
	}
}

func TestExperimentalParseReturnsDiagnosticsAsJSON(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main\n")
	code, stdout, stderr := runForTest([]string{"experimental", "parse", "main.fy"}, root)
	if code != ExitSourceDiagnostics || stderr != "" {
		t.Fatalf("Run() = code %d, stderr %q; want source diagnostics and empty stderr", code, stderr)
	}
	var report parseReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("parse output is not JSON: %v", err)
	}
	if report.Status != "diagnostics" || report.ASTLowered || len(report.Diagnostics) == 0 {
		t.Fatalf("parse report = %+v", report)
	}
	if report.Diagnostics[0].Code != "E0007" || report.Diagnostics[0].Expected != ";" {
		t.Fatalf("first diagnostic = %+v", report.Diagnostics[0])
	}
}

func TestExperimentalCheckSyntaxReportsHumanDiagnostics(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 {\n  return 0\n}\n")
	code, stdout, stderr := runForTest([]string{"experimental", "check-syntax", "main.fy"}, root)
	if code != ExitSourceDiagnostics || stdout != "" {
		t.Fatalf("Run() = code %d, stdout %q; want diagnostics and empty stdout", code, stdout)
	}
	if !strings.Contains(stderr, "main.fy:3:11: E0007: expected \";\"") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestExperimentalCheckSyntaxAcceptsCleanInput(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { return 0; }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "check-syntax", "main.fy"}, root)
	if code != ExitSuccess || stdout != "syntax ok: main.fy\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalCheckSemanticsBuildsTypedHIR(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn choose(flag: bool) -> i32 { if flag { 1 } else { 2 } }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "check-semantics", "main.fy"}, root)
	if code != ExitSuccess || stdout != "semantics ok: main.fy (typed HIR: 1 function(s))\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalCheckSemanticsAcceptsExplicitModuleSet(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nimport math as m;\nfn answer() -> i32 { m::forty_two() }\n")
	writeTestSource(t, root, "math.fy", "module math;\npub fn forty_two() -> i32 { 42 }\n")
	firstCode, firstOut, firstErr := runForTest([]string{"experimental", "check-semantics", "main.fy", "math.fy"}, root)
	secondCode, secondOut, secondErr := runForTest([]string{"experimental", "check-semantics", "math.fy", "main.fy"}, root)
	want := "semantics ok: 2 file(s), 2 module(s) (typed HIR: 2 function(s))\n"
	if firstCode != ExitSuccess || secondCode != ExitSuccess || firstOut != want || secondOut != want || firstErr != "" || secondErr != "" {
		t.Fatalf("Run() first=(%d, %q, %q), second=(%d, %q, %q)", firstCode, firstOut, firstErr, secondCode, secondOut, secondErr)
	}
}

func TestExperimentalCheckSemanticsReportsCrossFileVisibility(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nimport math as m;\nfn answer() -> i32 { m::hidden() }\n")
	writeTestSource(t, root, "math.fy", "module math;\nfn hidden() -> i32 { 42 }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "check-semantics", "main.fy", "math.fy"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E0103: item is not visible from this module") || !strings.Contains(stderr, "declaration: math.fy") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalCheckSemanticsRejectsDuplicateAndOversizedModuleSets(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { 0 }\n")
	duplicateCode, duplicateOut, duplicateErr := runForTest([]string{"experimental", "check-semantics", "main.fy", "main.fy"}, root)
	if duplicateCode != ExitInvalidInvocation || duplicateOut != "" || !strings.Contains(duplicateErr, "listed more than once") {
		t.Fatalf("duplicate Run() = code %d, stdout %q, stderr %q", duplicateCode, duplicateOut, duplicateErr)
	}
	arguments := []string{"experimental", "check-semantics"}
	for range maxSemanticInputFiles + 1 {
		arguments = append(arguments, "main.fy")
	}
	limitCode, limitOut, limitErr := runForTest(arguments, root)
	if limitCode != ExitInvalidInvocation || limitOut != "" || !strings.Contains(limitErr, "accepts at most") {
		t.Fatalf("limit Run() = code %d, stdout %q, stderr %q", limitCode, limitOut, limitErr)
	}
}

func TestExperimentalCheckSemanticsReportsResolveDiagnostic(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn answer() -> i32 { missing }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "check-semantics", "main.fy"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "main.fy:2:22: E0101: unresolved name or path") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalCheckSemanticsRejectsUnsupportedSubset(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn text() -> i32 { \"unsupported\" }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "check-semantics", "main.fy"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "experimental semantic subset does not support literal string") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalCheckSemanticsChecksResourceFlow(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn consume(token: Token) -> i32 { token.value }
fn invalid() -> i32 {
    let token: Token = Token { value: 1 };
    let first: i32 = consume(token);
    consume(token)
}
`)
	code, stdout, stderr := runForTest([]string{"experimental", "check-semantics", "main.fy"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E1101: resource is used after ownership was moved") ||
		!strings.Contains(stderr, "declaration: main.fy") || !strings.Contains(stderr, "move: main.fy") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestRepositorySyntaxExampleStaysRunnable(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	code, stdout, stderr := runForTest(
		[]string{"experimental", "check-syntax", "examples/syntax/main.fy"},
		root,
	)
	if code != ExitSuccess || stdout != "syntax ok: examples/syntax/main.fy\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestRepositorySemanticExampleStaysRunnable(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	code, stdout, stderr := runForTest(
		[]string{"experimental", "check-semantics", "examples/semantic/main.fy", "examples/semantic/math.fy"},
		root,
	)
	want := "semantics ok: 2 file(s), 2 module(s) (typed HIR: 2 function(s))\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestRepositoryResourceExampleStaysRunnable(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	code, stdout, stderr := runForTest(
		[]string{"experimental", "check-semantics", "examples/resources/main.fy"},
		root,
	)
	if code != ExitSuccess || stdout != "semantics ok: examples/resources/main.fy (typed HIR: 2 function(s))\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestStableCheckRemainsReserved(t *testing.T) {
	t.Parallel()
	code, stdout, stderr := runForTest([]string{"check", "main.fy"}, t.TempDir())
	if code != ExitInvalidInvocation || stdout != "" {
		t.Fatalf("Run() = code %d, stdout %q", code, stdout)
	}
	if !strings.Contains(stderr, "reserved for the full semantic pipeline") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestHelpAndInvocationErrorsAreSeparated(t *testing.T) {
	t.Parallel()
	helpCode, helpOut, helpErr := runForTest([]string{"--help"}, t.TempDir())
	if helpCode != ExitSuccess || !strings.Contains(helpOut, "experimental parse") || helpErr != "" {
		t.Fatalf("help = code %d, stdout %q, stderr %q", helpCode, helpOut, helpErr)
	}
	errorCode, errorOut, errorErr := runForTest(nil, t.TempDir())
	if errorCode != ExitInvalidInvocation || errorOut != "" || !strings.Contains(errorErr, "a command is required") {
		t.Fatalf("empty invocation = code %d, stdout %q, stderr %q", errorCode, errorOut, errorErr)
	}
}

func TestInputCannotEscapeSourceRoot(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, parent, "outside.fy", "module outside;\n")
	code, stdout, stderr := runForTest([]string{"experimental", "parse", "../outside.fy"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "escapes the current source root") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestInputSymlinkCannotEscapeSourceRoot(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.fy")
	writeTestSource(t, parent, "outside.fy", "module outside;\n")
	if err := os.Symlink(outside, filepath.Join(root, "link.fy")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	code, stdout, _ := runForTest([]string{"experimental", "parse", "link.fy"}, root)
	if code != ExitInvalidInvocation || stdout != "" {
		t.Fatalf("Run() = code %d, stdout %q; escaped symlink was accepted", code, stdout)
	}
}

func TestOversizedInputFailsBeforeUnboundedRead(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file, err := os.Create(filepath.Join(root, "large.fy"))
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(int64(source.DefaultLimits().MaxBytes + 1)); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runForTest([]string{"experimental", "parse", "large.fy"}, root)
	if code != ExitSourceDiagnostics || stderr != "" {
		t.Fatalf("Run() = code %d, stderr %q", code, stderr)
	}
	var report parseReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "E0010" || report.Status != "diagnostics" || !report.Truncated {
		t.Fatalf("report = %+v", report)
	}
}

func TestOutputFailureReturnsInternalFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\n")
	var stderr bytes.Buffer
	code := Run([]string{"experimental", "parse", "main.fy"}, root, failingWriter{}, &stderr)
	if code != ExitInternalFailure || !strings.Contains(stderr.String(), "write output failed") {
		t.Fatalf("Run() = code %d, stderr %q", code, stderr.String())
	}
}

func runForTest(args []string, root string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(args, root, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func writeTestSource(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
