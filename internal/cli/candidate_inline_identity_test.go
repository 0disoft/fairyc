package cli

import (
	"encoding/binary"
	"encoding/hex"
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

func candidateInlineProbe(t *testing.T, modules []string, imports, body string) {
	t.Helper()
	if (runtime.GOOS != "windows" && runtime.GOOS != "linux") || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows/Linux AMD64")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, append(modules, "integrity", "file_syntax")...)
	at := strings.Index(parserProbePrelude, "fn Result")
	writeTestSource(t, root, "src/main.fy", parserProbePrelude[:at]+imports+"\n"+parserProbePrelude[at:]+"\nfn i32 main(){"+body+"return 0;}\n")
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		code, out, diagnostics := runForTest(args, root)
		if code != ExitSuccess || out != "" || diagnostics != "" {
			t.Log(body)
			t.Fatalf("%v: %d %q %s", args, code, out, diagnostics)
		}
	}
}

func TestCandidateInlineIdentity(t *testing.T) {
	var probe strings.Builder
	for _, decl := range []string{
		"@inline pub fn i32 add(i32 x){return x+1;}",
		"@inline pub fn i32 add(i32 renamed){return (renamed+0x1);}",
		"@inline pub fn i32 add(i32 x){return x+2;}",
		"@inline pub fn i32 add(i32 x){return -x+0b10;}",
		"@inline pub fn i32 add(){return -2147483648;}",
		"@inline pub fn i32 add(i32 a,i32 b,i32 c,i32 d){return +(a*b)/(c-d)%7;}",
	} {
		input := "module sample;" + decl
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(input)), source.DefaultLimits())
		if err != nil || parsed.AST == nil {
			t.Fatalf("parse control %v", err)
		}
		checked, err := compiler.Check([]semantic.File{{Path: "sample.fy", Module: parsed.AST}})
		if err != nil || !checked.Verified {
			t.Fatalf("Go control: %s: %v %+v", input, err, checked.Semantic.Diagnostics)
		}
		var id string
		for _, fn := range checked.MIR.Functions {
			if fn.InlineBody != "" {
				id = fn.InlineBody
			}
		}
		raw, err := hex.DecodeString(strings.TrimPrefix(id, "fyinlinefn1-sha256-"))
		if err != nil || len(raw) != 32 {
			t.Fatal("missing Go identity", id)
		}
		fmt.Fprintf(&probe, "{let p=switch(make(b%s)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};let info=inline::read(ref p,1);assert(info.annotated && info.code==0);", strconv.Quote(input))
		for i := 0; i < 8; i++ {
			fmt.Fprintf(&probe, "assert(info.hash[%d]==%d);", i, binary.BigEndian.Uint32(raw[i*4:]))
		}
		probe.WriteString("}\n")
	}
	for _, decl := range []string{
		"@inline fn i32 f(){return 1;}", "@inline @inline pub fn i32 f(){return 1;}", "@inline(1) pub fn i32 f(){return 1;}",
		"@inline pub fn i64 f(){return 1;}", "@inline pub fn i32 f(i64 x){return 1;}", "@inline pub unsafe fn i32 f(){return 1;}",
		"@inline @effects(console) pub fn i32 f(){return 1;}", "@inline pub fn i32 f(){var x=1;return x;}",
		"@inline pub fn i32 f(){return f();}", "@inline pub fn i32 f(i32 x){return x<<1;}",
		"@inline pub fn i32 f(i32 a,i32 b,i32 c,i32 d,i32 e){return a;}",
		"@inline pub fn i32 f(i32 x){return " + strings.Repeat("x+", 32) + "x;}",
	} {
		fmt.Fprintf(&probe, "{let p=switch(make(b%s)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};assert(inline::read(ref p,1).code!=0);}\n", strconv.Quote("module sample;"+decl))
	}
	candidateInlineProbe(t, []string{"module_inline_declarations"}, "import module_inline_declarations as inline;", probe.String())
}
