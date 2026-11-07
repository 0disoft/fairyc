package cli

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func inlineCallsChain(count int) string {
	var out strings.Builder
	out.WriteString("module sample;")
	for i := 0; i < count; i++ {
		value := "x"
		if i+1 < count {
			value = fmt.Sprintf("f%d(x)", i+1)
		}
		fmt.Fprintf(&out, "@inline pub fn i32 f%d(i32 x){return %s;}", i, value)
	}
	return out.String()
}

func TestCandidateInlineCallsIdentity(t *testing.T) {
	leaf := "@inline pub fn i32 leaf(i32 x){return x+1;}"
	caller := "@inline pub fn i32 outer(i32 x){return leaf(x);}"
	argument := strings.Repeat("x+", 31) + "x"
	nested := func(calls int) string { return strings.Repeat("leaf(", calls) + "x" + strings.Repeat(")", calls) }
	cases := []struct {
		name, text string
		accepted   bool
	}{
		{"forward", "module sample;" + caller + leaf, true},
		{"backward", "module sample;" + leaf + caller, true},
		{"locals", "module sample;" + leaf + "@inline pub fn i32 outer(i32 x){let a=leaf(x);let b=leaf(a);return leaf(b);}", true},
		{"nested", "module sample;" + leaf + "@inline pub fn i32 outer(i32 x){return leaf(leaf(x))+leaf(x);}", true},
		{"exact-nested-nodes", "module sample;" + leaf + "@inline pub fn i32 outer(i32 x){return " + nested(63) + ";}", true},
		{"one-over-nested-nodes", "module sample;" + leaf + "@inline pub fn i32 outer(i32 x){return " + nested(64) + ";}", false},
		{"zero-arguments", "module sample;@inline pub fn i32 outer(){return leaf();}@inline pub fn i32 leaf(){return 42;}", true},
		{"diamond", "module sample;" + leaf + "@inline pub fn i32 left(i32 x){return leaf(x);}@inline pub fn i32 right(i32 x){return leaf(x);}@inline pub fn i32 outer(i32 x){return left(x)+right(x);}", true},
		{"public-label-closed", "module sample;" + leaf + "@inline pub fn i32 outer(i32 x){return leaf(.x=x);}", false},
		{"exact-nodes", "module sample;" + leaf + "@inline pub fn i32 outer(i32 x){return leaf(" + argument + ");}", true},
		{"one-over-nodes", "module sample;" + leaf + "@inline pub fn i32 outer(i32 x){return leaf((" + argument + "));}", false},
		{"exact-graph", inlineCallsChain(64), true},
		{"one-over-graph", inlineCallsChain(65), false},
		{"ordinary", "module sample;" + caller + "pub fn i32 leaf(i32 x){return x+1;}", false},
		{"private", "module sample;" + caller + "@inline fn i32 leaf(i32 x){return x+1;}", false},
		{"effectful", "module sample;" + caller + "@inline @effects(console) pub fn i32 leaf(i32 x){return x+1;}", false},
		{"unsafe", "module sample;" + caller + "@inline pub unsafe fn i32 leaf(i32 x){return x+1;}", false},
		{"self-cycle", "module sample;@inline pub fn i32 outer(i32 x){return outer(x);}", false},
		{"mutual-cycle", "module sample;@inline pub fn i32 outer(i32 x){return leaf(x);}@inline pub fn i32 leaf(i32 x){return outer(x);}", false},
		{"parameter-shadow", "module sample;" + leaf + "@inline pub fn i32 outer(i32 leaf){return leaf(1);}", false},
		{"local-shadow", "module sample;" + leaf + "@inline pub fn i32 outer(){let leaf=1;return leaf(1);}", false},
		{"wrong-arity", "module sample;" + leaf + "@inline pub fn i32 outer(){return leaf();}", false},
		{"generic", "module sample;" + caller + "@inline pub fn i32 leaf<T>(i32 x){return x;}", false},
	}
	var probe, markers, driver strings.Builder
	for caseIndex, test := range cases {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(test.text)), source.DefaultLimits())
		if err != nil || parsed.AST == nil {
			t.Fatalf("%s parser: %v", test.name, err)
		}
		checked, err := compiler.Check([]semantic.File{{Path: "sample.fy", Module: parsed.AST}})
		if err != nil || checked.Verified != test.accepted {
			t.Fatalf("%s Go oracle: verified=%t err=%v diagnostics=%+v", test.name, checked.Verified, err, checked.Semantic.Diagnostics)
		}
		// Separate source cases and marker assertions into bounded functions.
		// A single main with every graph marker exceeds normal MIR limits.
		fmt.Fprintf(&driver, "assert(identity_case_%d()==0);", caseIndex)
		if test.name == "one-over-nested-nodes" {
			// The existing body parser rejects the 64th nested call first
			// (bodies.fy depth >= 64, status 4). Do not widen that bound or
			// count this as an inline-admission observation. Flat 64/65-node
			// cases below reach inline admission independently.
			fmt.Fprintf(&probe, "fn i32 identity_case_%d(){let code=switch(make(b%s)){case .ok(p):{assert(false);return 1;}case .err(code):{code}};assert(code==4);return 0;}\n", caseIndex, strconv.Quote(test.text))
			continue
		}
		fmt.Fprintf(&probe, "fn i32 identity_case_%d(){{let p=switch(make(b%s)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};", caseIndex, strconv.Quote(test.text))
		if !test.accepted {
			// Check the caller, not an invalid callee declaration by itself.
			declaration := 1
			if strings.HasPrefix(test.text, "module sample;"+leaf) {
				declaration = 2
			}
			fmt.Fprintf(&probe, "assert(inline::read(ref p,%d).code!=0);}return 0;}\n", declaration)
			continue
		}
		for i, fn := range checked.Semantic.HIR.Modules[0].Functions {
			raw, err := hex.DecodeString(strings.TrimPrefix(fn.InlineBody, "fyinlinefn1-sha256-"))
			if err != nil || len(raw) != 32 {
				t.Fatalf("%s %s missing marker: %s", test.name, fn.Name, fn.InlineBody)
			}
			fmt.Fprintf(&probe, "identity_marker_%d_%d(ref p);", caseIndex, i)
			fmt.Fprintf(&markers, "fn void identity_marker_%d_%d(ref syntax::Parsed p){let info=inline::read(p,%d);assert(info.annotated && info.code==0);", caseIndex, i, i+1)
			for j := 0; j < 8; j++ {
				fmt.Fprintf(&markers, "assert(info.hash[%d]==%d);", j, binary.BigEndian.Uint32(raw[j*4:]))
			}
			markers.WriteString("}\n")
		}
		probe.WriteString("}return 0;}\n")
	}
	// This literal tuple is independent of either compiler's canonical encoder.
	const vector = `{"Version":"fairyc.inline-function/v3","Arity":1,"Root":{"Op":"call","Value":"leaf","Children":[{"Op":"path","Value":"0","Children":null}]}}`
	want := sha256.Sum256([]byte(vector))
	input := "module sample;" + caller + leaf
	parsed, _ := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(input)), source.DefaultLimits())
	checked, err := compiler.Check([]semantic.File{{Path: "sample.fy", Module: parsed.AST}})
	if err != nil || !checked.Verified || checked.Semantic.HIR.Modules[0].Functions[0].InlineBody != fmt.Sprintf("fyinlinefn1-sha256-%x", want) {
		t.Fatalf("independent v3 vector: %v %+v", err, checked.Semantic.Diagnostics)
	}
	t.Logf("%d source cases (one existing parser-depth refusal); every accepted declaration marker compared with Go; independent v3 vector", len(cases))
	imports := "import module_inline_declarations as inline;\n" + markers.String() + probe.String()
	if path := os.Getenv("FAIRYC_INLINE_CALLS_IDENTITY_SOURCE"); path != "" {
		at := strings.Index(parserProbePrelude, "fn Result")
		text := parserProbePrelude[:at] + imports + "\n" + parserProbePrelude[at:] + "\nfn i32 main(){" + driver.String() + "return 0;}\n"
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	candidateInlineProbe(t, []string{"module_inline_declarations"}, imports, driver.String())
}
