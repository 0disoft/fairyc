package cli

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

// Zero inline classes in the current candidate producer must describe an
// admitted no-inline source, never a silently discarded source annotation.
func TestCandidateInlineAdmissionBoundary(t *testing.T) {
	if (runtime.GOOS != "windows" && runtime.GOOS != "linux") || runtime.GOARCH != "amd64" {
		t.Skip("requires native Windows/Linux AMD64")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "syntax", "module_signatures", "integrity", "file_syntax")
	var program strings.Builder
	program.WriteString(parserProbePrelude)
	// Keep imports before declarations; parserProbePrelude already contains make.
	prefix := strings.Index(program.String(), "fn Result")
	text := program.String()
	program.Reset()
	program.WriteString(text[:prefix] + "import module_signatures as signatures; import std::vec as v;\n" + text[prefix:])
	program.WriteString("fn i32 main(){\n")
	for _, body := range []string{"return x+1;", "return (x+0x1);"} {
		input := "module sample; @inline pub fn i32 add(i32 x){" + body + "} fn i32 main(){return add(41);}"
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(input)), source.DefaultLimits())
		if err != nil || parsed.AST == nil {
			t.Fatalf("inline syntax %q: %v %+v", input, err, parsed.Tree.Diagnostics)
		}
		checked, err := compiler.Check([]semantic.File{{Path: "sample.fy", Module: parsed.AST}})
		if err != nil || !checked.Verified {
			t.Fatalf("Go inline control rejected: %v", err)
		}
		start := strings.Index(input, "@inline")
		fmt.Fprintf(&program, `{let parsed=switch(make(b%s)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed).is_ok);assert(switch(signatures::build(ref sources)){case .ok(_):{false}case .err(e):{e.code==3 && e.module_index==0 && e.start==%d && e.end==%d}});}`+"\n", strconv.Quote(input), start, start+len("@inline"))
	}
	input := "module sample; pub fn i32 add(i32 x){return x+1;} fn i32 main(){return add(41);}"
	fmt.Fprintf(&program, `{let parsed=switch(make(b%s)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed).is_ok);assert(switch(signatures::build(ref sources)){case .ok(_):{true}case .err(_):{false}});}`+"\n", strconv.Quote(input))
	program.WriteString("return 0;}\n")
	writeTestSource(t, root, "src/main.fy", program.String())
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		code, out, diagnostics := runForTest(args, root)
		if code != ExitSuccess || out != "" || diagnostics != "" {
			t.Fatalf("%v: %d %q %s", args, code, out, diagnostics)
		}
	}
}
