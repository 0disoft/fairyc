package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserRecordExpressions(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main() {\n")
	for _, expr := range []string{"R{}", "ns::R { .x = 1, .y = b\"ok\", }", "R{.x=R{.y=2}}", "R{.x=f(1)}.x", "R{.x=1,.x=2}", "R{/*gap*/.x=1+2*3}"} {
		input := "module main; fn void f() { return " + expr + "; }"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("record oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "{ let i32 result = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 0); 0 } case .err(_): { 1 } }; assert(result == 0); }\n", strconv.Quote(input))
	}
	p.WriteString(`{ let i32 result = switch(make(b"module main; fn void f() { return ns::R { .x = 1, .y = b\"ok\", }; }")) {
case .ok(parsed): {
 assert(syntax::expression_count(ref parsed) == 6);
 let syntax::Expression record = syntax::expression_at(ref parsed,6);
 assert(record.kind == 10 && record.left == 1 && record.right == 5 && record.start == 34 && record.end == 63);
 assert(syntax::record_field_count(ref parsed,6) == 2);
 assert(syntax::record_field(ref parsed,6,0) == 3 && syntax::record_field(ref parsed,6,1) == 5);
 let syntax::Expression x = syntax::expression_at(ref parsed,3);
 let syntax::Expression y = syntax::expression_at(ref parsed,5);
 assert(x.kind == 11 && x.left == 2 && x.right == 0 && x.start == 42 && x.end == 48);
 assert(y.kind == 11 && y.left == 4 && y.right == 3 && y.start == 50 && y.end == 60);
 assert(syntax::expression_text(ref parsed,3)[0] == 120 && syntax::expression_text(ref parsed,5)[0] == 121);
 0 } case .err(_): { 1 } }; assert(result == 0); }
`)
	for _, expr := range []string{"R{", "R{x=1}", "R{.=1}", "R{.x 1}", "R{.x=}", "R{.x=1 .y=2}", "R{.x=1,,}", "R<T>{.x=1}"} {
		input := "module main; fn void bad() { return " + expr + "; } fn void kept() {}"
		fmt.Fprintf(&p, "{ let i32 result = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) > 0 && syntax::expression_count(ref parsed) == 0 && syntax::entry_count(ref parsed) == 0); 0 } case .err(_): { 1 } }; assert(result == 0); }\n", strconv.Quote(input))
	}
	p.WriteString(`
var usize n = 2047;
while (n <= 2048) {
 let bytes::Buffer input = switch(bytes::copy(b"module main; fn void f() { return R{")) { case .ok(value): { value } case .err(_): { bytes::empty() } };
 var bytes::Buffer data = input;
 var usize i = 0;
 while (i < n) { discard bytes::append(ref mut data,b".x=0,"); i += 1; }
 discard bytes::append(ref mut data,b"};}");
 let i32 result = switch(make(bytes::view(ref data))) {
  case .ok(parsed): { assert(n == 2047 && syntax::expression_count(ref parsed) == 4096); 0 }
  case .err(status): { assert(n == 2048 && status == 4); 0 }
 };
 assert(result == 0); n += 1;
}
return 0; }`)
	runParserProbe(t, p.String())
}
