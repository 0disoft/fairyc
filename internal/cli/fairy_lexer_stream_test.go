package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func copyLexerStreamExample(t *testing.T, root string) {
	t.Helper()
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	for _, name := range []string{"fairy.toml", "src/scanner.fy", "src/numbers.fy", "src/diagnostics.fy", "src/utf8.fy", "src/strings.fy", "src/budget.fy", "src/types.fy", "src/records.fy", "src/bodies.fy", "src/body_state.fy", "src/cleanup_syntax.fy", "src/parse_tokens.fy", "src/attributes.fy"} {
		data, err := os.ReadFile(filepath.Join(example, name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, name, string(data))
	}
}

func TestWindowsFairyLexerStreamBorrowBoundaries(t *testing.T) {
	for _, test := range []struct{ name, code, body string }{
		{"local escape", "E2102", "fn ref u8[] bad(scanner::Stream stream) { return scanner::source_view(ref stream); }"},
		{"use after release", "E1101", "fn void bad(scanner::Stream stream) { scanner::release(stream); let usize count = scanner::token_count(ref stream); assert(count == 0); }"},
		{"release while borrowed", "E2105", "fn void bad(scanner::Stream stream) { let ref u8[] view = scanner::source_view(ref stream); scanner::release(stream); assert(view.len == 0); }"},
		{"cooked escape", "E2102", "fn ref u8[] bad(scanner::Stream stream) { return scanner::token_cooked(ref stream, 0); }"},
		{"release while cooked borrowed", "E2105", "fn void bad(scanner::Stream stream) { let ref u8[] view = scanner::token_cooked(ref stream, 0); scanner::release(stream); assert(view.len == 0); }"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerStreamExample(t, root)
			writeTestSource(t, root, "src/main.fy", "module main; import scanner as scanner; "+test.body+" fn i32 main() { return 0; }")
			code, stdout, stderr := runForTest([]string{"check"}, root)
			if code == ExitSuccess || stdout != "" || !strings.Contains(stderr, test.code) {
				t.Fatalf("%d %q %s", code, stdout, stderr)
			}
		})
	}
}

func TestWindowsFairyLexerStream(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	writeTestSource(t, root, "src/main.fy", `module main;
import scanner as scanner;
import diagnostics as diagnostics;
import std::bytes as bytes;
fn Result<scanner::Stream, i32> make(ref u8[] input, usize limit) {
    var bool allocated = true;
    let bytes::Buffer owned = switch (bytes::copy(input)) { case .ok(value): { value } case .err(_): { allocated = false; bytes::empty() } };
    if (!allocated) { return .err(3); }
    return scanner::collect(owned, false, 16384, limit, 256, 16777216);
}
fn i32 check(scanner::Stream stream) {
    let scanner::Storage storage = scanner::storage(ref stream);
    assert(storage.source_bytes == 7 && storage.cooked_bytes == 2 && storage.diagnostic_bytes == 0);
    assert(scanner::token_count(ref stream) == 4 && scanner::diagnostic_count(ref stream) == 0);
    let scanner::TokenInfo first = scanner::token_info(ref stream, 0);
    assert(first.kind == 4 && first.start == 0 && first.end == 4);
    let ref u8[] cooked = scanner::token_cooked(ref stream, 0);
    assert(cooked.len == 4 && cooked[0] == 116 && cooked[3] == 101);
    let ref u8[] number = scanner::token_cooked(ref stream, 2);
    assert(number.len == 2 && number[0] == 52 && number[1] == 50);
    let ref u8[] raw = scanner::token_raw(ref stream, 2);
    assert(raw.len == 2 && raw[0] == 52);
    let scanner::Stream moved = stream;
    assert(scanner::token_count(ref moved) == 4);
    return 0;
}
fn i32 check_limit(scanner::Stream limited) {
    assert(scanner::diagnostic_count(ref limited) == 1);
    let diagnostics::Info error = scanner::diagnostic_info(ref limited, 0);
    assert(error.code == 10);
    let ref u8[] source = scanner::source_view(ref limited);
    assert(source.len == 7 && source[6] == 50);
    return 0;
}
fn i32 main() {
    let i32 normal = switch (make(b"true 42", 1048576)) { case .ok(value): { check(value) } case .err(_): { 1 } };
    if (normal != 0) { return normal; }
    return switch (make(b"true 42", 0)) { case .ok(value): { check_limit(value) } case .err(_): { 2 } };
}`)
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stdout != "" || stderr != "" {
			t.Fatalf("%v: %d %q %s", args, code, stdout, stderr)
		}
	}
}
