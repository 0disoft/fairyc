package cli

import (
	"runtime"
	"strings"
	"testing"
)

func TestWindowsSharedFieldBorrow(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.field-borrow"
name = "fields"
version = "1.0.0"
language = "fairyc-language/16"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", `module main;
struct Inner { i32 number; }
resource Owner { Inner inner; cleanup {} }
fn ref i32 number(ref Owner value) { return ref value.inner.number; }
fn i32 main() {
    let Owner owner = Owner { .inner = Inner { .number = 42 } };
    let ref i32 direct = ref owner.inner.number;
    assert(*direct == 42);
    let ref i32 returned = number(ref owner);
    assert(*returned == 42);
    return 0;
}
`)
	for _, release := range []bool{false, true} {
		args := []string{"run"}
		if release {
			args = append(args, "--release", "--locked")
		}
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stdout != "" || stderr != "" {
			t.Fatalf("field borrow release=%v: %d %q %s", release, code, stdout, stderr)
		}
	}
}

func TestFieldBorrowAssignmentFailsClosed(t *testing.T) {
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.field-assignment"
name = "fields"
version = "1.0.0"
language = "fairyc-language/16"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", `module main;
struct Value { i32 number; }
struct State { Value count; }
fn void edit(ref mut State state) { state.count = Value { .number = 1 }; }
fn i32 main() { var State state = State { .count = Value { .number = 0 } }; edit(ref mut state); return 0; }
`)
	for _, command := range []string{"check", "build", "run"} {
		code, stdout, stderr := runForTest([]string{command}, root)
		if code == ExitSuccess || stdout != "" || !strings.Contains(stderr, "field assignment without executable field storage") {
			t.Fatalf("%s accepted field assignment: code=%d stdout=%q stderr=%q", command, code, stdout, stderr)
		}
	}
}
