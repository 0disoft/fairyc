package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type shortWriter struct{}

func (shortWriter) Write(value []byte) (int, error) {
	return len(value) / 2, nil
}

const cliFixture = `module wallet::token;
extern contract Token {
    @view
    fn u64 balance_of(Address owner);
}
@event struct Transfer { Address from; Address to; u64 amount; }
`

func TestRunEmitsJSONIDL(t *testing.T) {
	t.Parallel()
	path := writeFixture(t, "token.fy", cliFixture)
	var stdout, stderr bytes.Buffer
	code := run([]string{path}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("run() code = %d, stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run() stderr = %q", stderr.String())
	}
	for _, fragment := range []string{`"schema": "fairyc.contract-idl/v0"`, `"module": "wallet::token"`, `"external_call"`} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("run() output does not contain %q:\n%s", fragment, stdout.String())
		}
	}
}

func TestRunEmitsTypeScriptAndGoBindings(t *testing.T) {
	t.Parallel()
	path := writeFixture(t, "token.fy", cliFixture)
	tests := []struct {
		name     string
		args     []string
		contains string
	}{
		{name: "typescript", args: []string{"--format", "ts", path}, contains: "export class TokenClient"},
		{name: "go", args: []string{"--format", "go", "--package", "walletbindings", path}, contains: "package walletbindings"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			code := run(test.args, &stdout, &stderr)
			if code != exitSuccess {
				t.Fatalf("run() code = %d, stderr = %q", code, stderr.String())
			}
			if !strings.Contains(stdout.String(), test.contains) {
				t.Fatalf("run() output does not contain %q:\n%s", test.contains, stdout.String())
			}
		})
	}
}

func TestRunReportsSourceDiagnostics(t *testing.T) {
	t.Parallel()
	path := writeFixture(t, "invalid.fy", "module main; extern contract Token { fn void call(ref Address owner); }")
	var stdout, stderr bytes.Buffer
	code := run([]string{path}, &stdout, &stderr)
	if code != exitSourceDiagnostics {
		t.Fatalf("run() code = %d, want %d; stderr = %q", code, exitSourceDiagnostics, stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "CIDL1007") || !strings.Contains(stderr.String(), path+":1:") {
		t.Fatalf("run() stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestRunRejectsInvalidInvocation(t *testing.T) {
	t.Parallel()
	tests := [][]string{
		nil,
		{"--format", "evm", "input.fy"},
		{"--format", "go", "--package", "Bad-Package", "input.fy"},
		{"input.txt"},
		{"--format", "go", "--package", strings.Repeat("a", 129), "input.fy"},
	}
	for _, args := range tests {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != exitInvalidInvocation {
			t.Fatalf("run(%v) code = %d, stderr = %q", args, code, stderr.String())
		}
	}
}

func TestRunHelp(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("run(--help) code = %d", code)
	}
	if !strings.Contains(stderr.String(), "fairyc-contract-idl") {
		t.Fatalf("run(--help) stderr = %q", stderr.String())
	}
}

func TestRunRejectsShortOutputWrite(t *testing.T) {
	t.Parallel()
	path := writeFixture(t, "token.fy", cliFixture)
	if code := run([]string{path}, shortWriter{}, io.Discard); code != exitInternalFailure {
		t.Fatalf("run(short output) code = %d, want %d", code, exitInternalFailure)
	}
}

func writeFixture(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
