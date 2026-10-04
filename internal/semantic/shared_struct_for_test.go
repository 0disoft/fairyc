package semantic_test

import (
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"testing"
)

func TestSharedStructForFlow(t *testing.T) {
	for _, tc := range []struct{ name, body, code string }{
		{"shared-source", `fn void test(ref Token[] view){for(let ref Token item in view){assert(item.value==view[0].value);}}`, ""},
		{"exclusive-source", `fn void test(ref mut Token[] view){for(let ref Token item in view){assert(item.value==view[0].value);}view[0].value=9;}`, ""},
		{"nested-fields", `fn void test(ref Token[] view){for(let ref Token item in view){let ref Span child=ref item.position;assert(child.end>=child.start);}}`, ""},
		{"return", `fn Option<ref Token> find(ref Token[] view){for(let ref Token item in view){return .some(item);}return .none;}`, ""},
		{"exclusive-return", `fn Option<ref Token> find(ref mut Token[] view){for(let ref Token item in view){return .some(item);}return .none;}`, ""},
		{"field-return", `fn Option<ref Span> find(ref Token[] view){for(let ref Token item in view){return .some(ref item.position);}return .none;}`, ""},
		{"retained-shared", `fn void test(ref Token[] view){var Option<ref Token> saved=.none;for(let ref Token item in view){switch(saved){case .some(old):{assert(old.value>0);}case .none:{}};saved=.some(item);}switch(saved){case .some(old):{assert(old.value>0);}case .none:{}};}`, ""},
		{"nested-same", `fn void test(ref Token[] view){for(let ref Token a in view){for(let ref Token b in view){assert(a.value==b.value);}}}`, ""},
		{"shared-read-call", `fn void read(ref Token value){}fn void test(ref mut Token[] view){for(let ref Token item in view){read(item);read(ref view[0]);}}`, ""},
		{"parent-write-after-last-use", `fn void test(ref mut Token[] view){for(let ref Token item in view){assert(item.value>0);view[0].value=9;break;}}`, "E2101"},
		{"whole-parent-store", `fn void test(ref mut Token[] view){for(let ref Token item in view){assert(item.value>0);view[0]=Token{.value=9,.position=Span{.start=0,.end=1}};return;}}`, "E2101"},
		{"alias-write", `fn void test(ref mut Token[] view){let ref mut Token[] alias=view;for(let ref Token item in view){assert(item.value>0);alias[0].value=9;}}`, "E2101"},
		{"exclusive-element", `fn void edit(ref mut Token value){}fn void test(ref mut Token[] view){for(let ref Token item in view){assert(item.value>0);edit(ref mut view[0]);}}`, "E2101"},
		{"retained-return-write", `fn Option<ref Token> find(ref mut Token[] view){for(let ref Token item in view){return .some(item);}return .none;}fn void test(ref mut Token[] view){let Option<ref Token> found=find(view);view[0].value=9;switch(found){case .some(item):{assert(item.value>0);}case .none:{}};}`, "E2101"},
		{"shared-field-store", `fn void test(ref Token[] view){for(let ref Token item in view){item.value=9;}}`, "E1008"},
		{"scope", `fn void test(ref Token[] view){for(let ref Token item in view){}assert(item.value>0);}`, "E0101"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;struct Span{usize start;usize end;}struct Token{i64 value;Span position;}" + tc.body})
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

func TestSharedStructForBoundary(t *testing.T) {
	for name, body := range map[string]string{
		"same-layout-other":          `struct Other{i64 value;}fn void test(ref Token[] view){for(let ref Other item in view){}}`,
		"mutable-descriptor":         `fn void test(ref Token[] view){var ref Token[] alias=view;for(let ref Token item in alias){}}`,
		"temporary":                  `fn void test(ref Token[] view){for(let ref Token item in view[..][..]){}}`,
		"field-root":                 `struct Holder{ref Token[] view;}fn void test(ref Holder holder){for(let ref Token item in holder.view){}}`,
		"exclusive-item-from-shared": `fn void test(ref Token[] view){for(let ref mut Token item in view){}}`,
		"whole-shared-store":         `fn void test(ref Token[] view){for(let ref Token item in view){*item=Token{.value=9};}}`,
		"pointer-graph":              `struct Pointer{ptr u8 value;}fn void test(ref Pointer[] view){for(let ref Pointer item in view){}}`,
		"resource-graph":             `resource Owned{i64 value;cleanup{}}fn void test(ref Owned[] view){for(let ref Owned item in view){}}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;struct Token{i64 value;}" + body})
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("unadmitted struct iteration accepted")
			}
		})
	}
}
