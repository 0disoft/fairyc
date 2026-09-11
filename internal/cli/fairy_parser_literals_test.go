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
		{`""`, "", 7},
		{`"hello"`, "hello", 7},
		{`"한글😀"`, "한글😀", 7},
		{`"a\n\"\\z"`, "a\n\"\\z", 7},
		{`b""`, "", 8},
		{`b"a\0\xFF"`, string([]byte{'a', 0, 255}), 8},
		{`b"한글"`, "한글", 8},
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
		for _, view := range []struct{ getter, want string }{{"expression_text", c.raw}, {"literal_cooked", hex.EncodeToString([]byte(c.cooked))}} {
			fmt.Fprintf(&p, "{ let ref u8[] value = syntax::%s(ref parsed,1); assert(value.len == %d);", view.getter, len(view.want))
			for i, b := range []byte(view.want) {
				fmt.Fprintf(&p, "assert(value[%d] == %d);", i, b)
			}
			p.WriteString("}")
		}
		p.WriteString("0 } case .err(_): { 1 } }; assert(result == 0); }\n")
	}
	for _, raw := range []string{`"unterminated`, `"bad\q"`, `b"bad\xG0"`} {
		input := "module main; fn void f() { return " + raw + "; }"
		fmt.Fprintf(&p, "{ let i32 result = switch(make(b%s)) { case .ok(_): { 0 } case .err(status): { status } }; assert(result == 5); }\n", strconv.Quote(input))
	}
	p.WriteString(`{ let i32 result = switch(make(b"module main; fn void f() { return c\"not yet\"; } fn void kept() {}")) {
case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 1 && syntax::declaration_count(ref parsed) == 2 && syntax::expression_count(ref parsed) == 0); 0 }
case .err(_): { 1 } }; assert(result == 0); }
return 0; }`)
	runParserProbe(t, p.String())
}
