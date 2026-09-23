package cli

import (
	"runtime"
	"strings"
	"testing"
)

func TestWindowsFieldStoreStableCLI(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.field-store"
name = "fields"
version = "1.0.0"
language = "fairyc-language/18"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", `module main;
struct State { i32 count; usize size; bool ready; }
resource Owner { i32 count; cleanup {} }
resource Holder { State state; cleanup {} }
struct Unsigned { u8 byte; u32 word; }
resource OwnedUnsigned { u8 byte; u32 word; cleanup {} }
fn void fill(ref mut Unsigned value) { value.byte = 255; value.word = 4294967295; }
fn void State::set(ref mut State this) { this.count = 42; this.size = 9; this.ready = true; }
fn i32 main() {
    var State original = State { .count = 1, .size = 2, .ready = false };
    var State copy = original;
    copy.count = 5;
    assert(original.count == 1);
    copy.set();
    assert(copy.count == 42 && copy.size == 9 && copy.ready);
    assert(original.count == 1 && original.size == 2 && !original.ready);
    let Holder holder = Holder { .state = original };
    original.count = 8;
    assert(holder.state.count == 1);
    var Owner owner = Owner { .count = 0 };
    owner.count = copy.count;
    assert(owner.count == 42);
    var Unsigned original_unsigned = Unsigned { .byte = 0, .word = 0 };
    var Unsigned copied_unsigned = original_unsigned;
    fill(ref mut copied_unsigned);
    assert(copied_unsigned.byte == 255 && copied_unsigned.word == 4294967295);
    assert(original_unsigned.byte == 0 && original_unsigned.word == 0);
    var OwnedUnsigned owned_unsigned = OwnedUnsigned { .byte = 0, .word = 0 };
    owned_unsigned.byte = copied_unsigned.byte; owned_unsigned.word = copied_unsigned.word;
    copied_unsigned.byte = 0; copied_unsigned.word = 0;
    assert(owned_unsigned.byte == 255 && owned_unsigned.word == 4294967295);
    return 0;
}`)
	for _, args := range [][]string{{"check"}, {"run"}, {"run", "--release", "--locked"}} {
		code, stdout, stderr := runForTest(args, root)
		validOutput := stdout == ""
		if args[0] == "check" {
			validOutput = strings.HasPrefix(stdout, "check ok:")
		}
		if code != ExitSuccess || !validOutput || stderr != "" {
			t.Fatalf("%v: %d %q %s", args, code, stdout, stderr)
		}
	}
}
