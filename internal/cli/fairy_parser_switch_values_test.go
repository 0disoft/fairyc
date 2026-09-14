package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserSwitchValues(t *testing.T) {
	t.Run("enum declaration construction decomposition", testParserEnumSyntaxComposition)
	var p strings.Builder
	p.WriteString(`fn i32 rejected(syntax::Parsed parsed, usize declarations) {
 assert(syntax::problem_count(ref parsed)>0 && syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==declarations && syntax::type_count(ref parsed)+1==declarations && integrity::valid(ref parsed)); return 0;
}
fn i32 main() {
`)
	for _, expr := range []string{
		"switch(x){}",
		"switch(x){case .ok(value):{value} case .err(_):{0}}",
		"switch(x){case .none:{} case .none:{1}}",
		"switch(f()){case .ok(x):{let y=x; if(y){1}else{2}}}",
		"1 + switch(x){case .ok(y):{y}} * 2",
		"switch(x){case .ok(y):{switch(y){case .done:{1}}}}",
		"switch(x){case .Empty {}:{0} case .Pair{.left=first,.right=second,}:{first+second}}",
		"switch(x){case .Pair{.left=_,.left=again}:{again} case .Some(value):{value}}",
	} {
		input := "module main; fn void f(){return " + expr + ";}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("switch oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "{ let i32 r = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 0 && integrity::valid(ref parsed)); 0 } case .err(_): { 1 } }; assert(r == 0); }\n", strconv.Quote(input))
	}
	namedInput := "module main; fn void f(){return switch(x){case . /*gap*/ Pair { .left=first, .right=second, }:{first}};}"
	oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(namedInput)), source.DefaultLimits())
	if err != nil || oracle.AST == nil || len(oracle.Tree.Diagnostics) != 0 {
		t.Fatalf("named pattern oracle: %v", err)
	}
	value := oracle.AST.Functions[0].Body.Statements[0].Expression
	if value == nil || len(value.Children) != 2 || len(value.Children[1].Children) != 2 {
		t.Fatal("switch oracle shape")
	}
	pat := value.Children[1].Children[0]
	if len(pat.Children) != 3 {
		t.Fatal("pattern field oracle shape")
	}
	fmt.Fprintf(&p, "{ let r = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed)==0 && syntax::expression_count(ref parsed)==10 && integrity::valid(ref parsed));\n", strconv.Quote(namedInput))
	fmt.Fprintf(&p, "{ let n=syntax::expression_at(ref parsed,6); assert(n.kind==32 && n.left==0 && n.right==5 && n.start==%d && n.end==%d); } assert(syntax::pattern_field_count(ref parsed,6)==2);\n", pat.Span.StartByte, pat.Span.EndByte)
	name := strings.TrimPrefix(pat.Text, ".")
	fmt.Fprintf(&p, "assert(syntax::expression_text(ref parsed,6).len==%d);\n", len(name))
	for j, b := range []byte(name) {
		fmt.Fprintf(&p, "assert(syntax::expression_text(ref parsed,6)[%d]==%d);\n", j, b)
	}
	for i, f := range pat.Children[1:] {
		if len(f.Children) != 0 || f.Cooked == "" {
			t.Fatal("pattern binding oracle shape")
		}
		id, binding, previous := 3+2*i, 2+2*i, 0
		if i > 0 {
			previous = id - 2
		}
		fmt.Fprintf(&p, "{ let n=syntax::expression_at(ref parsed,%d); assert(n.kind==33 && n.left==%d && n.right==%d && n.start==%d && n.end==%d); } assert(syntax::pattern_field(ref parsed,6,%d)==%d);\n", id, binding, previous, f.Span.StartByte, f.Span.EndByte, i, id)
		bindingEnd := f.Span.EndByte
		bindingStart := bindingEnd - len(f.Cooked)
		if namedInput[bindingStart:bindingEnd] != f.Cooked {
			t.Fatal("binding source span")
		}
		fmt.Fprintf(&p, "{ let n=syntax::expression_at(ref parsed,%d); assert(n.kind==17 && n.left==0 && n.right==0 && n.start==%d && n.end==%d); }\n", binding, bindingStart, bindingEnd)
		for index, text := range []string{f.Text, f.Cooked} {
			node := []int{id, binding}[index]
			fmt.Fprintf(&p, "assert(syntax::expression_text(ref parsed,%d).len==%d);\n", node, len(text))
			for j, c := range []byte(text) {
				fmt.Fprintf(&p, "assert(syntax::expression_text(ref parsed,%d)[%d]==%d);\n", node, j, c)
			}
		}
	}
	p.WriteString("let original=syntax::expression_at(ref parsed,6); var broken=original;\n")
	for _, mutation := range []string{"left=2", "right=2", "right=6", "right=999999"} {
		fmt.Fprintf(&p, "broken=original; broken.%s; assert(!integrity::valid_expression(ref parsed,broken,6));\n", mutation)
	}
	p.WriteString("let field=syntax::expression_at(ref parsed,5);\n")
	for _, mutation := range []string{"left=0", "left=1", "left=5", "left=999999", "right=1", "right=5", "right=999999"} {
		fmt.Fprintf(&p, "broken=field; broken.%s; assert(!integrity::valid_expression(ref parsed,broken,5));\n", mutation)
	}
	p.WriteString("assert(integrity::valid(ref parsed)); 0 } case .err(_): {1} }; assert(r==0); }\n")
	for i, pattern := range []string{".Empty", ".Empty{}"} {
		input := "module main; fn void f(){return switch(x){case " + pattern + ":{0}};}"
		kind := 16
		if i == 1 {
			kind = 32
		}
		fmt.Fprintf(&p, "{ let r=switch(make(b%s)) { case .ok(parsed): { let n=syntax::expression_at(ref parsed,2); assert(n.kind==%d && n.left==0 && n.right==0 && syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed)); 0 } case .err(_): {1} }; assert(r==0); }\n", strconv.Quote(input), kind)
	}
	input := "module main; fn void f(){return switch(x){case .ok(v):{v} case .err(_):{0}};}"
	fmt.Fprintf(&p, `{ let i32 r = switch(make(b%s)) { case .ok(parsed): {
 let syntax::Expression e = syntax::expression_at(ref parsed,12);
 assert(e.kind == 15 && e.start == %d && e.end == %d && e.left == 1 && e.right == 11);
 assert(syntax::switch_case_count(ref parsed,12) == 2 && syntax::switch_case(ref parsed,12,0) == 5 && syntax::switch_case(ref parsed,12,1) == 10);
 let syntax::Expression arm = syntax::expression_at(ref parsed,5);
 assert(arm.kind == 18 && arm.left == 3 && arm.right == 1 && arm.start == %d && arm.end == %d);
 let syntax::Expression pat = syntax::expression_at(ref parsed,arm.left);
 assert(pat.kind == 16 && pat.left == 2 && pat.right == 0 && pat.start == %d && pat.end == %d);
 assert(syntax::expression_text(ref parsed,3).len == 2 && syntax::expression_text(ref parsed,2)[0] == 118);
 assert(syntax::expression_at(ref parsed,7).kind == 17 && syntax::expression_text(ref parsed,7)[0] == 95);
 assert(syntax::statement_at(ref parsed,1).expression == 4 && syntax::statement_at(ref parsed,2).expression == 9);
 assert(syntax::declaration_at(ref parsed,1).expression == 12); 0
 } case .err(_): { 1 } }; assert(r == 0); }
`, strconv.Quote(input), strings.Index(input, "switch("), strings.LastIndex(input, ";"), strings.Index(input, "case"), strings.Index(input, " case .err"), strings.Index(input, ".ok"), strings.Index(input, ":{v}"))
	for _, expr := range []string{"switch(x){case _: {1}}", "switch(x){case .ok():{1}}", "switch(x){case .ok(a,b):{1}}", "switch(x){case .ok(x) {1}}", "switch(x){case .ok(x):1}", "switch(x){.ok:{1}}", "switch(x){case .ok{.x=}:{1}}", "switch(){case .ok:{1}}", "switch(x){case .ok{.=v}:{1}}", "switch(x){case .ok{.x v}:{1}}", "switch(x){case .ok{.x=v:{1}}", "switch(x){case .ok{.x=v,,}:{1}}", "switch(x){case .ok{.x=v .y=w}:{1}}"} {
		input := "module main; fn void bad(){return " + expr + ";} fn void kept(){}"
		ref, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || ref.Tree == nil || len(ref.Tree.Diagnostics) == 0 {
			t.Fatalf("invalid pattern oracle %q: %v", expr, err)
		}
		declarations := 2
		if expr == "switch(x){case .ok{.x=v:{1}}" {
			// An unclosed pattern consumes the following function during recovery.
			// Keep this explicit limitation separate from balanced-input recovery.
			declarations = 1
		}
		fmt.Fprintf(&p, "assert(switch(make(b%s)) { case .ok(parsed): { rejected(parsed,%d) } case .err(_): {1} }==0);\n", strconv.Quote(input), declarations)
	}
	p.WriteString(`
var usize n=818;
while(n<=819) {
 var bytes::Buffer data=bytes::empty(); discard bytes::append(ref mut data,b"module main;fn void f(){return switch(a+a+a){");
 var usize i=0; while(i<n) { discard bytes::append(ref mut data,b"case .ok(x):{1}"); i+=1; }
 discard bytes::append(ref mut data,b"};}");
 let i32 r=switch(make(bytes::view(ref data))) { case .ok(parsed): { assert(n==818 && syntax::problem_count(ref parsed)==0 && syntax::expression_count(ref parsed)==4096 && syntax::switch_case_count(ref parsed,4096)==818); 0 } case .err(status): { assert(n==819 && status==4); 0 } }; assert(r==0); n+=1;
}
n=2045;
while(n<=2046) {
 var bytes::Buffer data=bytes::empty(); discard bytes::append(ref mut data,b"module main;fn void f(){return switch(x){case .Pair{");
 var usize i=0; while(i<n) { discard bytes::append(ref mut data,b".x=v,"); i+=1; }
 discard bytes::append(ref mut data,b"}:{0}};}");
 let i32 r=switch(make(bytes::view(ref data))) {
  case .ok(parsed): { assert(n==2045 && syntax::problem_count(ref parsed)==0 && syntax::expression_count(ref parsed)==4096 && syntax::pattern_field_count(ref parsed,4092)==2045 && integrity::valid(ref parsed)); 0 }
  case .err(status): { assert(n==2046 && status==4); 0 }
 }; assert(r==0); n+=1;
}
n=31;
while(n<=32) {
 var bytes::Buffer data=bytes::empty(); discard bytes::append(ref mut data,b"module main;fn void f(){return ");
 var usize i=0; while(i<n) { discard bytes::append(ref mut data,b"switch(x){case .ok:{"); i+=1; }
 discard bytes::append(ref mut data,b"1");
 i=0; while(i<n) { discard bytes::append(ref mut data,b"}}"); i+=1; }
 discard bytes::append(ref mut data,b";}");
 let i32 r=switch(make(bytes::view(ref data))) { case .ok(parsed): { assert(n==31 && syntax::problem_count(ref parsed)==0); 0 } case .err(status): { assert(n==32 && status==4); 0 } }; assert(r==0); n+=1;
}
return 0; }`)
	runParserProbe(t, p.String())
}

