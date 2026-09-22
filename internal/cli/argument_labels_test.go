package cli

import (
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"runtime"
	"testing"
)

func TestArgumentLabelsStableNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC")
	}
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.labels\"\nname = \"labels\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
	writeTestSource(t, root, "src/main.fy", `module main;
resource Item { i32 number; cleanup { assert(this.number == 42); } }
fn i32 take(Item value) { return value.number; }
struct Counter { i32 value; } fn i32 bump(ref mut Counter n) { n.value = n.value + 1; return n.value; }
fn i32 pair(i32 left, i32 right) { return left * 10 + right; }
fn usize expression(usize minimum, usize depth, bool allow_record) { return if (allow_record) { minimum + depth } else { depth }; }
fn i32 main() {
    assert(expression(.minimum = 1, .depth = 0, .allow_record = true) == 1);
    var Counter n = Counter { .value = 0 };
    assert(pair(.left = bump(ref mut n), .right = bump(ref mut n)) == 12);
    let Item value = Item { .number = 42 };
    assert(take(.value = value) == 42);
    return 0;
}`)
	for _, args := range [][]string{{"check"}, {"run", "--locked"}, {"run", "--release", "--locked"}} {
		code, out, err := runForTest(args, root)
		if code != ExitSuccess || err != "" {
			t.Fatalf("%v: %d %q %s", args, code, out, err)
		}
	}
}
