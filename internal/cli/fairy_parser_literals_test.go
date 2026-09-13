package cli

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserLiterals(t *testing.T) {
	cases := []struct {
		raw, cooked string
		kind        int
	}{
		{`null`, "", 4},
		{`true`, "", 4},
		{`false`, "", 4},
		{`""`, "", 7},
		{`"hello"`, "hello", 7},
		{`"한글😀"`, "한글😀", 7},
		{`"a\n\"\\z"`, "a\n\"\\z", 7},
		{`b""`, "", 8},
		{`b"a\0\xFF"`, string([]byte{'a', 0, 255}), 8},
		{`b"한글"`, "한글", 8},
		{`c""`, "\x00", 9},
		{`c"fairy"`, "fairy\x00", 9},
		{`c"한글😀"`, "한글😀\x00", 9},
		{`c"a\n"`, "a\n\x00", 9},
	}
	var p strings.Builder
	p.WriteString("fn i32 main() {\n")
	for _, c := range cases {
		input := "module main; fn void f() { return " + c.raw + "; }"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("literal oracle %q: %v", c.raw, err)
		}
		start := strings.Index(input, c.raw)
		fmt.Fprintf(&p, "{ let i32 result = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 0 && syntax::expression_count(ref parsed) == 1); let syntax::Expression e = syntax::expression_at(ref parsed,1); assert(e.kind == 1 && e.start == %d && e.end == %d && e.left == 0 && e.right == 0); assert(syntax::literal_kind(ref parsed,1) == %d);", strconv.Quote(input), start, start+len(c.raw), c.kind)
		cooked := hex.EncodeToString([]byte(c.cooked))
		if c.kind == 4 {
			cooked = c.raw // Keyword literals retain spelling, unlike hex-encoded string payloads.
		}
		for _, view := range []struct{ getter, want string }{{"expression_text", c.raw}, {"literal_cooked", cooked}} {
			fmt.Fprintf(&p, "{ let ref u8[] value = syntax::%s(ref parsed,1); assert(value.len == %d);", view.getter, len(view.want))
			for i, b := range []byte(view.want) {
				fmt.Fprintf(&p, "assert(value[%d] == %d);", i, b)
			}
			p.WriteString("}")
		}
		p.WriteString("assert(integrity::valid(ref parsed));0 } case .err(_): { 1 } }; assert(result == 0); }\n")
	}
	for _, raw := range []string{`"unterminated`, `"bad\q"`, `b"bad\xG0"`, `c"unterminated`, `c"bad\q"`, `c"\0"`, `c"\x00"`, `c"\u{0}"`} {
		input := "module main; fn void f() { return " + raw + "; }"
		fmt.Fprintf(&p, "{ let i32 result = switch(make(b%s)) { case .ok(_): { 0 } case .err(status): { status } }; assert(result == 5); }\n", strconv.Quote(input))
	}
	for _, body := range []string{
		"var ptr mut void slot = null; assert(slot != null); return slot == null;",
		"let R value = R { .data = null }; consume(null); return (null);",
	} {
		input := "module main; fn void f() { " + body + " }"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("null context oracle %q: %v", body, err)
		}
		fmt.Fprintf(&p, "{ let i32 result = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 0); var usize found = 0; var usize id = 1; while (id <= syntax::expression_count(ref parsed)) { let syntax::Expression e = syntax::expression_at(ref parsed,id); if (e.kind == 1) { assert(syntax::literal_kind(ref parsed,id) == 4 && e.left == 0 && e.right == 0 && e.end == e.start + 4); let ref u8[] raw = syntax::expression_text(ref parsed,id); assert(raw.len == 4 && raw[0] == 110 && raw[1] == 117 && raw[2] == 108 && raw[3] == 108); found += 1; } id += 1; } assert(found == 3); assert(integrity::valid(ref parsed)); 0 } case .err(_): { 1 } }; assert(result == 0); }\n", strconv.Quote(input))
	}
	for _, name := range []string{"nullish", "null_", "Null"} {
		input := "module main; fn void f() { return " + name + "; }"
		fmt.Fprintf(&p, "{ let i32 result = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 0 && syntax::expression_count(ref parsed) == 1); assert(syntax::expression_at(ref parsed,1).kind == 2); assert(integrity::valid(ref parsed)); 0 } case .err(_): { 1 } }; assert(result == 0); }\n", strconv.Quote(input))
	}
	p.WriteString("return 0; }")
	runParserProbe(t, p.String())
}
