package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

func TestLowerProgramSourcesBuildsExplicitModuleClosure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nimport helper as h;\nfn main() -> i32 { h::answer() }\n")
	writeTestSource(t, root, "helper.fy", "module helper;\npub fn answer() -> i32 { 42 }\n")
	var stderr bytes.Buffer
	lowered, code := lowerProgramSources([]string{"main.fy", "helper.fy"}, root, &stderr, "test C-emission")
	if code != ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("lowerProgramSources() = code %d, stderr %q", code, stderr.String())
	}
	if lowered.entryName != "main::main" || len(lowered.program.Functions) != 2 {
		t.Fatalf("lowered program = entry %q, functions %d", lowered.entryName, len(lowered.program.Functions))
	}
}

func TestLowerProgramSourcesRejectsDuplicateInput(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { 42 }\n")
	var stderr bytes.Buffer
	_, code := lowerProgramSources([]string{"main.fy", "main.fy"}, root, &stderr, "test C-emission")
	if code != ExitInvalidInvocation || !strings.Contains(stderr.String(), "program source \"main.fy\" is listed more than once") {
		t.Fatalf("lowerProgramSources() = code %d, stderr %q", code, stderr.String())
	}
}

func TestLowerProgramSourcesDoesNotDiscoverUnlistedModule(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nimport helper as h;\nfn main() -> i32 { h::answer() }\n")
	writeTestSource(t, root, "helper.fy", "module helper;\npub fn answer() -> i32 { 42 }\n")
	var stderr bytes.Buffer
	_, code := lowerProgramSources([]string{"main.fy"}, root, &stderr, "test C-emission")
	if code != ExitSourceDiagnostics || !strings.Contains(stderr.String(), "E0101: unresolved name or path") {
		t.Fatalf("lowerProgramSources() = code %d, stderr %q", code, stderr.String())
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
	for range maxExplicitInputFiles + 1 {
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
	writeTestSource(t, root, "main.fy", "module main;\nfn value() -> i32 { 1.5 }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "check-semantics", "main.fy"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "experimental semantic subset does not support literal float") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalCheckSemanticsRejectsNonEmptyResourceDropBody(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nresource Token { value: i32; drop fn(self: &mut Self) { let marker: i32 = 1; } }\nfn main() -> i32 { 0 }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "check-semantics", "main.fy"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "experimental semantic subset does not support non-empty resource drop body") {
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

func TestExperimentalCheckSemanticsReportsRegionCapabilityStorage(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nstruct Holder { scratch: Region; }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "check-semantics", "main.fy"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E3103: compiler-owned Region capability cannot be stored") ||
		!strings.Contains(stderr, "capability: main.fy") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalCheckSemanticsReportsQuestionOnNonResult(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn invalid(input: i32) -> i32 { input? }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "check-semantics", "main.fy"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E4102: postfix ? requires a Result value") ||
		!strings.Contains(stderr, "result-expression: main.fy") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalCheckSemanticsReportsStoredUnusedResult(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", `module main;
fn invalid() -> i32 {
    let pending: Result<i32, i32> = Result::Ok(42);
    0
}
`)
	code, stdout, stderr := runForTest([]string{"experimental", "check-semantics", "main.fy"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "main.fy:3:") ||
		!strings.Contains(stderr, "E4101: Result value must be returned, propagated, matched, or explicitly discarded") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalEmitCWritesDeterministicTranslationUnit(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\n// SOURCE_SENTINEL\nfn add(left: i32, right: i32) -> i32 { left + right }\nfn main() -> i32 { add(20, 22) }\n")
	firstCode, firstOut, firstErr := runForTest([]string{"experimental", "emit-c", "main.fy"}, root)
	secondCode, secondOut, secondErr := runForTest([]string{"experimental", "emit-c", "main.fy"}, root)
	if firstCode != ExitSuccess || secondCode != ExitSuccess || firstErr != "" || secondErr != "" {
		t.Fatalf("Run() first=(%d, %q), second=(%d, %q)", firstCode, firstErr, secondCode, secondErr)
	}
	if firstOut != secondOut {
		t.Fatal("emit-c output is not deterministic")
	}
	for _, required := range []string{
		"/* fairyc constrained C transport: x86_64-linux-gnu, ISO C11 */",
		"_Noreturn void fy_runtime_trap(uint32_t category);",
		"int main(void)",
	} {
		if !strings.Contains(firstOut, required) {
			t.Errorf("emit-c output is missing %q", required)
		}
	}
	if strings.Contains(firstOut, "SOURCE_SENTINEL") || strings.Contains(firstOut, filepath.ToSlash(root)) || strings.Contains(firstOut, "main::add") {
		t.Fatalf("emit-c leaked source text, an absolute path, or a Fairy symbol name:\n%s", firstOut)
	}
}

func TestExperimentalEmitCEmitsScalarForeignCall(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\n@link_name(\"fairyc_fixture_add_one\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { unsafe { add_one(41) } }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "emit-c", "main.fy"}, root)
	if code != ExitSuccess || stderr != "" || !strings.Contains(stdout, "extern int32_t fairyc_fixture_add_one(int32_t fy_a_0);") || !strings.Contains(stdout, "fairyc_fixture_add_one(INT32_C(41))") || strings.Contains(stdout, "extern int32_t add_one(") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunRejectsForeignCallWithoutHostBinding(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\n@link_name(\"fairyc_fixture_add_one\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { unsafe { add_one(41) } }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "outside the experimental interpreter subset") || !strings.Contains(stderr, "cannot execute foreign ABI \"C\" call") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalEmitCEmitsFlatI32Resource(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let token: Token = Token { value: 42 };
    token.value
}
`)
	code, stdout, stderr := runForTest([]string{"experimental", "emit-c", "main.fy"}, root)
	if code != ExitSuccess || stderr != "" || !strings.Contains(stdout, "} fy_resource_0;") || !strings.Contains(stdout, ".f_0;") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if strings.Contains(stdout, "Token") || strings.Contains(stdout, "value") {
		t.Fatalf("emit-c leaked resource source identifiers:\n%s", stdout)
	}
}

func TestExperimentalEmitCEmitsExplicitModuleClosure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nimport helper as h;\nfn main() -> i32 { h::answer() }\n")
	writeTestSource(t, root, "helper.fy", "module helper;\npub fn answer() -> i32 { 42 }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "emit-c", "main.fy", "--source", "helper.fy"}, root)
	if code != ExitSuccess || stderr != "" || strings.Count(stdout, "static int32_t fy_fn_") != 4 || !strings.Contains(stdout, "fy_l_1 = fy_fn_0();") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if strings.Contains(stdout, "helper") || strings.Contains(stdout, "answer") {
		t.Fatalf("emit-c leaked module source identifiers:\n%s", stdout)
	}
}

func TestExperimentalEmitCRejectsUnsupportedBackendSubset(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", `module main;
struct Marker {
    value: u32;
}
resource Holder {
    marker: Marker;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 { 0 }
`)
	code, stdout, stderr := runForTest([]string{"experimental", "emit-c", "main.fy"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "experimental C-emission subset") || !strings.Contains(stderr, "field \"marker\" of type \"main::Marker\"") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunInterpretsScalarProgram(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn add(left: i32, right: i32) -> i32 { left + right }\nfn main() -> i32 { add(20, 22) }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunInterpretsExplicitModuleClosure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nimport helper as h;\nfn main() -> i32 { h::answer() }\n")
	writeTestSource(t, root, "helper.fy", "module helper;\npub fn answer() -> i32 { 42 }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy", "--source", "helper.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestParseRunSourcesRejectsInvalidClosureArguments(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		arguments []string
		contains  string
	}{
		{nil, "expects one entry source file"},
		{[]string{"--source", "helper.fy"}, "entry source file before"},
		{[]string{"main.fy", "--profile", "low"}, "does not accept flag"},
		{[]string{"main.fy", "--source"}, "requires a value for --source"},
		{[]string{"main.fy", "--source", "--target"}, "requires a value for --source"},
	} {
		if _, err := parseRunSources(test.arguments); err == nil || !strings.Contains(err.Error(), test.contains) {
			t.Errorf("parseRunSources(%q) error = %v; want containing %q", test.arguments, err, test.contains)
		}
	}
	overLimit := []string{"main.fy"}
	for index := 0; index < maxExplicitInputFiles; index++ {
		overLimit = append(overLimit, "--source", fmt.Sprintf("module-%d.fy", index))
	}
	if _, err := parseRunSources(overLimit); err == nil || !strings.Contains(err.Error(), "accepts at most 64 source files") {
		t.Fatalf("over-limit source error = %v", err)
	}
}

func TestExperimentalRunDoesNotAuthorizeTestIntrinsicByName(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", `module main;
fn write_stdout_byte(value: i32) -> i32 { value }
fn main() -> i32 { write_stdout_byte(42) }
`)
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunInterpretsResultMatchAndPropagation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", `module main;
fn increment(input: Result<i32, i32>) -> Result<i32, i32> {
    let value: i32 = input?;
    Result::Ok(value + 1)
}
fn main() -> i32 {
    match increment(Result::Ok(41)) {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
`)
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunInterpretsOptionVariants(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", `module main;
fn choose(present: bool) -> Option<i32> {
    if present { Option::Some(42) } else { Option::None }
}
fn main() -> i32 {
    match choose(false) {
        Option::Some(value) => value,
        Option::None => 42,
    }
}
`)
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunReportsCanonicalTrap(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { 42 / 0 }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitProgramTrap || stdout != "" || stderr != "trap: division-by-zero\n" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunReportsCheckedArithmeticTraps(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		expression string
		category   string
	}{
		{name: "integer overflow", expression: "2147483647 + 1", category: "integer-overflow"},
		{name: "invalid shift", expression: "1 >> 32", category: "invalid-shift"},
	}
	for _, fixture := range cases {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { "+fixture.expression+" }\n")
			code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
			if code != ExitProgramTrap || stdout != "" || stderr != "trap: "+fixture.category+"\n" {
				t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
			}
		})
	}
}

func TestExperimentalRunExecutesCheckedLeftShift(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { 1 << 1 }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitSuccess || stdout != "result: 2\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunInterpretsBooleanBranch(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn choose(flag: bool) -> i32 { if flag { 42 } else { 0 } }\nfn main() -> i32 { choose(true) }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunShortCircuitsTrap(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { if (false && (1 / 0 == 0)) || (true || (1 / 0 == 0)) { 42 } else { 0 } }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunInterpretsLogicalNot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { if !false { 42 } else { 0 } }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunInterpretsBoundedLoop(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { var count: i32 = 0; var total: i32 = 0; while count < 6 { total = total + 7; count = count + 1; } total }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunInterpretsLoopControl(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		source string
		value  string
	}{
		{
			name:   "break",
			source: "module main;\nfn main() -> i32 { var count: i32 = 0; while count < 5 { count = count + 1; if count == 2 { break; }; } count }\n",
			value:  "result: 2\n",
		},
		{
			name:   "continue",
			source: "module main;\nfn main() -> i32 { var count: i32 = 0; var total: i32 = 0; while count < 5 { count = count + 1; if count < 3 { continue; }; total = total + count; } total + 30 }\n",
			value:  "result: 42\n",
		},
	}
	for _, fixture := range cases {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeTestSource(t, root, "main.fy", fixture.source)
			code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
			if code != ExitSuccess || stdout != fixture.value || stderr != "" {
				t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
			}
		})
	}
}

func TestExperimentalRunRejectsLoopControlOutsideLoop(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { break; 0 }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E1012: break and continue require an enclosing while loop") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunReportsStepLimitAsCapability(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { while true {} 0 }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "outside the experimental interpreter subset: interpreter step limit exceeded") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunRequiresScalarEntry(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main(value: i32) -> i32 { value }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "main::main must have type fn() -> i32") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunExecutesResourceAndDrop(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let token: Token = Token { value: 42 };
    token.value
}
`)
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunExecutesRegionReset(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "std/alloc.fy", `module std::alloc;

pub resource Arena {
    opaque: i32;
    drop fn(self: &mut Self) {}
}

resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}

fn main() -> i32 {
    var arena: Arena = Arena { opaque: 0 };
    region scratch in &mut arena {
        let token: Token = Token { value: 7 };
        let stored: &i32 = scratch.place(42);
    }
    42
}
`)
	code, stdout, stderr := runForTest([]string{"experimental", "run", "std/alloc.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunExecutesRegionEarlyReturnReset(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "std/alloc.fy", `module std::alloc;

pub resource Arena {
    opaque: i32;
    drop fn(self: &mut Self) {}
}

resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}

fn main() -> i32 {
    var arena: Arena = Arena { opaque: 0 };
    let outside: Token = Token { value: 20 };
    region scratch in &mut arena {
        let inner: Token = Token { value: 22 };
        let stored: &i32 = scratch.place(1);
        return outside.value + inner.value;
    }
}
`)
	code, stdout, stderr := runForTest([]string{"experimental", "run", "std/alloc.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalRunExecutesResourceBearingOption(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let wrapped: Option<Token> = Option::Some(Token { value: 42 });
    match wrapped {
        Option::Some(token) => token.value,
        Option::None => 0,
    }
}
`)
	code, stdout, stderr := runForTest([]string{"experimental", "run", "main.fy"}, root)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
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

func TestRepositoryRegionExampleStaysRunnable(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	code, stdout, stderr := runForTest(
		[]string{"experimental", "check-semantics", "examples/regions/main.fy"},
		root,
	)
	if code != ExitSuccess || stdout != "semantics ok: examples/regions/main.fy (typed HIR: 1 function(s))\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestRepositoryResultExampleStaysRunnable(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	code, stdout, stderr := runForTest(
		[]string{"experimental", "check-semantics", "examples/results/main.fy"},
		root,
	)
	if code != ExitSuccess || stdout != "semantics ok: examples/results/main.fy (typed HIR: 3 function(s))\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestRepositoryInterpreterExampleStaysRunnable(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	code, stdout, stderr := runForTest(
		[]string{"experimental", "run", "examples/run/main.fy"},
		root,
	)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestRepositoryMultiSourceInterpreterExampleStaysRunnable(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	code, stdout, stderr := runForTest(
		[]string{
			"experimental", "run", "examples/run/modules/main.fy",
			"--source", "examples/run/modules/math.fy",
		},
		root,
	)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestRepositoryCEmissionExampleStaysRunnable(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	code, stdout, stderr := runForTest(
		[]string{"experimental", "emit-c", "examples/c/main.fy"},
		root,
	)
	if code != ExitSuccess || stderr != "" || !strings.Contains(stdout, "int main(void)") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestRepositoryTaggedCEmissionExampleStaysRunnable(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	code, stdout, stderr := runForTest(
		[]string{"experimental", "emit-c", "examples/c/tagged.fy"},
		root,
	)
	if code != ExitSuccess || stderr != "" || !strings.Contains(stdout, "fy_option_i32") || !strings.Contains(stdout, "fy_result_i32_i32") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestRepositoryResourceOptionExampleStaysRunnable(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	code, stdout, stderr := runForTest(
		[]string{"experimental", "run", "examples/run/resource_option.fy"},
		root,
	)
	if code != ExitSuccess || stdout != "result: 42\n" || stderr != "" {
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

func TestExplainReportsReservedDiagnosticWithoutReadingSource(t *testing.T) {
	t.Parallel()
	missingRoot := filepath.Join(t.TempDir(), "missing")
	code, stdout, stderr := runForTest([]string{"explain", "E1101"}, missingRoot)
	want := "E1101: Resource used after move or consumption.\n" +
		"Category: Resource ownership and drop\n" +
		"Catalog: fairyc.diagnostic-catalog/v0 (provisional Stage 0)\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExplainCoversAtlasAndVerifierCodes(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		code     string
		category string
	}{
		{code: "A2001", category: "Category: Atlas warnings or hard map-integrity errors\n"},
		{code: "V1403", category: "Category: MIR verifier defects\n"},
	} {
		code, stdout, stderr := runForTest([]string{"explain", testCase.code}, t.TempDir())
		if code != ExitSuccess || !strings.HasPrefix(stdout, testCase.code+": ") || !strings.Contains(stdout, testCase.category) || stderr != "" {
			t.Fatalf("Run(%s) = code %d, stdout %q, stderr %q", testCase.code, code, stdout, stderr)
		}
	}
}

func TestExplainRejectsMissingMalformedUnknownAndExtraCodes(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		args []string
		want string
	}{
		{name: "missing", args: []string{"explain"}, want: "expects exactly one diagnostic code"},
		{name: "lowercase", args: []string{"explain", "e1101"}, want: "is not in canonical"},
		{name: "help-word-is-not-a-code", args: []string{"explain", "help"}, want: "is not in canonical"},
		{name: "unknown", args: []string{"explain", "E9999"}, want: "unknown diagnostic code"},
		{name: "extra", args: []string{"explain", "E1101", "E1102"}, want: "expects exactly one diagnostic code"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			code, stdout, stderr := runForTest(testCase.args, t.TempDir())
			if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, testCase.want) {
				t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
			}
		})
	}
}

func TestExplainHelpFlagsUseGlobalHelp(t *testing.T) {
	t.Parallel()
	for _, flag := range []string{"-h", "--help"} {
		code, stdout, stderr := runForTest([]string{"explain", flag}, t.TempDir())
		if code != ExitSuccess || !strings.Contains(stdout, "fairyc explain <diagnostic-code>") || stderr != "" {
			t.Fatalf("Run(%s) = code %d, stdout %q, stderr %q", flag, code, stdout, stderr)
		}
	}
}

func TestExplainOutputFailureReturnsInternalFailure(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	code := Run([]string{"explain", "E1101"}, t.TempDir(), failingWriter{}, &stderr)
	if code != ExitInternalFailure || !strings.Contains(stderr.String(), "write output failed") {
		t.Fatalf("Run() = code %d, stderr %q", code, stderr.String())
	}
}

func TestHelpAndInvocationErrorsAreSeparated(t *testing.T) {
	t.Parallel()
	helpCode, helpOut, helpErr := runForTest([]string{"--help"}, t.TempDir())
	if helpCode != ExitSuccess || !strings.Contains(helpOut, "fairyc explain") || !strings.Contains(helpOut, "experimental parse") || !strings.Contains(helpOut, "experimental target-spec") || !strings.Contains(helpOut, "experimental emit-c") || !strings.Contains(helpOut, "experimental build") || !strings.Contains(helpOut, "experimental build-target") || !strings.Contains(helpOut, "experimental run-target") || !strings.Contains(helpOut, "experimental exec-target") || !strings.Contains(helpOut, "experimental build-host") || !strings.Contains(helpOut, "experimental run-host") || helpErr != "" {
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
