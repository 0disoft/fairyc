package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestCallTypeDiagnosticsScannerDeclarations(t *testing.T) {
	root, _ := tokenIndexWorkspace(t, true)
	data, err := os.ReadFile(filepath.Join(root, "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(data), "token_info(ref stream, token_index)", "token_info(ref stream, byte_offset)", 1)
	if text == string(data) {
		t.Fatal("call mutation missed")
	}
	writeTestSource(t, root, "src/main.fy", text)
	code, stdout, stderr := runForTest([]string{"check"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, `expected "scanner::TokenIndex", got "scanner::ByteOffset"`) {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	scanner, err := os.ReadFile(filepath.Join(root, "src", "scanner.fy"))
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ role, name string }{
		{"expected-type-declaration", "TokenIndex"}, {"actual-type-declaration", "ByteOffset"},
	} {
		start := strings.Index(string(scanner), "struct "+item.name) + len("struct ")
		line, column := bytePosition(scanner, start)
		want := fmt.Sprintf("%s: scanner.fy:%d:%d [bytes %d..%d]", item.role, line, column, start, start+len(item.name))
		if !strings.Contains(stderr, want) {
			t.Fatalf("missing %q in %s", want, stderr)
		}
	}
}

func TestCallTypeDiagnosticsStructJSONFailure(t *testing.T) {
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "src/left.fy", "module left; pub struct Index { usize raw; } pub fn void take(Index value) {}")
	writeWorkspaceFile(t, root, "src/right.fy", "module right; pub struct Index { usize raw; }")
	writeWorkspaceFile(t, root, "tests/bad.fy", "module tests::bad; import left as l; import right as r; @test fn void bad() { l::take(r::Index { .raw = 0 }); }")
	code, stdout, stderr := runForTest([]string{"test", "--json"}, root)
	if code != ExitSourceDiagnostics || stdout != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, want := range []string{`expected "left::Index", got "right::Index"`, "expected-type:", "expected-type-declaration: left.fy:", "actual-type-declaration: right.fy:"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("missing %q in %s", want, stderr)
		}
	}
}

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
