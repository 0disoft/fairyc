package cli

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestCandidateInlineLocalsIdentity(t *testing.T) {
	var probe strings.Builder
	for _, decl := range []string{
		"@inline pub fn i32 choose(i32 x){return if (x < 0) { 1/0 } else if ((x == 0)) { 41 } else { x+1 };}",
		"@inline pub fn i32 choose(i32 x){return if (!(x<0) && (x==0 || 42/x==1)) { 42 } else { 0 };}",
		"@inline pub fn i32 choose(i32 x){let i32 y=if (true) { x } else { 0 };return if (y != 0) { y } else { 42 };}",
		"@inline pub fn i32 choose(i32 x){var i32 y=if (true) { x } else { 0 };return if (y != 0) { y } else { 42 };}",
		"@inline pub fn i32 choose(i32 x){return if (true) { let i32 y=x; if(y==0){let z=42;z}else{y} } else { let y=0; y };}",
		"@inline pub fn i32 choose(i32 x){var i32 y=0;if(x==0){return 42;}if(x<0){y=2;}else if(x==1){let v=1;y=v+1;}else{y=10/x;}return y;}",
		"@inline pub fn i32 choose(i32 x){if(x==0){return 42;}else{return 1;};}",
		"@inline pub fn i32 choose(i32 x){if(x==0){}return 42;}",
		"@inline pub fn i32 choose(i32 x){var i32 y=0;for(var i32 i=0;i<x;i=i+1){if(i==2){continue;}if(i==6){break;}y=y+i;}while(y<42 && x>0){y=y+1;}return y;}",
		"@inline pub fn i32 choose(i32 x){if(x==0){for(;;){break;}}return 42;}",
		"@inline pub fn i32 add(i32 x){var i32 value=x;value=value+2;let old=value;value=old+value;return value/2;}",
		"@inline pub fn i32 add(i32 x){let i32 a=x+2;let b=a*2;return b/2;}",
		"@inline pub fn i32 add(i32 renamed){let left=(renamed+0x2);let i32 right=left*0b10;return (right/2);}",
		"@inline pub fn i32 add(){let i32 minimum=-2147483648;return minimum;}",
		"@inline pub fn i32 add(){let i32 unused=1/0;return 42;}",
		"@inline pub fn i32 add(i32 x){let i32 y=x;let i32 z=y;return z+y;}",
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
		loop := strings.HasPrefix(id, hir.InlineLoopPrefix)
		if !hir.ValidInlineFunctionID(id) {
			t.Fatal("missing Go identity", id)
		}
		raw, err := hex.DecodeString(id[len(hir.InlineFunctionPrefix):])
		if err != nil || len(raw) != 32 {
			t.Fatal("missing Go identity", id)
		}
		fmt.Fprintf(&probe, "{let p=switch(make(b%s)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};let info=inline::read(ref p,1);assert(info.annotated && info.code==0 && info.loop==%t);", strconv.Quote(input), loop)
		for i := 0; i < 8; i++ {
			fmt.Fprintf(&probe, "assert(info.hash[%d]==%d);", i, binary.BigEndian.Uint32(raw[i*4:]))
		}
		probe.WriteString("}\n")
	}
	for _, decl := range []string{
		"@inline pub fn i32 choose(i32 x){return if ((x < 0) == true) { 41 } else { 42 };}",
		"@inline pub fn i32 choose(){return if (true) { var i32 value=42; value } else { 0 };}",
		"@inline pub fn i32 choose(){return if (true) { let i32 value=42; value } else { value };}",
		"@inline pub fn i32 choose(i32 x){if(x==0){let v=1;}else{return v;}return 42;}",
		"@inline pub fn i32 choose(i32 x){let y=0;if(x==0){y=1;}return y;}",
		"@inline pub fn i32 f(){let i32 x=h();return x;}",
		"@inline pub fn i32 f(){let i32 x=1;x=2;return x;}",
		"@inline pub fn i32 f(){let i64 x=1;return 42;}",
		"@inline pub fn i32 f(){let i32 x=1;discard x;return x;}",
		"@inline pub fn i32 f(){let i32 x=y;let i32 y=2;return x;}",
		"@inline pub fn i32 f(){let i32 x=x;return x;}",
		"@inline fn i32 f(){return 1;}", "@inline @inline pub fn i32 f(){return 1;}", "@inline(1) pub fn i32 f(){return 1;}",
		"@inline pub fn i64 f(){return 1;}", "@inline pub fn i32 f(i64 x){return 1;}", "@inline pub unsafe fn i32 f(){return 1;}",
		"@inline @effects(console) pub fn i32 f(){return 1;}", "@inline pub fn i32 f(){var x=1;x+=1;return x;}",
		"@inline pub fn i32 f(){return f();}", "@inline pub fn i32 f(i32 x){return x<<1;}",
		"@inline pub fn i32 f(i32 a,i32 b,i32 c,i32 d,i32 e){return a;}",
		"@inline pub fn i32 f(i32 x){return " + strings.Repeat("x+", 32) + "x;}",
	} {
		fmt.Fprintf(&probe, "{let p=switch(make(b%s)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};assert(inline::read(ref p,1).code!=0);}\n", strconv.Quote("module sample;"+decl))
	}
	// Admission boundaries are independent of either compiler's hash output.
	for _, n := range []int{16, 17} {
		var body strings.Builder
		for j := 0; j < n; j++ {
			fmt.Fprintf(&body, "let i32 v%d=1;", j)
		}
		input := "module sample;@inline pub fn i32 f(){" + body.String() + "return 42;}"
		fmt.Fprintf(&probe, "{let p=switch(make(b%s)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};assert((inline::read(ref p,1).code==0)==%t);}\n", strconv.Quote(input), n == 16)
	}
	for _, extra := range []bool{false, true} {
		prefix := "let i32 unused=1;"
		if extra {
			prefix += "let i32 second=1;"
		}
		input := "module sample;@inline pub fn i32 f(){" + prefix + "return " + strings.Repeat("1+", 31) + "1;}"
		fmt.Fprintf(&probe, "{let p=switch(make(b%s)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};assert((inline::read(ref p,1).code==0)==%t);}\n", strconv.Quote(input), !extra)
	}
	candidateInlineProbe(t, []string{"module_inline_declarations"}, "import module_inline_declarations as inline;", probe.String())
}
