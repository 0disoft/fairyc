package cli

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestCallTypeDiagnosticsMessage(t *testing.T) {
	for _, test := range []struct{ expected, actual, suffix string }{
		{"bool", "i32", `: expected "bool", got "i32"`},
		{"$p1::left::Index", "$p2::right::Index", `: expected "left::Index", got "right::Index"`},
		{"$p1::ids::Index", "$p2::ids::Index", `: expected "$p1::ids::Index", got "$p2::ids::Index"`},
		{"", "", ""},
	} {
		d := semantic.Diagnostic{Code: "E1006", ExpectedType: hir.Type(test.expected), ActualType: hir.Type(test.actual)}
		if got, want := semanticDiagnosticMessage(d), semanticDiagnosticSummary("E1006")+test.suffix; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestCallTypeDiagnosticsStableJSONFailure(t *testing.T) {
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/bad.fy", "module tests::bad; fn void take(bool flag) {} @test fn void bad() { take(42); }")
	code, stdout, stderr := runForTest([]string{"test", "--json"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, `E1006: incompatible expected and actual types: expected "bool", got "i32"`) || !strings.Contains(stderr, "expected-type:") || !strings.Contains(stderr, "[bytes ") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestCallTypeDiagnosticsExplicitFiles(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "main.fy", "module main; fn void take(bool flag) {} fn void caller() { take(42); }")
	code, stdout, stderr := runForTest([]string{"experimental", "check-semantics", "main.fy"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, `expected "bool", got "i32"`) || !strings.Contains(stderr, "expected-type:") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}
