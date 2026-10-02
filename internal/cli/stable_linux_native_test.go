package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStableLinuxBuildRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux GNU amd64 and GCC or Clang")
	}
	root := t.TempDir()
	writeNativeWorkspace(t, root, true)
	for _, release := range []bool{false, true} {
		args := []string{"build"}
		if release {
			args = append(args, "--release", "--locked")
		}
		code, out, diag := runForTest(args, root)
		if code != 0 || diag != "" || !strings.HasPrefix(out, "built: build/app (x86_64-linux-gnu, ") {
			t.Fatalf("build: %d %q %q", code, out, diag)
		}
		image := filepath.Join(root, "build", "app")
		before, err := os.ReadFile(image)
		if err != nil || len(before) < 4 || string(before[:4]) != "\x7fELF" {
			t.Fatalf("ELF: %v", err)
		}
		info, err := os.Stat(image)
		if err != nil || info.Mode().Perm()&0100 == 0 {
			t.Fatalf("executable mode: %v", err)
		}
		args[0] = "run"
		code, out, diag = runForTest(args, root)
		if code != 42 || out != "" || diag != "" {
			t.Fatalf("run: %d %q %q", code, out, diag)
		}
		practicalFileEquals(t, image, string(before))
	}
	writeTestSource(t, root, "src/main.fy", "module main;\nfn i32 main() { return missing; }\n")
	image := filepath.Join(root, "build", "app")
	before, _ := os.ReadFile(image)
	code, out, _ := runForTest([]string{"build", "--locked"}, root)
	if code != ExitSourceDiagnostics || out != "" {
		t.Fatalf("invalid source build: %d %q", code, out)
	}
	practicalFileEquals(t, image, string(before))
}

const linuxStableStreams = `module main;
import std::io as io;
@effects(console)
fn i32 main() {
    var u8[1] input = [0];
    assert(switch (io::read_stdin(ref mut input)) { case .ok(n) : { n == 1 && input[0] == 65 } case .err(_) : { false } });
    assert(switch (io::write("한글\n")) { case .ok(_) : { true } case .err(_) : { false } });
    assert(switch (io::write_stderr("diagnostic\n")) { case .ok(_) : { true } case .err(_) : { false } });
    return 131;
}
`

func TestStableLinuxStreamsAndTrap(t *testing.T) {
	root := t.TempDir()
	writeNativeWorkspace(t, root, true)
	writeTestSource(t, root, "src/main.fy", linuxStableStreams)
	code, _, diag := runForTest([]string{"check"}, root)
	if code != 0 {
		t.Fatalf("check: %d %s", code, diag)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("native execution requires Linux GNU amd64")
	}
	for _, release := range []bool{false, true} {
		args := []string{"run", "--locked"}
		if release {
			args = append(args, "--release")
		}
		var out, diag strings.Builder
		code := RunWithInput(args, root, strings.NewReader("A"), &out, &diag)
		if code != 131 || out.String() != "한글\n" || diag.String() != "diagnostic\n" {
			t.Fatalf("streams: %d %q %q", code, out.String(), diag.String())
		}
		out.Reset()
		diag.Reset()
		code = RunWithInput(args, root, strings.NewReader("B"), &out, &diag)
		if code != ExitProgramTrap || out.String() != "" || diag.String() != "trap: assertion-failed\n" {
			t.Fatalf("trap: %d %q %q", code, out.String(), diag.String())
		}
	}
	if _, err := os.Stat(filepath.Join(root, "build")); !os.IsNotExist(err) {
		t.Fatalf("run retained an artifact: %v", err)
	}
}

func TestStableLinuxTaskArgumentsAndWorkingDirectory(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux GNU amd64")
	}
	root := taskManagerWorkspace(t)
	code, out, diag := runForTest([]string{"run", "--", "작업.tsv", "add", "한글 제목"}, root)
	if code != 0 || out != "added id=1\n" || diag != "" {
		t.Fatalf("add: %d %q %q", code, out, diag)
	}
	practicalFileEquals(t, filepath.Join(root, "작업.tsv"), "FYTASK1\t2\n1\t0\t한글 제목\n")
	code, out, diag = runForTest([]string{"run", "--release", "--locked", "--", "작업.tsv", "list"}, root)
	if code != 0 || out != "1\topen\t한글 제목\n" || diag != "" {
		t.Fatalf("list: %d %q %q", code, out, diag)
	}
}
