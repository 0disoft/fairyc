package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func cliWorkspace(t *testing.T, text string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	manifest := "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"example.review-cli\"\nname = \"review-cli\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n"
	if err := os.WriteFile(filepath.Join(root, "fairy.toml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.fy"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestReviewRejectsInvalidArgumentsAndDoesNotWriteOnOutputFailure(t *testing.T) {
	for _, args := range [][]string{nil, {"unknown"}, {"diff"}, {"diff", "--before", "x"}, {"notice", "--threshold-bytes", "0"}, {"notice", "--format", "xml"}, {"notice", "extra"}} {
		var out, stderr bytes.Buffer
		if code := run(context.Background(), args, &out, &stderr); code != 2 || out.Len() != 0 {
			t.Fatalf("arguments %v: exit=%d stdout=%q", args, code, out.String())
		}
	}
	if code := run(context.Background(), []string{"notice", "--root", t.TempDir()}, errorWriter{}, &bytes.Buffer{}); code != 2 {
		t.Fatal("output error ignored")
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("test output failure") }

func TestWindowsNativeReviewCLI(t *testing.T) {
	tool := filepath.Join(t.TempDir(), "fairyc-review.exe")
	command := exec.Command("go", "build", "-mod=readonly", "-o", tool, ".")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build review executable: %v %s", err, output)
	}
	before := cliWorkspace(t, "module main; pub fn i32 value(i32 x){return x;} fn i32 main(){return 0;}")
	after := cliWorkspace(t, "module main; @effects(console) pub fn i32 value(i32 x){return x;} fn i32 main(){return 0;}")
	invalid := cliWorkspace(t, "module main; pub fn i32 value(){return false;}")
	for _, test := range []struct {
		name     string
		args     []string
		exit     int
		contains string
	}{
		{"changed", []string{"diff", "--before", before, "--after", after, "--format", "json", "--fail-on-change"}, 1, `"effectsAdded": [`},
		{"unchanged", []string{"diff", "--before", before, "--after", before, "--format", "json"}, 0, `"changed": false`},
		{"invalid", []string{"diff", "--before", before, "--after", invalid, "--format", "json"}, 2, `"changed": null`},
		{"text", []string{"diff", "--before", before, "--after", after}, 0, "+console"},
		{"help", []string{"--help"}, 0, "fairyc-review diff"},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := exec.Command(tool, test.args...)
			command.Dir = t.TempDir()
			output, err := command.CombinedOutput()
			code := 0
			if err != nil {
				var exit *exec.ExitError
				if !errors.As(err, &exit) {
					t.Fatal(err)
				}
				code = exit.ExitCode()
			}
			if code != test.exit || !strings.Contains(string(output), test.contains) {
				t.Fatalf("native CLI: exit=%d expected=%d err=%v output=%s", code, test.exit, err, output)
			}
			if strings.Contains(strings.Join(test.args, " "), "json") {
				var value map[string]any
				if err := json.Unmarshal(output, &value); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	large := filepath.Join(after, "src", "large.fy")
	if err := os.WriteFile(large, []byte(strings.Repeat("?", 40001)), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(tool, "notice", "--root", after, "--format", "json").CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"bytes": 40001`)) {
		t.Fatalf("advisory notice failed: %v %s", err, output)
	}
	for _, root := range []string{before, after, invalid} {
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			t.Fatalf("review wrote into workspace: %v", entries)
		}
	}
	contents, err := os.ReadFile(large)
	if err != nil || len(contents) != 40001 {
		t.Fatal("source truncated or removed")
	}
	t.Log("Windows executable: changed=1, unchanged=0, incomplete=2, text/help/notice=0; no workspace writes")
}
