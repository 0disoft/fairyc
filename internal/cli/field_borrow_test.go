package cli

import (
	"runtime"
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
language = "fairyc-language/14"
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
