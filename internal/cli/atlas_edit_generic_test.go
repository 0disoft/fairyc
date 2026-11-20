package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/sourceedit"
)

func TestAtlasEditGenericKeepsConcreteCoverage(t *testing.T) {
	const uid = "example.session.app"
	const original = `module main;
fn T transform<T>(T value) { return value; }
fn T unused<T>(T value) { return value; }
fn bool flag() { return transform<bool>(true); }
fn i32 main() {
 if (flag()) { return transform<i32>(2); }
 return 0;
}
`
	const body = "{ let T copy = value; return copy; }"
	root := writeWorkspaceSessionFixture(t, original)
	catalog := func(name string) string {
		t.Helper()
		code, out, stderr := runForTest([]string{"experimental", "atlas-symbols", "list",
			"--catalog-version", "3", "--package-uid", uid, "--kind", "function", "--name", name,
			"--max-symbols", "4", "--max-selection-bytes", "1048576", "--max-output-bytes", "2097152"}, root)
		if code != ExitSuccess || stderr != "" {
			t.Fatalf("catalog %s: %d %s", name, code, stderr)
		}
		records := decodeAtlasSymbols(t, out).Catalog.Symbols
		if len(records) != 1 {
			t.Fatalf("catalog %s: %s", name, out)
		}
		return records[0].Symbol.Visible
	}
	target, unused, flag := catalog("transform"), catalog("unused"), catalog("flag")
	hash := sourceedit.Digest([]byte(original))
	args := func(symbol string) []string {
		return []string{"experimental", "atlas-edit", "replace-body", "--package-uid", uid,
			"--symbol", symbol, "--expected-source-sha256", hash, "--body-file", "edits/body.txt"}
	}
	assertRejected := func(arguments []string, reason string) {
		t.Helper()
		if code, out, stderr := runForTest(arguments, root); code != ExitInvalidInvocation || out != "" || !strings.Contains(stderr, reason) {
			t.Fatalf("expected %q rejection, got %d %q %q", reason, code, out, stderr)
		}
	}
	writeWorkspaceFile(t, root, "edits/body.txt", "{ return 1; }")
	assertRejected(args(target), "failed semantic/MIR validation")
	writeWorkspaceFile(t, root, "edits/body.txt", body)
	assertRejected(args(unused), "requires a current concrete specialization")

	writeWorkspaceFile(t, root, "edits/flag.txt", "{ return true; }")
	request, err := json.Marshal(map[string]any{"edits": []map[string]string{
		{"symbol": target, "bodyFile": "edits/body.txt", "expectedSourceSha256": hash},
		{"symbol": flag, "bodyFile": "edits/flag.txt", "expectedSourceSha256": hash},
	}})
	if err != nil {
		t.Fatal(err)
	}
	writeWorkspaceFile(t, root, "edits/batch.json", string(request))
	assertRejected([]string{"experimental", "atlas-edit", "apply-batch", "--package-uid", uid,
		"--edit-file", "edits/batch.json"}, "removed a required generic specialization")
	file := filepath.Join(root, "src", "main.fy")
	if data, err := os.ReadFile(file); err != nil || string(data) != original {
		t.Fatalf("rejection changed source: %v", err)
	}

	code, out, stderr := runForTest(append(args(target), "--apply"), root)
	var report sourceedit.Report
	if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &report) != nil || !report.Verified || !report.Applied {
		t.Fatalf("generic apply: %d %s %s", code, out, stderr)
	}
	want := strings.Replace(original, "{ return value; }", body, 1)
	if data, err := os.ReadFile(file); err != nil || string(data) != want || report.AfterSHA256 != sourceedit.Digest(data) {
		t.Fatalf("unexpected applied source: %v", err)
	}
}
