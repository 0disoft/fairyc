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
	p.WriteString(`fn i32 accepted(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 0 && integrity::valid(ref parsed)); return 0; }
fn i32 rejected(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) > 0 && syntax::expression_count(ref parsed) == 0 && syntax::entry_count(ref parsed) == 0 && syntax::type_count(ref parsed) + 1 == syntax::declaration_count(ref parsed) && integrity::valid(ref parsed)); return 0; }
fn i32 main() {
`)
	for _, expr := range []string{"R{}", "R<T>>{}", "ns::R { .x = 1, .y = b\"ok\", }", "R{.x=R{.y=2}}", "R{.x=f(1)}.x", "R{.x=1,.x=2}", "R{/*gap*/.x=1+2*3}", "R<T>{}", "ns::R<ptr mut T, U[],>{.x=null,.y=f<T>()}.x", "R<Box<Box<T>>>{.x=R<T>{.x=1}}", "R /*gap*/ <T> /*gap*/ {.x=1}.x[0]", "R<T>{.x=1,.x=2}", ".Empty{}", ".Some{.value=1}", ".Outer{.inner=.Inner{.value=2},}", ".Some{.value=R<T>{.x=1}}.value", ". /*gap*/ Some { .value=1,.value=2 }"} {
		input := "module main; fn void f() { return " + expr + "; }"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("record oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "assert(switch(make(b%s)) { case .ok(parsed): { accepted(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input))
	}
	variantInput := "module main; fn void f() { return . /*gap*/ Pair { .left = 20, .right = 22, }; }"
	variantOracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(variantInput)), source.DefaultLimits())
	if err != nil || variantOracle.AST == nil || len(variantOracle.Tree.Diagnostics) != 0 {
		t.Fatalf("contextual record oracle: %v", err)
	}
	record := variantOracle.AST.Functions[0].Body.Statements[0].Expression
	if record == nil || len(record.Children) != 3 {
		t.Fatal("contextual record oracle shape")
	}
	callee := record.Children[0]
	fmt.Fprintf(&p, "{ let result = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 0 && syntax::expression_count(ref parsed) == 6 && integrity::valid(ref parsed));\n", strconv.Quote(variantInput))
	fmt.Fprintf(&p, "{ let r = syntax::expression_at(ref parsed,6); assert(r.kind == 10 && r.left == 1 && r.right == 5 && r.start == %d && r.end == %d); }\n", record.Span.StartByte, record.Span.EndByte)
	fmt.Fprintf(&p, "{ let v = syntax::expression_at(ref parsed,1); assert(v.kind == 28 && v.left == 0 && v.right == 0 && v.start == %d && v.end == %d); }\n", callee.Span.StartByte, callee.Span.EndByte)
	name := strings.TrimPrefix(callee.Text, ".")
	fmt.Fprintf(&p, "assert(syntax::expression_text(ref parsed,1).len == %d);\n", len(name))
	for i, b := range []byte(name) {
		fmt.Fprintf(&p, "assert(syntax::expression_text(ref parsed,1)[%d] == %d);\n", i, b)
	}
	p.WriteString("assert(syntax::record_field_count(ref parsed,6) == 2);\n")
	for i, f := range record.Children[1:] {
		id, value, previous := 3+2*i, 2+2*i, 0
		if i > 0 {
			previous = id - 2
		}
		fmt.Fprintf(&p, "{ let f = syntax::expression_at(ref parsed,%d); assert(syntax::record_field(ref parsed,6,%d) == %d && f.kind == 11 && f.left == %d && f.right == %d && f.start == %d && f.end == %d); }\n", id, i, id, value, previous, f.Span.StartByte, f.Span.EndByte)
		fmt.Fprintf(&p, "assert(syntax::expression_text(ref parsed,%d).len == %d);\n", id, len(f.Text))
		for j, b := range []byte(f.Text) {
			fmt.Fprintf(&p, "assert(syntax::expression_text(ref parsed,%d)[%d] == %d);\n", id, j, b)
		}
	}
	p.WriteString("let original = syntax::expression_at(ref parsed,6); var broken = original;\n")
	for _, mutation := range []string{"left=0", "left=2", "left=6", "left=999999", "right=2", "right=6", "right=999999"} {
		fmt.Fprintf(&p, "broken=original; broken.%s; assert(!integrity::valid_expression(ref parsed,broken,6));\n", mutation)
	}
	p.WriteString("let field = syntax::expression_at(ref parsed,5);\n")
	for _, mutation := range []string{"left=0", "left=5", "right=5", "right=2"} {
		fmt.Fprintf(&p, "broken=field; broken.%s; assert(!integrity::valid_expression(ref parsed,broken,5));\n", mutation)
	}
	p.WriteString("broken=syntax::expression_at(ref parsed,1); broken.left=2; assert(!integrity::valid_expression(ref parsed,broken,1)); assert(integrity::valid(ref parsed));\n")
	p.WriteString("0 } case .err(_): {1} }; assert(result == 0); }\n")
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
	for _, expr := range []string{"R{", "R{x=1}", "R{.=1}", "R{.x 1}", "R{.x=}", "R{.x=1 .y=2}", "R{.x=1,,}", "R<T>{.x=}", "R<>{}", "R<T,,U>{}", "R<ptr>{}", "R<T>>{.x=1}", "R<T>{.x=1,,}", "R<T>{.x=1", "R<T>{}{}", "(R)<T>{}", ".Some{.value=}", ".Some{value=1}", ".Some{.=1}", ".Some{.value 1}", ".Some{.value=1", ".Some{.x=1 .y=2}", ".Some{.value=1,,}", ".Some{}{}"} {
		input := "module main; fn void bad() { return " + expr + "; } fn void kept() {}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) == 0 {
			t.Fatalf("invalid record oracle accepted %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "assert(switch(make(b%s)) { case .ok(parsed): { rejected(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input))
	}

	p.WriteString(`{ let i32 result = switch(make(b"module main; fn void f() { return ns::R<T,U>{.x=null,.y=2}; }")) {
case .ok(parsed): {
 assert(syntax::expression_count(ref parsed) == 7 && syntax::type_count(ref parsed) == 5);
 let syntax::Expression record = syntax::expression_at(ref parsed,7);
 let syntax::Expression spec = syntax::expression_at(ref parsed,2);
 let syntax::Expression path = syntax::expression_at(ref parsed,1);
 assert(record.kind == 10 && record.left == 2 && record.right == 6 && record.start == 34 && record.end == 58);
 assert(spec.kind == 20 && spec.left == 1 && spec.right == 5 && spec.start == 34 && spec.end == 44);
 assert(path.kind == 2 && path.start == 34 && path.end == 39);
 assert(syntax::specialization_argument_count(ref parsed,2) == 2);
 assert(syntax::specialization_argument(ref parsed,2,0) == 2 && syntax::specialization_argument(ref parsed,2,1) == 4);
 assert(syntax::type_at(ref parsed,2).start == 40 && syntax::type_at(ref parsed,4).start == 42);
 assert(syntax::record_field_count(ref parsed,7) == 2 && syntax::record_field(ref parsed,7,0) == 4 && syntax::record_field(ref parsed,7,1) == 6);
 let syntax::Expression x = syntax::expression_at(ref parsed,4);
 let syntax::Expression y = syntax::expression_at(ref parsed,6);
 assert(x.kind == 11 && x.left == 3 && x.right == 0 && x.start == 45 && x.end == 52);
 assert(y.kind == 11 && y.left == 5 && y.right == 4 && y.start == 53 && y.end == 57);
 assert(integrity::valid(ref parsed));
 var syntax::Expression broken = record; broken.left = 3;
 assert(!integrity::valid_expression(ref parsed,broken,7));
 broken = spec; broken.right = 2;
 assert(!integrity::valid_expression(ref parsed,broken,2));
 0 } case .err(_): { 1 } }; assert(result == 0); }
`)
	p.WriteString(`
var usize form = 0;
while (form < 2) {
var usize n = 8191;
while (n <= 8192) {
 let bytes::Buffer input = switch(bytes::copy(b"module main; fn void f() { return ")) { case .ok(value): { value } case .err(_): { bytes::empty() } };
 var bytes::Buffer data = input;
 if (form == 0) { discard bytes::append(ref mut data,b"R{"); } else { discard bytes::append(ref mut data,b".R{"); }
 var usize i = 0;
 while (i < n) { discard bytes::append(ref mut data,b".x=0,"); i += 1; }
 discard bytes::append(ref mut data,b"};}");
 let i32 result = switch(make(bytes::view(ref data))) {
  case .ok(parsed): { assert(n == 8191 && syntax::expression_count(ref parsed) == 16384 && syntax::problem_count(ref parsed) == 0 && integrity::valid(ref parsed)); assert(syntax::expression_at(ref parsed,1).kind == (if (form == 0) { 2 } else { 28 })); 0 }
  case .err(status): { assert(n == 8192 && status == 4); 0 }
 };
 assert(result == 0); n += 1;
}
form += 1;
}
return 0; }`)
	runParserProbe(t, p.String())
}
