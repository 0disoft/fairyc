// fairyc-lexer-file-oracle writes bounded verification fixtures from the Go
// reference. It is a test utility, never a lexer-candidate runtime dependency.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

type token struct {
	Kind      uint8  `json:"kind"`
	Trivia    uint8  `json:"trivia"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	RawHex    string `json:"rawHex"`
	CookedHex string `json:"cookedHex"`
}
type fixture struct {
	Name        string                  `json:"name"`
	InputHex    string                  `json:"inputHex"`
	Tokens      []token                 `json:"tokens"`
	Diagnostics []diagnostic.Diagnostic `json:"diagnostics"`
}

func run() error {
	if len(os.Args) == 3 && os.Args[1] == "--hir" {
		return writeHIR(os.Args[2])
	}
	if len(os.Args) == 3 && os.Args[1] == "--modules" {
		return writeModules(os.Args[2])
	}
	if len(os.Args) == 3 && os.Args[1] == "--types" {
		return writeTypes(os.Args[2])
	}
	if len(os.Args) == 3 && os.Args[1] == "--names" {
		return writeNames(os.Args[2])
	}
	if len(os.Args) == 3 && os.Args[1] == "--declarations" {
		return writeDeclarations(os.Args[2])
	}
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: fairyc-lexer-file-oracle [--declarations|--names|--types|--modules|--hir] <output.json>")
	}
	var fixtures []fixture
	add := func(name string, input []byte) error {
		file := source.NewFile("input.fy", input)
		result := lexer.Lex(file, source.NewBudget(source.DefaultLimits(), len(input)))
		reconstructed, ok := result.Reconstruct()
		if !ok || string(reconstructed) != string(input) {
			return fmt.Errorf("reference coverage: %s", name)
		}
		f := fixture{Name: name, InputHex: hex.EncodeToString(input), Tokens: []token{}, Diagnostics: result.Diagnostics}
		for _, t := range result.Tokens {
			f.Tokens = append(f.Tokens, token{uint8(t.Kind), uint8(t.Trivia), t.Span.StartByte, t.Span.EndByte,
				hex.EncodeToString(input[t.Span.StartByte:t.Span.EndByte]), hex.EncodeToString([]byte(t.Cooked))})
		}
		fixtures = append(fixtures, f)
		return nil
	}
	seeds := []struct{ name, input string }{
		{"empty", ""}, {"whitespace", " \t\n\r\n"}, {"lone-CR", "module a;\rfn i32 f() { return 0; }"},
		{"program", "module main;\n@effects(console)\nfn i64 f() { let bytes::Buffer b = bytes::empty(); return 4000000000; }"},
		{"keywords", "as assert break case cleanup const continue discard else enum extern false fn for if import in let module mut null pub ptr ref region resource return struct switch true unsafe var while defer"},
		{"keyword-prefixes", "modulex let2 _fn deferx i64 usize Buffer"},
		{"symbols", "<<= >>= += -= *= /= %= &= ^= |= :: -> => == != <= >= << >> && || .. @ ( ) { } [ ] , ; : . ? = + - * / % ! ~ & | ^ < >"},
		{"integers", "0 0007 4000000000 9223372036854775808 0xFF 0b1010 1_000"},
		{"floats", "1.5 1e3 1.2e-3 2..4"}, {"bad-number", "module a;\nlet i32 x=0xG;"},
		{"comments", "/// docs\r\nmodule a; // line\n/* outer /* nested */ end */"},
		{"unclosed-comment", "module a;\n/* open"},
		{"strings", "\"hello\\n한글\" b\"\\x00\\xFF\" c\"ok\" true false null"},
		{"bad-escape", "\"\\q\""}, {"unclosed-string", "module a;\n\"open"},
		{"UTF8-BOM", "\xef\xbb\xbfmodule a;"}, {"invalid-UTF8", "module a;\n\xff"},
		{"NUL", "module a;\n\x00"}, {"non-ASCII-code", "module 한글;"},
		{"multiple-errors", "0xG \"\\q\" #"}, {"diagnostic-limit", strings.Repeat("#", 257)},
		{"source-limit", strings.Repeat("a", 32768)}, {"output-limit", strings.Repeat("a ", 16384)},
	}
	for _, seed := range seeds {
		if err := add(seed.name, []byte(seed.input)); err != nil {
			return err
		}
	}
	for _, path := range []string{"examples/token_page_cli/src/main.fy", "examples/token_page_cli/src/tokens.fy",
		"examples/lexer_candidate/src/main.fy", "examples/lexer_candidate/src/file_cli.fy", "examples/lexer_candidate/src/token_output.fy",
		"examples/lexer_candidate/samples/token-demo.fy"} {
		input, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(input) > 32768 {
			return fmt.Errorf("source guard: %s", path)
		}
		if err := add(path, input); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(struct {
		Schema   string    `json:"schema"`
		Fixtures []fixture `json:"fixtures"`
	}{"fairyc.lexer-file-oracle/v1", fixtures}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(os.Args[1]), 0755); err != nil {
		return err
	}
	return os.WriteFile(os.Args[1], append(data, '\n'), 0644)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
