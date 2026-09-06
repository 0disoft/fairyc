package cli

import (
	"runtime"
	"strings"
	"testing"
)

func writeExplicitGenericFiles(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main; import helper as helper; fn i32 main() { return helper::twice<i32>(40); }")
	writeTestSource(t, root, "helper.fy", "module helper; fn T step<T>(T input) { return input + 1; } pub fn T twice<T>(T input) { return step<T>(step<T>(input)); }")
	return root
}

func TestExplicitGenericCLICommands(t *testing.T) {
	root := writeExplicitGenericFiles(t)
	for _, args := range [][]string{
		{"experimental", "check-semantics", "main.fy", "helper.fy"},
		{"experimental", "check", "main.fy", "--source", "helper.fy"},
		{"experimental", "emit-c", "main.fy", "--source", "helper.fy"},
		{"experimental", "run", "main.fy", "--source", "helper.fy"},
	} {
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stdout == "" || stderr != "" {
			t.Fatalf("%v: (%d, %q, %q)", args, code, stdout, stderr)
		}
		if args[1] == "run" && stdout != "result: 42\n" {
			t.Fatalf("interpreter: %q", stdout)
		}
	}
	writeTestSource(t, root, "main.fy", "module main; import helper as helper; fn i32 main() { return helper::twice<Unknown>(40); }")
	for _, args := range [][]string{
		{"experimental", "check-semantics", "main.fy", "helper.fy"},
		{"experimental", "run", "main.fy", "--source", "helper.fy"},
	} {
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E4502") || !strings.Contains(stderr, "main.fy") {
			t.Fatalf("%v: (%d, %q, %q)", args, code, stdout, stderr)
		}
	}
}

func TestExplicitGenericMissingImportKeepsResolveDiagnostic(t *testing.T) {
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main; import absent as absent; fn T id<T>(T input) { return input; } fn i32 main() { return id<i32>(42); }")
	code, stdout, stderr := runForTest([]string{"experimental", "check", "main.fy"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E0101") {
		t.Fatalf("missing import: (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestExplicitGenericNativeExecution(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native explicit-file proof requires Windows amd64")
	}
	root := writeExplicitGenericFiles(t)
	for _, profile := range []string{"low", "optimized"} {
		args := []string{"experimental", "exec-target", "main.fy", "--source", "helper.fy", "--target", "x86_64-windows-msvc", "--profile", profile}
		code, stdout, stderr := runForTest(args, root)
		if code != 42 || stdout != "" || stderr != "" {
			t.Fatalf("%s: (%d, %q, %q)", profile, code, stdout, stderr)
		}
	}
}
