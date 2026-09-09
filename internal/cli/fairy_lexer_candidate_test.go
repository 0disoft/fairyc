package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyLexerCandidate(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	inputs := []string{
		"", " \t\n\r\n", "\xef\xbb\xbfmodule main;\r\n",
		"as assert break case cleanup const continue discard else enum extern false fn for if import in let module mut null pub ptr ref region resource return struct switch true unsafe var while defer",
		"if ifx xif while2 _return return_ If defer deferx state key",
		"<<= >>= += -= *= /= %= &= ^= |= :: -> => == != <= >= << >> && || .. @ ( ) { } [ ] , ; : . ? = + - * / % ! ~ & | ^ < >",
		"<<<==== >>>= ->=> .... / /= //tail\n///docs\r\n",
		"/*outer /*nested*/ tail*/fn /*\xff\x00*/ main() {}",
		strings.Repeat("a", 4096),
	}
	root := t.TempDir()
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	for _, name := range []string{"fairy.toml", "src/scanner.fy"} {
		data, err := os.ReadFile(filepath.Join(example, name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, name, string(data))
	}
	var program, expected strings.Builder
	program.WriteString("module main;\nimport scanner as scanner;\nimport std::bytes as bytes;\n@effects(console)\nfn i32 main() {\n")
	writeInput := func(input string, index, status int) {
		name := fmt.Sprintf("input%d", index)
		fmt.Fprintf(&program, "{ let bytes::Buffer %s = switch (bytes::copy(b\"", name)
		for _, b := range []byte(input) {
			fmt.Fprintf(&program, "\\x%02X", b)
		}
		program.WriteString("\")) { case .ok(value) : { value } case .err(_) : { assert(false); bytes::empty() } };\n")
		fmt.Fprintf(&program, "assert(scanner::report(bytes::view(ref %s)) == %d); }\n", name, status)
	}
	for i, input := range inputs {
		writeInput(input, i, 0)
		result := lexer.Lex(source.NewFile("input.fy", []byte(input)), source.NewBudget(source.DefaultLimits(), len(input)))
		if len(result.Diagnostics) != 0 {
			t.Fatalf("invalid supported oracle input %d: %+v", i, result.Diagnostics)
		}
		for _, token := range result.Tokens {
			fmt.Fprintf(&expected, "%d %d %d %d %d ", token.Kind, token.Trivia, token.Span.StartByte, token.Span.EndByte, len(token.Cooked))
			for _, b := range []byte(token.Cooked) {
				fmt.Fprintf(&expected, "%d ", b)
			}
		}
	}
	for i, input := range []string{"123", "x 1", "\"text\"", "b\"bytes\"", "c\"string\"", "\r", " \r", "\v", "\f", "\x00", "\xff", "한", "/*open", "a\xef\xbb\xbf"} {
		writeInput(input, len(inputs)+i, 2)
	}
	writeInput(strings.Repeat("a", 4097), 100, 4)
	program.WriteString("return 0; }\n")
	writeTestSource(t, root, "src/main.fy", program.String())
	for _, release := range []bool{false, true} {
		args := []string{"run"}
		if release {
			args = append(args, "--release", "--locked")
		}
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stderr != "" || stdout != expected.String() {
			t.Fatalf("candidate release=%v code=%d stderr=%s\ngot %q\nwant %q", release, code, stderr, stdout, expected.String())
		}
	}
}
