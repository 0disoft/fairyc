package compiler_test

import (
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"os"
	"testing"
)

func TestMutableFieldBorrowRoundTrip(t *testing.T) {
	source, err := os.ReadFile("testdata/mutable_field_borrow.fy")
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(source))})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v %+v", err, result.Semantic)
	}
	encoded, err := mircodec.Encode(result.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for name, program := range map[string]mir.Program{"fresh": result.MIR, "decoded": decoded} {
		t.Run(name, func(t *testing.T) {
			outcome, err := interpreter.Execute(program, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
				t.Fatalf("execution: %+v %v", outcome, err)
			}
		})
	}
}
func TestMutableFieldBorrowRejections(t *testing.T) {
	const prelude = `module main;struct Span{i64 start;i64 end;}struct Token{Span position;i64 sibling;}
 fn Token make(){return Token{.position=Span{.start=3,.end=4},.sibling=7};}
 fn ref mut Span project(ref mut Token token){return ref mut token.position;}
 fn void edit(ref mut Span s){s.start=9;}
 `
	for name, body := range map[string]string{
		"immutable":          `fn i32 main(){let Token t=make();let ref mut Span s=ref mut t.position;edit(s);return 0;}`,
		"shared":             `fn ref mut Span bad(ref Token t){return ref mut t.position;}`,
		"temporary":          `fn i32 main(){let ref mut Span s=ref mut make().position;edit(s);return 0;}`,
		"local-escape":       `fn ref mut Span bad(){var Token t=make();return ref mut t.position;}`,
		"value-param-escape": `fn ref mut Span bad(Token t){return ref mut t.position;}`,
		"sibling":            `fn i32 main(){var Token t=make();let ref mut Span s=ref mut t.position;t.sibling=8;edit(s);return 0;}`,
		"owner-read":         `fn i32 main(){var Token t=make();let ref mut Span s=ref mut t.position;assert(t.sibling==7);edit(s);return 0;}`,
		"owner-copy":         `fn i32 main(){var Token t=make();let ref mut Span s=ref mut t.position;let Token c=t;edit(s);return 0;}`,
		"owner-replace":      `fn i32 main(){var Token t=make();let ref mut Span s=ref mut t.position;t=make();edit(s);return 0;}`,
		"sibling-loans":      `fn i32 main(){var Token t=make();let ref mut i64 a=ref mut t.position.start;let ref mut i64 b=ref mut t.position.end;*b=8;*a=9;return 0;}`,
		"parent-read":        `fn void bad(ref mut Token t){let ref mut Span s=ref mut t.position;assert(t.sibling==7);edit(s);}`,
		"parent-alias":       `fn void bad(ref mut Token t){let ref mut Token a=t;let ref mut Span s=ref mut t.position;a.sibling=8;edit(s);}`,
		"returned-parent":    `fn void bad(ref mut Token t){let ref mut Span s=project(t);t.sibling=8;edit(s);}`,
		"returned-alias":     `fn void bad(ref mut Token t){let ref mut Token a=t;let ref mut Span s=project(t);a.sibling=8;edit(s);}`,
		"child-alias":        `fn void bad(ref mut Token t){let ref mut Span s=ref mut t.position;let ref mut Span a=s;let ref mut i64 n=ref mut s.start;a.end=8;*n=9;}`,
		"args-first-child":   `fn void both(ref mut Span s,ref mut Token t){} fn void bad(ref mut Token t){both(ref mut t.position,t);}`,
		"args-first-parent":  `fn void both(ref mut Token t,ref mut Span s){} fn void bad(ref mut Token t){both(t,ref mut t.position);}`,
		"tagged-parent":      `fn Option<ref mut Span> wrap(ref mut Token t){return .some(project(t));} fn void bad(ref mut Token t){let Option<ref mut Span> s=wrap(t);t.sibling=8;switch(s){case .some(v):{edit(v);}case .none:{}};}`,
		"shared-held":        `fn void bad(ref mut Token t){let ref Token v=read(t);let ref mut Span s=ref mut t.position;edit(s);assert(v.sibling==7);} fn ref Token read(ref Token t){return t;}`,
		"resource":           `resource Owner{Span position;cleanup{}}fn ref mut Span bad(ref mut Owner t){return ref mut t.position;}`,
		"copied-element":     `fn void bad(ref mut Token[] t){let ref mut Token s=ref mut t[0];}`,
		"nonplain":           `struct Owner{Span position;u8[2] bytes;}fn ref mut Span bad(ref mut Owner t){return ref mut t.position;}`,
	} {
		t.Run(name, func(t *testing.T) {
			result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", prelude+body)})
			if err != nil {
				t.Fatal(err)
			}
			if result.Verified || len(result.Semantic.Diagnostics)+len(result.Semantic.Unsupported) == 0 {
				t.Fatalf("accepted: %+v", result.Semantic)
			}
			switch name {
			case "sibling", "owner-read", "owner-copy", "owner-replace", "sibling-loans", "parent-read", "parent-alias", "returned-parent", "returned-alias", "child-alias", "args-first-child", "args-first-parent", "tagged-parent", "shared-held":
				found := false
				for _, diagnostic := range result.Semantic.Diagnostics {
					found = found || diagnostic.Code == "E2101"
				}
				if !found {
					t.Fatalf("missing exclusive conflict: %+v", result.Semantic)
				}
			case "local-escape":
				found := false
				for _, diagnostic := range result.Semantic.Diagnostics {
					found = found || diagnostic.Code == "E2102"
				}
				if !found {
					t.Fatalf("missing local escape: %+v", result.Semantic)
				}
			}
			if len(result.Semantic.Diagnostics) > 0 && result.Semantic.Diagnostics[0].Primary.EndByte <= result.Semantic.Diagnostics[0].Primary.StartByte {
				t.Fatal("missing diagnostic span")
			}
		})
	}
}