// The input is parsed, never type-checked or executed. Native execution below
// exercises the candidate parser and its assertions, not the input program.
func testParserEnumSyntaxComposition(t *testing.T) {
	input := "module main; /* 한글: byte offsets */ enum Item { Empty; Pair { i32 left; i32 right; }; }\r\n" +
		"fn void inspect(){return switch(. /* construct */ Pair { .right=22, .left=20, }){case . /* match */ Pair { .left=first, .right=second, }:{first}};}"
	oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || oracle.Tree == nil || oracle.AST == nil || len(oracle.Tree.Diagnostics) != 0 {
		t.Fatalf("enum composition oracle: %v", err)
	}
	if len(oracle.AST.Enums) != 1 || len(oracle.AST.Functions) != 1 || len(oracle.AST.Enums[0].Variants) != 2 {
		t.Fatal("enum composition declaration shape")
	}
	e := oracle.AST.Enums[0]
	v := e.Variants[1]
	sw := oracle.AST.Functions[0].Body.Statements[0].Expression
	if sw == nil || len(sw.Children) != 2 || len(sw.Children[1].Children) != 2 {
		t.Fatal("enum composition switch shape")
	}
	record, pattern := sw.Children[0], sw.Children[1].Children[0]
	if len(v.Fields) != 2 || len(record.Children) != 3 || len(pattern.Children) != 3 {
		t.Fatal("enum composition field shape")
	}
	var p strings.Builder
	p.WriteString("fn i32 main(){ return switch(make(b" + strconv.Quote(input) + ")){case .ok(parsed):{\n")
	p.WriteString("assert(syntax::problem_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==3 && syntax::field_count(ref parsed)==4 && integrity::valid(ref parsed));\n")
	fmt.Fprintf(&p, "{let d=syntax::declaration_at(ref parsed,1); assert(d.kind==11 && d.start==%d && d.end==%d && d.name_start==%d && d.name_end==%d && d.field_start==0 && d.field_count==4);}\n", e.Span.StartByte, e.Span.EndByte, e.NameSpan.StartByte, e.NameSpan.EndByte)
	for i, member := range e.Variants {
		id := []int{0, 3}[i]
		fmt.Fprintf(&p, "{let m=syntax::field_at(ref parsed,%d); assert(m.kind==3 && m.start==%d && m.end==%d && m.name_start==%d && m.name_end==%d && m.payload_count==%d);}\n", id, member.Span.StartByte, member.Span.EndByte, member.NameSpan.StartByte, member.NameSpan.EndByte, len(member.Fields))
	}
	p.WriteString("assert(syntax::field_at(ref parsed,3).payload_start==1);\n")
	for i, f := range v.Fields {
		fmt.Fprintf(&p, "{let f=syntax::field_at(ref parsed,%d); assert(f.kind==4 && f.start==%d && f.end==%d && f.name_start==%d && f.name_end==%d);}\n", i+1, f.Span.StartByte, f.Span.EndByte, f.NameSpan.StartByte, f.NameSpan.EndByte)
	}
	p.WriteString("let root=syntax::declaration_at(ref parsed,2).expression; let s=syntax::expression_at(ref parsed,root);\n")
	fmt.Fprintf(&p, "assert(s.kind==15 && s.start==%d && s.end==%d && syntax::switch_case_count(ref parsed,root)==1);\n", sw.Span.StartByte, sw.Span.EndByte)
	p.WriteString("let constructor=s.left; let r=syntax::expression_at(ref parsed,constructor); let arm=syntax::switch_case(ref parsed,root,0); let matched=syntax::expression_at(ref parsed,arm).left; let pat=syntax::expression_at(ref parsed,matched);\n")
	fmt.Fprintf(&p, "assert(r.kind==10 && r.start==%d && r.end==%d && syntax::record_field_count(ref parsed,constructor)==2);\n", record.Span.StartByte, record.Span.EndByte)
	fmt.Fprintf(&p, "assert(pat.kind==32 && pat.start==%d && pat.end==%d && syntax::pattern_field_count(ref parsed,matched)==2);\n", pattern.Span.StartByte, pattern.Span.EndByte)
	textCheck := func(node, want string) {
		fmt.Fprintf(&p, "assert(syntax::expression_text(ref parsed,%s).len==%d);\n", node, len(want))
		for i, b := range []byte(want) {
			fmt.Fprintf(&p, "assert(syntax::expression_text(ref parsed,%s)[%d]==%d);\n", node, i, b)
		}
	}
	callee := record.Children[0]
	fmt.Fprintf(&p, "{let n=syntax::expression_at(ref parsed,r.left); assert(n.kind==28 && n.start==%d && n.end==%d);}\n", callee.Span.StartByte, callee.Span.EndByte)
	textCheck("r.left", strings.TrimPrefix(callee.Text, "."))
	textCheck("matched", strings.TrimPrefix(pattern.Text, "."))
	for i := 0; i < 2; i++ {
		f, binding := record.Children[i+1], pattern.Children[i+1]
		if len(f.Children) != 1 || binding.Cooked == "" {
			t.Fatal("enum composition initializer or binding shape")
		}
		p.WriteString("{\n")
		fmt.Fprintf(&p, "let a=syntax::record_field(ref parsed,constructor,%d); let b=syntax::pattern_field(ref parsed,matched,%d); let a_node=syntax::expression_at(ref parsed,a); let b_node=syntax::expression_at(ref parsed,b);\n", i, i)
		fmt.Fprintf(&p, "assert(a_node.kind==11 && a_node.start==%d && a_node.end==%d && b_node.kind==33 && b_node.start==%d && b_node.end==%d);\n", f.Span.StartByte, f.Span.EndByte, binding.Span.StartByte, binding.Span.EndByte)
		value := f.Children[0]
		fmt.Fprintf(&p, "{let n=syntax::expression_at(ref parsed,a_node.left); assert(n.kind==1 && n.start==%d && n.end==%d);}\n", value.Span.StartByte, value.Span.EndByte)
		end := binding.Span.EndByte
		start := end - len(binding.Cooked)
		if input[start:end] != binding.Cooked {
			t.Fatal("enum composition binding byte span")
		}
		fmt.Fprintf(&p, "{let n=syntax::expression_at(ref parsed,b_node.left); assert(n.kind==17 && n.start==%d && n.end==%d);}\n", start, end)
		textCheck("a", f.Text)
		textCheck("b", binding.Text)
		textCheck("b_node.left", binding.Cooked)
		p.WriteString("}\n")
	}
	p.WriteString("assert(integrity::valid(ref parsed)); 0 } case .err(_):{1}};}")
	runParserProbe(t, p.String())
}
