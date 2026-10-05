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
	"github.com/0disoft/fairyc/internal/syntax"
)

type declaration struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	NameStart int    `json:"nameStart"`
	NameEnd   int    `json:"nameEnd"`
}
type parserGolden struct {
	Message string `json:"message"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
}
type declarationFixture struct {
	Name        string                  `json:"name"`
	InputHex    string                  `json:"inputHex"`
	LogicalPath string                  `json:"logicalPath"`
	Outcome     string                  `json:"outcome"`
	Records     []declaration           `json:"records"`
	Lexical     []diagnostic.Diagnostic `json:"lexical"`
	Parser      []diagnostic.Diagnostic `json:"parser"`
	Goldens     []parserGolden          `json:"goldens"`
}

func writeDeclarations(destination string) error {
	var fixtures []declarationFixture
	add := func(name, logicalPath, input, outcome string, goldens ...parserGolden) error {
		data := []byte(input)
		file := source.NewFile(logicalPath, data)
		lexed := lexer.Lex(file, source.NewBudget(source.DefaultLimits(), len(data)))
		tree := syntax.Parse(file, source.DefaultLimits())
		f := declarationFixture{Name: name, InputHex: hex.EncodeToString(data), LogicalPath: logicalPath,
			Outcome: outcome, Records: []declaration{}, Lexical: lexed.Diagnostics, Parser: tree.Diagnostics, Goldens: goldens}
		if outcome == "pass" || outcome == "limit" {
			if !tree.ErrorFree() {
				return fmt.Errorf("positive reference %s: %+v", name, tree.Diagnostics)
			}
			root, ok := tree.Node(tree.Root)
			if !ok {
				return fmt.Errorf("missing reference root: %s", name)
			}
			for _, id := range root.Children {
				node, ok := tree.Node(id)
				if !ok {
					return fmt.Errorf("missing reference declaration: %s", name)
				}
				kind := string(node.Kind)
				switch node.Kind {
				case syntax.KindModule, syntax.KindImport, syntax.KindFunction, syntax.KindStruct, syntax.KindResource,
					syntax.KindConst, syntax.KindStaticAssert, syntax.KindForeign, syntax.KindState, syntax.KindEnum:
				default:
					return fmt.Errorf("unmapped declaration %s in %s", kind, name)
				}
				span := node.NameSpan
				if node.Kind == syntax.KindModule {
					path, _ := tree.Node(node.Children[0])
					span = path.Span
				}
				if node.Kind == syntax.KindImport {
					alias := false
					for _, t := range tree.Tokens[node.TokenStart:node.TokenEnd] {
						if t.Text(data) == "as" {
							alias = true
							continue
						}
						if alias && t.Kind == lexer.Identifier {
							span = t.Span
							break
						}
					}
				}
				f.Records = append(f.Records, declaration{kind, string(data[span.StartByte:span.EndByte]), node.Span.StartByte,
					node.Span.EndByte, span.StartByte, span.EndByte})
			}
		} else if outcome == "lexical" {
			if len(lexed.Diagnostics) == 0 {
				return fmt.Errorf("missing lexical rejection: %s", name)
			}
		} else if outcome == "syntax" {
			if len(lexed.Diagnostics) != 0 || tree.ErrorFree() || len(goldens) == 0 {
				return fmt.Errorf("invalid syntax witness: %s", name)
			}
		} else {
			return fmt.Errorf("unknown fixture outcome: %s", outcome)
		}
		fixtures = append(fixtures, f)
		return nil
	}
	seeds := []struct{ name, input string }{
		{"module-only", "module main;"},
		{"declarations", "module main; import std::bytes as bytes; pub const i64 BIG=4000000000; static_assert(true); struct Pair { i64 left; i64 right; } resource Handle { i32 code; cleanup {} } enum Kind { Empty; Some { i32 code; }; } unsafe extern C fn i32 foreign(i32 value); fn i64 value() { return BIG; } state { cell i32 count; map i32 i64 totals; }"},
		{"attributes", "module main; /// docs\n@repr(C) pub struct S { i32 code; } @effects(console) pub fn void f() { return; }"},
		{"generics", "module main; struct Box<T> { T item; } fn T same<T>(T item) { return item; }"},
		{"methods", "module main; struct S {} fn i32 S::value(ref S self) { return 1; } unsafe fn i32 raw() { return 2; }"},
		{"semantic-only-errors", "module main; fn Missing unknown(Missing arg) { return absent; } fn i32 duplicate() { return 1; } fn i32 duplicate() { return 2; }"},
		{"CRLF-no-final-newline", "module main;\r\nfn i32 main() {\r\n return 0;\r\n}"},
		{"BOM", "\xef\xbb\xbfmodule main;"},
		{"body-structure", "module main; fn i32 work(i32 n) { let i32 total=0; var i32 i=0; for(i=0;i<n;i+=1) { total+=i; } if(total>0) { return total; } return 0; }"},
		{"declaration-limit", "module main;" + strings.Repeat("const i32 n=0;", 511)},
	}
	for _, s := range seeds {
		if err := add(s.name, "main.fy", s.input, "pass"); err != nil {
			return err
		}
	}
	if err := add("declaration-limit-exceeded", "main.fy", "module main;"+strings.Repeat("const i32 n=0;", 512), "limit"); err != nil {
		return err
	}
	if err := add("source-limit", "main.fy", "module main;"+strings.Repeat(" ", 32768-len("module main;")), "pass"); err != nil {
		return err
	}
	if err := add("qualified-module", "pkg/main.fy", "module pkg::main; import other::file as alias; fn void run() {}", "pass"); err != nil {
		return err
	}
	bad := []struct {
		name, input, message string
		start, end           int
	}{
		{"empty", "", "expected module", 0, 0},
		{"missing-module", "fn i32 main() {}", "expected module", 0, 2},
		{"module-semicolon", "module main", "expected ;", 11, 11},
		{"late-const-semicolon", "module main;\nconst i32 x=1", "expected ;", 26, 26},
		{"import-alias", "module main; import std::io;", "expected as", 27, 28},
		{"function-close", "module main; fn i32 f() { return 0;", "expected }", 35, 35},
		{"struct-name", "module main; struct { i32 code; }", "expected identifier", 20, 21},
		{"enum-member", "module main; enum Kind { 1; }", "expected identifier", 25, 26},
		{"late-syntax", "module main; fn i32 good() { return 1; }\nfn i32 bad() { return 2 }", "expected ;", 65, 66},
	}
	for _, s := range bad {
		if err := add(s.name, "main.fy", s.input, "syntax", parserGolden{s.message, s.start, s.end}); err != nil {
			return err
		}
	}
	for _, s := range []struct{ name, input string }{{"invalid-byte", "module main;\n\xff"}, {"lone-CR", "module main;\rfn void f() {}"}, {"bad-number", "module main; const i32 n=0xG;"}, {"unclosed-comment", "module main; /*open"}} {
		if err := add(s.name, "main.fy", s.input, "lexical"); err != nil {
			return err
		}
	}
	for _, path := range []string{"examples/lexer_candidate/src/main.fy", "examples/lexer_candidate/src/scanner.fy", "examples/lexer_candidate/samples/token-demo.fy"} {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		logical := filepath.Base(path)
		if strings.HasSuffix(path, "token-demo.fy") {
			logical = "main.fy"
		}
		if len(data) > 32768 {
			return fmt.Errorf("source guard: %s", path)
		}
		if err := add(path, logical, string(data), "pass"); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(struct {
		Schema   string               `json:"schema"`
		Fixtures []declarationFixture `json:"fixtures"`
	}{"fairyc.declarations-file-oracle/v1", fixtures}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	return os.WriteFile(destination, append(data, '\n'), 0644)
}
