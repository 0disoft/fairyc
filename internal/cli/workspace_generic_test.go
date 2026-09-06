package cli

import (
	"runtime"
	"strings"
	"testing"
)

func writeGenericCLIWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeNativeWorkspace(t, root, true)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", "module answer; pub fn T value<T>(T input) { return input; }")
	writeWorkspaceFile(t, root, "src/main.fy", "module main; import answer as answer; fn i32 main() { return answer::value<i32>(42); }")
	writeWorkspaceFile(t, root, "tests/main.fy", "module tests::main; import answer as answer; @test fn void generic_answer() { assert(answer::value<i32>(42) == 42); }")
	return root
}

func TestStableGenericWorkspaceCheckAndTest(t *testing.T) {
	root := writeGenericCLIWorkspace(t)
	for _, args := range [][]string{{"check"}, {"check", "--locked"}, {"test"}, {"test", "--locked"}} {
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stderr != "" || stdout == "" {
			t.Fatalf("%v: (%d, %q, %q)", args, code, stdout, stderr)
		}
	}
	writeWorkspaceFile(t, root, "src/main.fy", "module main; import answer as answer; fn i32 main() { return answer::value<Unknown>(42); }")
	for _, args := range [][]string{{"check"}, {"test"}, {"build"}, {"run"}} {
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E4502") {
			t.Fatalf("%v rejection: (%d, %q, %q)", args, code, stdout, stderr)
		}
	}
}

func TestStableOrdinaryWorkspaceKeepsNonGenericNames(t *testing.T) {
	root := t.TempDir()
	writeNativeWorkspace(t, root, true)
	writeWorkspaceFile(t, root, "src/main.fy", "module main; fn i32 _fy_generic_user() { return 42; } fn i32 main() { return _fy_generic_user(); }")
	code, stdout, stderr := runForTest([]string{"check"}, root)
	if code != ExitSuccess || stdout == "" || stderr != "" {
		t.Fatalf("ordinary source: (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestStableGenericNativeCommands(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("stable native commands require Windows amd64")
	}
	root := writeGenericCLIWorkspace(t)
	for _, release := range []bool{false, true} {
		for _, command := range []string{"build", "run", "test"} {
			args := []string{command}
			if command == "test" {
				args = append(args, "--native")
			}
			if release {
				args = append(args, "--release")
			}
			code, stdout, stderr := runForTest(args, root)
			want := ExitSuccess
			if command == "run" {
				want = 42
			}
			if code != want || stderr != "" {
				t.Fatalf("%v: (%d, %q, %q)", args, code, stdout, stderr)
			}
		}
	}
}
