package semantic_test

import (
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"testing"
)

func TestCopiedMutableStructForFlow(t *testing.T) {
	for _, tc := range []struct{ name, body, code string }{
		{"read-copy", `fn void test(ref mut Token[] view){for(let Token item in view){assert(item.value>0);}}`, ""},
		{"independent-copy", `fn void test(ref mut Token[] view){for(let Token item in view){var Token copy=item;copy.value=9;copy.position.start=7;assert(item.value>0);}}`, ""},
		{"parent-write-with-live-copy", `fn void test(ref mut Token[] view){for(let Token item in view){view[0].value=9;assert(item.value>0);}}`, ""},
		{"whole-parent-replace", `fn void test(ref mut Token[] view){for(let Token item in view){view[0]=Token{.value=9,.position=Span{.start=0}};assert(item.value>0);}}`, ""},
		{"exclusive-child", `fn void edit(ref mut Token value){value.value=9;}fn void test(ref mut Token[] view){for(let Token item in view){edit(ref mut view[0]);assert(item.value>0);}}`, ""},
		{"shared-copy-child", `fn void test(ref mut Token[] view){for(let Token item in view){let ref Span copy=ref item.position;view[0].position.start=9;assert(copy.start>0);}}`, ""},
		{"returned-value", `fn Option<Token> first(ref mut Token[] view){for(let Token item in view){return .some(item);}return .none;}`, ""},
		{"retained-copy", `fn void test(ref mut Token[] view){var Option<Token> saved=.none;for(let Token item in view){saved=.some(item);view[0].value=9;}switch(saved){case .some(copy):{assert(copy.value>0);}case .none:{}};}`, ""},
		{"nested-same", `fn void test(ref mut Token[] view){for(let Token outer in view){for(let Token inner in view){assert(outer.value>0 && inner.value>0);}}}`, ""},
		{"item-store", `fn void test(ref mut Token[] view){for(let Token item in view){item.value=9;}}`, "E1008"},
		{"copy-escape", `fn Option<ref Token> first(ref mut Token[] view){for(let Token item in view){return .some(ref item);}return .none;}`, "E2102"},
		{"scope", `fn void test(ref mut Token[] view){for(let Token item in view){}assert(item.value>0);}`, "E0101"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;struct Span{usize start;}struct Token{i64 value;Span position;}" + tc.body})
			if tc.code != "" {
				for _, d := range r.Diagnostics {
					if d.Code == tc.code {
						return
					}
				}
				t.Fatalf("missing %s: %+v %+v", tc.code, r.Diagnostics, r.Unsupported)
			}
			if len(r.Diagnostics)+len(r.Unsupported) != 0 || r.CompletedThrough != semantic.PhaseFlow {
				t.Fatalf("analysis: %+v %+v", r.Diagnostics, r.Unsupported)
			}
			p, err := mir.Lower(r.HIR)
			if err != nil {
				t.Fatal(err)
			}
			if d := mir.Verify(p); len(d) != 0 {
				t.Fatal(d)
			}
		})
	}
}
func TestCopiedMutableStructForBoundary(t *testing.T) {
	for name, source := range map[string]string{
		"mutable-descriptor": `fn void test(ref mut Token[] view){var ref mut Token[] alias=view;for(let Token item in alias){}}`,
		"temporary":          `fn void test(ref mut Token[] view){for(let Token item in view[..][..]){}}`,
		"wrong-nominal":      `struct Other{i64 value;}fn void test(ref mut Token[] view){for(let Other item in view){}}`,
		"wrong-scalar":       `fn void test(ref mut Token[] view){for(let i64 item in view){}}`,
		"resource-graph":     `resource Owned{i64 value;cleanup{}}fn void test(ref mut Owned[] view){for(let Owned item in view){}}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;struct Token{i64 value;}" + source})
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("unadmitted copied iteration accepted")
			}
		})
	}
}
