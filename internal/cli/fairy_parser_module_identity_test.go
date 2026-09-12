package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserModuleIdentity(t *testing.T) {
	var p strings.Builder
	p.WriteString(`fn Result<syntax::Parsed,i32> make_file(ref u8[] input,ref u8[] path){
  var bool allocated=true;
  let bytes::Buffer owned=switch(bytes::copy(input)){case .ok(value):{value}case .err(_):{allocated=false;bytes::empty()}};
  if(!allocated){return .err(3);}
  return switch(scanner::collect(owned,false,32768,1048576,256,16777216)){
   case .ok(stream):{file_syntax::parse(stream,512,path)}case .err(status):{.err(status)}
  };
 }
 fn i32 main(){`)
	for _, c := range []struct{ path, module string }{
		{"main.fy", "main"}, {"net/http.fy", "net::http"}, {`net\http.fy`, "net::http"},
		{"./net//http.fy/", "net /*gap*/ :: http"}, {"bad-name/../main.fy", "main"},
		{"net/../main.fy", "main"}, {"net/./http.fy", "net::http"},
		{"net/http.fy", "net::https"}, {"main.fy", "other"},
		{"../main.fy", "main"}, {"net/../../main.fy", "main"}, {"/main.fy", "main"},
		{`C:\main.fy`, "main"}, {"bad-name/main.fy", "main"}, {"main.txt", "main"},
		{"", "main"}, {".fy", "main"}, {"dir/..", "main"}, {"9main.fy", "main"}, {"한글.fy", "main"},
	} {
		input := "module " + c.module + ";"
		file := source.NewFile(c.path, []byte(input))
		oracle, err := frontend.AnalyzeSyntax(file, source.DefaultLimits())
		if err != nil || oracle.Tree == nil {
			t.Fatal(err)
		}
		want := 0
		if _, err := file.ExpectedModule(); err != nil {
			want = 6
		} else if len(oracle.Tree.Diagnostics) != 0 {
			want = 7
		}
		fmt.Fprintf(&p, `{let i32 status=switch(make_file(b%s,b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));0}case .err(error):{error}};assert(status==%d);}`, strconv.Quote(input), strconv.Quote(c.path), want)
	}
	p.WriteString(`
 // Byte-only callers intentionally do not acquire file identity checks.
 {let i32 status=switch(make(b"module net::https;")){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0);0}case .err(error):{error}};assert(status==0);}
 // Syntax errors still return an owned incomplete tree, not a mismatch error.
 {let i32 status=switch(make_file(b"module wrong;fn void bad( {}",b"main.fy")){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && integrity::valid(ref parsed));0}case .err(error):{error}};assert(status==0);}
 var bytes::Buffer long_path=bytes::empty();var usize n=0;
 while(n<32769){discard bytes::append_byte(ref mut long_path,97);n+=1;}
 assert(switch(make_file(b"module main;",bytes::view(ref long_path))){case .ok(_):{false}case .err(status):{status==4}});
 `)
	for _, n := range []int{1024, 1025} {
		path := strings.Repeat("a/", n-1) + "main.fy"
		want := 7
		if n == 1025 {
			want = 4
		}
		fmt.Fprintf(&p, `assert(switch(make_file(b"module main;",b%s)){case .ok(_):{false}case .err(status):{status==%d}});`, strconv.Quote(path), want)
	}
	p.WriteString("return 0;}")
	runParserProbe(t, p.String())
}

// Fixture transport prefixes a caller-supplied path line, leaving source bytes intact.
// Exact replacement guards fail if the shared byte-only driver changes shape.
func parserFileCorpusProgram(t *testing.T) string {
	t.Helper()
	program := parserCorpusProgram
	replace := func(old, next string) {
		if strings.Count(program, old) != 1 {
			t.Fatalf("file corpus driver anchor drift: %q", old)
		}
		program = strings.Replace(program, old, next, 1)
	}
	replace("import integrity as integrity;", "import integrity as integrity; import file_syntax as file_syntax;")
	replace("var bytes::Buffer input = bytes::empty();", `var bytes::Buffer input = bytes::empty();
 var bytes::Buffer path = bytes::empty(); var usize path_length = 0; var bool path_done = false;`)
	replace("if (length >= 32768)", `if (!path_done) {
   if (data[0] == 10) { path_done = true; continue; }
   if (path_length >= 32768) { return 4; }
   if (!switch(bytes::append_byte(ref mut path,data[0])) { case .ok(_): { true } case .err(_): { false } }) { return 3; }
   path_length += 1; continue;
  }
  if (length >= 32768)`)
	replace("syntax::parse(stream,512)", "file_syntax::parse(stream,512,bytes::view(ref path))")
	replace("assert(status == 3 || status == 4 || status == 5); number(if (status == 3) { 3 } else if (status == 4) { 4 } else { 5 });", "assert(status >= 3 && status <= 7); number(if (status == 3) { 3 } else if (status == 4) { 4 } else if (status == 5) { 5 } else if (status == 6) { 6 } else { 7 });")
	return program
}
