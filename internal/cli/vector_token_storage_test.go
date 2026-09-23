package cli

import (
	"runtime"
	"testing"
)

// This is a storage prerequisite probe, not a Fairy lexer implementation. The
// fields mirror lexer.Token's kind, trivia, byte span and owned cooked bytes.
func TestWindowsVectorTokenStorage(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.vector-token-storage"
name = "tokens"
version = "1.0.0"
language = "fairyc-language/17"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", vectorTokenStorageSource)
	for _, release := range []bool{false, true} {
		args := []string{"run"}
		if release {
			args = append(args, "--release", "--locked")
		}
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stdout != "" || stderr != "" {
			t.Fatalf("token storage release=%v: %d %q %s", release, code, stdout, stderr)
		}
	}
}

const vectorTokenStorageSource = `module main;
import std::vec as v;
import std::bytes as bytes;

resource Token {
    u32 kind;
    u32 trivia;
    usize start;
    usize end;
    bytes::Buffer cooked;
    cleanup { }
}

fn Token token(usize index) {
    var bytes::Buffer cooked = bytes::empty();
    assert(switch (bytes::append_byte(ref mut cooked, 97)) {
        case .ok(n) : { n == 1 }
        case .err(_) : { false }
    });
    return Token { .kind = 7, .trivia = 0, .start = index * 3,
        .end = index * 3 + 3, .cooked = cooked };
}

fn bool valid(ref Token value, usize index) {
    let ref u8[] cooked = bytes::view(ref value.cooked);
    return value.kind == 7 && value.trivia == 0 && value.start == index * 3 &&
        value.end == index * 3 + 3 && cooked.len == 1 && cooked[0] == 97;
}

fn i32 main() {
    var v::Vec<Token> tokens = v::empty<Token>();
    var usize index = 0;
    while (index < 33) {
        assert(switch (v::push<Token>(ref mut tokens, token(index))) {
            case .ok(n) : { n == index + 1 }
            case .err(_) : { false }
        });
        index += 1;
    }
    index = 0;
    while (index < 33) {
        assert(valid(v::at<Token>(ref tokens, index), index));
        index += 1;
    }
    assert(switch (v::pop<Token>(ref mut tokens)) {
        case .some(value) : { valid(ref value, 32) }
        case .none : { false }
    });
    let usize capacity = v::capacity<Token>(ref tokens);
    assert(v::clear<Token>(ref mut tokens) == 32);
    assert(v::capacity<Token>(ref tokens) == capacity);
    assert(v::is_empty<Token>(ref tokens));
    assert(switch (v::push<Token>(ref mut tokens, token(0))) {
        case .ok(n) : { n == 1 }
        case .err(_) : { false }
    });
    assert(valid(v::at<Token>(ref tokens, 0), 0));
    return 0;
}
`
