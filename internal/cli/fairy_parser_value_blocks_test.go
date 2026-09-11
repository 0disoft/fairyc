package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserValueBlocks(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main() {\n")
	for _, expr := range []string{"{}", "{ 1 }", "{ let x = 1; x }", "{ var x = 1; x += 2; x; }", "{{1}}", "{ {} 1 }", "({ 1 }) + 2", "f({1}, {2})"} {
		input := "module main; fn void f() { return " + expr + "; }"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.AST == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("value block oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "{ let i32 r = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 0); 0 } case .err(_): { 1 } }; assert(r == 0); }\n", strconv.Quote(input))
	}
	input := "module main; fn void f() { return { let x = 1; x }; }"
	start, end := strings.Index(input, "{ let"), strings.Index(input, " };")+2
	fmt.Fprintf(&p, `{ let i32 r = switch(make(b%s)) { case .ok(parsed): {
 let syntax::Expression e = syntax::expression_at(ref parsed,3);
 assert(e.kind == 12 && e.start == %d && e.end == %d && e.left == 3 && e.right == 0);
 let syntax::Statement b = syntax::statement_at(ref parsed,e.left);
 assert(b.kind == 1 && b.expression == 2 && b.second == 1 && syntax::block_statement(ref parsed,e.left,0) == 1);
 assert(syntax::statement_at(ref parsed,1).kind == 2 && syntax::statement_at(ref parsed,1).entry == 2 && syntax::entry_at(ref parsed,1).expression == 1 && syntax::entry_at(ref parsed,0).expression == 3);
 assert(syntax::declaration_at(ref parsed,1).expression == 3);
 0 } case .err(_): { 1 } }; assert(r == 0); }
`, strconv.Quote(input), start, end)
	for _, body := range []string{"return { let x = 1; x x };", "return { x = 1 };", "return { let x = ; };", "1"} {
		input := "module main; fn void bad() { " + body + " } fn void kept() {}"
		fmt.Fprintf(&p, "{ let i32 r = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) > 0 && syntax::expression_count(ref parsed) == 0 && syntax::entry_count(ref parsed) == 0 && syntax::declaration_count(ref parsed) == 2); 0 } case .err(_): { 1 } }; assert(r == 0); }\n", strconv.Quote(input))
	}
	p.WriteString(`
var usize n = 31;
while(n <= 32) {
 var bytes::Buffer data = bytes::empty();
 discard bytes::append(ref mut data,b"module main; fn void f(){return ");
 var usize i = 0; while(i < n) { discard bytes::append(ref mut data,b"{"); i += 1; }
 discard bytes::append(ref mut data,b"1");
 i = 0; while(i < n) { discard bytes::append(ref mut data,b"}"); i += 1; }
 discard bytes::append(ref mut data,b";}");
 let i32 r = switch(make(bytes::view(ref data))) { case .ok(parsed): { assert(n == 31 && syntax::problem_count(ref parsed) == 0); 0 } case .err(status): { assert(n == 32 && status == 4); 0 } }; assert(r == 0);
 n += 1;
}
n = 1023;
while (n <= 1024) {
 var bytes::Buffer data = bytes::empty();
 discard bytes::append(ref mut data,b"module main; fn void f(){return {");
 var usize i = 0; while(i < n) { discard bytes::append(ref mut data,b"let x=0;"); i += 1; }
 discard bytes::append(ref mut data,b"1};}");
 let i32 r = switch(make(bytes::view(ref data))) { case .ok(parsed): { assert(n == 1023 && syntax::problem_count(ref parsed) == 0 && syntax::entry_count(ref parsed) == 1024 && syntax::entry_at(ref parsed,0).kind == 4); 0 } case .err(status): { assert(n == 1024 && status == 4); 0 } }; assert(r == 0);
 n += 1;
}
return 0; }`)
	runParserProbe(t, p.String())
}
