package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractprofile"
)

func TestRunEmitsAcceptedContractProfileReport(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeSource(t, root, "main.fy", `module main;

@max_iter(4)
fn i32 main() {
    var i32 value = 0;
    while (value < 2) {
        value += 1;
    }
    return value;
}
`)
	code, stdout, stderr := runForTest([]string{"main.fy"}, root)
	if code != exitSuccess || stderr != "" {
		t.Fatalf("run = %d, stdout=%q, stderr=%q", code, stdout, stderr)
	}
	var report contractprofile.Report
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout)
	}
	if !report.Accepted || report.Profile != contractprofile.ProfileID || report.SchemaVersion != contractprofile.ReportSchema {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunReturnsProfileRejectedExitWithJSON(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeSource(t, root, "main.fy", `module main;

fn i32 main() {
    var i32 value = 0;
    while (value < 2) {
        value += 1;
    }
    return value;
}
`)
	code, stdout, stderr := runForTest([]string{"main.fy"}, root)
	if code != exitProfileRejected || stderr != "" {
		t.Fatalf("run = %d, stdout=%q, stderr=%q", code, stdout, stderr)
	}
	var report contractprofile.Report
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout)
	}
	if report.Accepted || report.Status != "profile-rejected" {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunReturnsSourceDiagnosticsExitWithJSON(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeSource(t, root, "main.fy", "module main;\nfn i32 main() { return 0 }\n")
	code, stdout, stderr := runForTest([]string{"main.fy"}, root)
	if code != exitSourceDiagnostics || stderr != "" {
		t.Fatalf("run = %d, stdout=%q, stderr=%q", code, stdout, stderr)
	}
	var report contractprofile.Report
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout)
	}
	if report.Status != "source-diagnostics" || len(report.SourceDiagnostics) == 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunRejectsEscapingAndDuplicateSourcePaths(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeSource(t, root, "main.fy", "module main;\nfn i32 main() { return 0; }\n")
	for _, arguments := range [][]string{
		{"../outside.fy"},
		{"main.fy", "./main.fy"},
		{"main.txt"},
		{"--source"},
	} {
		code, stdout, stderr := runForTest(arguments, root)
		if code != exitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "invalid invocation") {
			t.Fatalf("run(%q) = %d, stdout=%q, stderr=%q", arguments, code, stdout, stderr)
		}
	}
}

func TestRunHelpDoesNotTouchSourceRoot(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--help"}, filepath.Join(t.TempDir(), "missing"), &stdout, &stderr)
	if code != exitSuccess || stdout.String() != usage || stderr.String() != "" {
		t.Fatalf("run = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
	}
}

func runForTest(arguments []string, root string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(arguments, root, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func writeSource(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
