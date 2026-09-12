package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserResources(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, decl := range []string{"resource R {}", "resource R {u8 x;}", "pub resource R {cleanup {}}", "@tag pub resource R {pub Box<u8> x;ref u8[] y;cleanup {/*empty*/}}"} {
		input := "module main;" + decl
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("resource oracle %q: %v", decl, err)
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0 && syntax::declaration_at(ref parsed,1).kind==6);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	input := "module main;@tag pub resource R {pub u8 x; cleanup {/*empty*/}} resource S {}"
	fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
 let syntax::Declaration d=syntax::declaration_at(ref parsed,1);
 assert(d.kind==6 && d.public && d.attribute_count==1 && d.field_start==0 && d.field_count==1 && d.body==0 && d.result_type==0);
 assert(d.start==%d && d.end==%d && d.name_start==%d && d.name_end==%d && d.cleanup_start==%d && d.cleanup_end==%d);
 assert(syntax::field_at(ref parsed,0).public && syntax::type_count(ref parsed)==1);
 let syntax::Declaration absent=syntax::declaration_at(ref parsed,2);
 assert(absent.kind==6 && absent.cleanup_start==0 && absent.cleanup_end==0 && absent.field_start==1 && absent.field_count==0);
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), strings.Index(input, "@"), strings.Index(input, " resource S"), strings.Index(input, "R {"), strings.Index(input, "R {")+1, strings.Index(input, "cleanup"), strings.Index(input, "}}")+1)
	for _, bad := range []string{"resource R<T,,> {}", "pub resource R {u8 x}", "resource R {u8 x;cleanup {assert();}}", "resource R {cleanup {} u8 x;}", "resource R {cleanup {} cleanup {}}", "resource R {cleanup}", "resource R {cleanup { }", "resource R {@tag cleanup {}}", "struct R {cleanup {}}"} {
		input := "module main;@bad " + bad + " @kept resource Kept {}"
		// Leave an unmatched resource brace out of the recovery-to-sibling cases.
		if bad == "resource R {cleanup { }" {
			continue
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && syntax::field_count(ref parsed)==0 && syntax::type_count(ref parsed)==0 && syntax::attribute_count(ref parsed)==1 && syntax::declaration_at(ref parsed,1).kind==6);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	for _, input := range []string{"module main;resource R {cleanup { }", "module main;resource R {} import x as x;"} {
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	p.WriteString(`
var usize n=1024;while(n<=1025){
 var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;struct S {u8 x;} resource R {");
 var usize i=1;while(i<n){discard bytes::append(ref mut data,b"u8 x;");i+=1;}
 discard bytes::append(ref mut data,b"cleanup {}}");
 let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(n==1024 && syntax::problem_count(ref parsed)==0 && syntax::field_count(ref parsed)==1024);0}case .err(status):{assert(n==1025 && status==4);0}};assert(r==0);n+=1;
}
return 0;}`)
	runParserProbe(t, p.String())
}
