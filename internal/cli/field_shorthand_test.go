package cli

import (
	"runtime"
	"strings"
	"testing"
)

func TestFieldShorthandStableNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	manifest := `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.shorthand"
name = "shorthand"
version = "1.0.0"
language = "fairyc-language/18"
source = "src"
`
	writeTestSource(t, root, "fairy.toml", manifest)
	writeTestSource(t, root, "src/main.fy", `module main;
struct Pair { i32 left; i32 right; }
resource Leaf { i32 number; cleanup { assert(this.number == 42); } }
resource Owner { Leaf value; cleanup {} }
fn i32 main() {
    let i32 left = 40;
    let i32 right = 2;
    let Pair pair = Pair { .left, .right };
    let i32 number = pair.left + pair.right;
    let Leaf value = Leaf { .number };
    let Owner owner = Owner { .value };
    assert(number == 42);
    return 0;
}
`)
	for _, args := range [][]string{{"check"}, {"build", "--locked"}, {"run", "--locked"}, {"run", "--release", "--locked"}} {
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stderr != "" || (args[0] == "run" && stdout != "") {
			t.Fatalf("%v: %d %q %s", args, code, stdout, stderr)
		}
	}
}

func TestFieldShorthandRejectsOldManifest(t *testing.T) {
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.old-shorthand"
name = "old-shorthand"
version = "1.0.0"
language = "fairyc-language/14"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main() { return 0; }")
	code, _, stderr := runForTest([]string{"check"}, root)
	if code == ExitSuccess || !strings.Contains(stderr, "language") {
		t.Fatalf("old exact pin accepted: %d %s", code, stderr)
	}
}
