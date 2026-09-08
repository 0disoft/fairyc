package cli

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

// The Go lexer is the oracle and producer, not a second lexer implemented by
// this test. Native Fairy owns, grows and reads the resulting token sequence.
func TestWindowsVectorLexerRoundTrip(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	inputs := []string{
		"",
		"\xef\xbb\xbf/// docs\r\nmodule main; /* trivia */ fn i32 main() { let ref str s = \"a\\n한\"; return 42; }\n",
		"\xff \"unterminated\n /* unclosed",
	}
	var program, expected strings.Builder
	program.WriteString(vectorLexerRoundTripHelpers)
	for i, input := range inputs {
		file := source.NewFile("input.fy", []byte(input))
		result := lexer.Lex(file, source.NewBudget(source.DefaultLimits(), len(input)))
		rebuilt, ok := result.Reconstruct()
		if !ok || !bytes.Equal(rebuilt, []byte(input)) {
			t.Fatalf("oracle reconstruction %d", i)
		}
		if i == 2 && len(result.Diagnostics) == 0 {
			t.Fatal("malformed corpus must exercise diagnostic-producing tokens")
		}
		fmt.Fprintf(&program, "@effects(console)\nfn void case%d() {\nvar bytes::Buffer source = bytes::empty();\n", i)
		appendVectorRoundTripBytes(&program, "source", []byte(input))
		program.WriteString("var v::Vec<Token> tokens = v::empty<Token>();\n")
		for j, token := range result.Tokens {
			fmt.Fprintf(&program, "var bytes::Buffer cooked%d = bytes::empty();\n", j)
			appendVectorRoundTripBytes(&program, fmt.Sprintf("cooked%d", j), []byte(token.Cooked))
			fmt.Fprintf(&program, "push(ref mut tokens, make(%d, %d, %d, %d, cooked%d));\n", token.Kind, token.Trivia, token.Span.StartByte, token.Span.EndByte, j)
			fmt.Fprintf(&expected, "%d %d %d %d %d ", token.Kind, token.Trivia, token.Span.StartByte, token.Span.EndByte, len(token.Cooked))
			for _, b := range []byte(token.Cooked) {
				fmt.Fprintf(&expected, "%d ", b)
			}
		}
		program.WriteString("report(ref tokens, bytes::view(ref source));\n}\n")
		expected.WriteString(input)
	}
	program.WriteString("@effects(console)\nfn i32 main() { case0(); case1(); case2(); return 0; }\n")
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.vector-lexer-roundtrip"
name = "roundtrip"
version = "1.0.0"
language = "fairyc-language/14"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", program.String())
	for _, release := range []bool{false, true} {
		args := []string{"run"}
		if release {
			args = append(args, "--release", "--locked")
		}
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stderr != "" || stdout != expected.String() {
			t.Fatalf("roundtrip release=%v: code=%d stderr=%s\ngot %q\nwant %q", release, code, stderr, stdout, expected.String())
		}
	}
}

func appendVectorRoundTripBytes(out *strings.Builder, name string, value []byte) {
	for _, b := range value {
		fmt.Fprintf(out, "put(ref mut %s, %d);\n", name, b)
	}
}

const vectorLexerRoundTripHelpers = `module main;
import std::vec as v;
import std::bytes as bytes;
import std::io as io;
resource Token {
    u32 kind; u32 trivia; usize start; usize end; bytes::Buffer cooked;
    cleanup { }
}
fn Token make(u32 kind, u32 trivia, usize start, usize end, bytes::Buffer cooked) {
    return Token { .kind = kind, .trivia = trivia, .start = start, .end = end, .cooked = cooked };
}
fn void put(ref mut bytes::Buffer output, u8 value) {
    assert(switch (bytes::append_byte(output, value)) { case .ok(_) : { true } case .err(_) : { false } });
}
fn void push(ref mut v::Vec<Token> tokens, Token value) {
    assert(switch (v::push<Token>(tokens, value)) { case .ok(_) : { true } case .err(_) : { false } });
}
@effects(console)
fn void number(usize value) {
    assert(switch (io::write_usize(value)) { case .ok(_) : { true } case .err(_) : { false } });
    assert(switch (io::write(" ")) { case .ok(_) : { true } case .err(_) : { false } });
}
@effects(console)
fn void report(ref v::Vec<Token> tokens, ref u8[] source) {
    var bytes::Buffer rebuilt = bytes::empty();
    var usize position = 0;
    var usize index = 0;
    while (index < v::len<Token>(tokens)) {
        let ref Token token = v::at<Token>(tokens, index);
        number(token.kind as usize); number(token.trivia as usize);
        number(token.start); number(token.end);
        let ref u8[] cooked = bytes::view(ref token.cooked);
        number(cooked.len);
        var usize byte_index = 0;
        while (byte_index < cooked.len) { number(cooked[byte_index] as usize); byte_index += 1; }
        assert(token.start == position && token.end >= token.start && token.end <= source.len);
        while (position < token.end) { put(ref mut rebuilt, source[position]); position += 1; }
        index += 1;
    }
    assert(position == source.len);
    let ref u8[] actual = bytes::view(ref rebuilt);
    assert(switch (io::write_bytes(actual)) { case .ok(n) : { n == source.len } case .err(_) : { false } });
}
`
