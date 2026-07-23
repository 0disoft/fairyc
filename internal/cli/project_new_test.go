package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStableNewCreatesAWorkspaceThatChecksAndTests(t *testing.T) {
	parent := t.TempDir()
	code, stdout, stderr := runForTest([]string{"new", "hello-fairy", "--uid", "example.hello-fairy"}, parent)
	if code != ExitSuccess || stdout != "created Fairy C package \"hello-fairy\" at hello-fairy\n" || stderr != "" {
		t.Fatalf("new = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	root := filepath.Join(parent, "hello-fairy")

	code, stdout, stderr = runForTest([]string{"check"}, root)
	if code != ExitSuccess || !strings.HasPrefix(stdout, "check ok: ") || !strings.HasSuffix(stdout, "lock updated\n") || stderr != "" {
		t.Fatalf("generated check = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "fairy.lock")); err != nil {
		t.Fatalf("generated check did not publish fairy.lock: %v", err)
	}
	code, stdout, stderr = runForTest([]string{"check", "--locked"}, root)
	if code != ExitSuccess || !strings.HasSuffix(stdout, "lock verified\n") || stderr != "" {
		t.Fatalf("generated locked check = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"test", "--locked"}, root)
	if code != ExitSuccess || stdout != "test tests::main::answer_is_42 ... ok\ntest result: ok. 1 passed; 0 failed; lock verified\n" || stderr != "" {
		t.Fatalf("generated test = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	const fenceStart = "```fairyc\n"
	start := strings.Index(string(readme), fenceStart)
	if start < 0 {
		t.Fatal("generated README is missing its Fairy C source fence")
	}
	start += len(fenceStart)
	end := strings.Index(string(readme)[start:], "\n```\n")
	if end < 0 {
		t.Fatal("generated README has an unterminated Fairy C source fence")
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.fy"), []byte(string(readme)[start:start+end]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runForTest([]string{"check"}, root)
	if code != ExitSuccess || !strings.HasPrefix(stdout, "check ok: ") || !strings.HasSuffix(stdout, "lock updated\n") || stderr != "" {
		t.Fatalf("generated README console example check = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestStableNewRejectsInvalidArgumentsBeforeFilesystemAccess(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	for _, test := range []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "missing", arguments: nil, contains: "expects <package-name>"},
		{name: "missing uid", arguments: []string{"hello"}, contains: "expects <package-name>"},
		{name: "wrong flag", arguments: []string{"hello", "--name", "example.hello"}, contains: "expects <package-name>"},
		{name: "extra", arguments: []string{"hello", "--uid", "example.hello", "extra"}, contains: "expects <package-name>"},
		{name: "invalid name", arguments: []string{"Bad_Name", "--uid", "example.hello"}, contains: "lowercase kebab-case"},
		{name: "reserved name", arguments: []string{"con", "--uid", "example.hello"}, contains: "lowercase kebab-case"},
		{name: "invalid uid", arguments: []string{"hello", "--uid", "single"}, contains: "dot-separated"},
	} {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := runForTest(append([]string{"new"}, test.arguments...), missingRoot)
			if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, test.contains) || strings.Contains(stderr, missingRoot) {
				t.Fatalf("new = code %d, stdout %q, stderr %q", code, stdout, stderr)
			}
		})
	}
}

func TestStableNewReportsMissingParentWithoutLeakingIt(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "missing-parent")
	code, stdout, stderr := runForTest([]string{"new", "hello", "--uid", "example.hello"}, parent)
	if code != ExitPackageFailure || stdout != "" ||
		stderr != "fairyc: project creation failed: project parent directory is unavailable\n" ||
		strings.Contains(stderr, parent) {
		t.Fatalf("new = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestStableNewOutputFailureKeepsCompletedProject(t *testing.T) {
	parent := t.TempDir()
	var stderr strings.Builder
	code := Run([]string{"new", "hello", "--uid", "example.hello"}, parent, failingWriter{}, &stderr)
	if code != ExitInternalFailure || !strings.Contains(stderr.String(), "write output failed") {
		t.Fatalf("new = code %d, stderr %q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(parent, "hello", "src", "main.fy")); err != nil {
		t.Fatalf("completed project missing after output failure: %v", err)
	}
}

func TestStableNewRejectsExistingTargetWithoutChangingIt(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "hello")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(target, "marker")
	if err := os.WriteFile(marker, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runForTest([]string{"new", "hello", "--uid", "example.hello"}, parent)
	if code != ExitPackageFailure || stdout != "" || stderr != "fairyc: project creation failed: project path \"hello\" already exists\n" {
		t.Fatalf("new = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	contents, err := os.ReadFile(marker)
	if err != nil || string(contents) != "unchanged" {
		t.Fatalf("existing target changed: contents %q, error %v", contents, err)
	}
}
