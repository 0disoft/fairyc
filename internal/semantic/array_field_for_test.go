package semantic_test

import (
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"testing"
)

func TestArrayFieldForFlow(t *testing.T) {
	base := `module main;struct P{u8[2] bytes;i32 other;}fn void run(){var P p=P{.bytes=[1,2],.other=0};`
	for _, tc := range []struct{ name, body, code string }{
		{"copy", `for(let u8 x in p.bytes){assert(x>0);}p.bytes[0]=9;`, ""},
		{"shared", `for(let ref u8 x in p.bytes[..]){assert(*x==p.bytes[0]||*x==p.bytes[1]);}p.other=9;`, ""},
		{"mutable", `for(let ref mut u8 x in p.bytes){*x+=1;}p.other=9;`, ""},
		{"break", `for(let ref mut u8 x in p.bytes[..]){*x=9;break;}p.other=9;`, ""},
		{"continue", `for(let ref mut u8 x in p.bytes){*x=9;continue;}p.other=9;`, ""},
		{"return", `for(let ref mut u8 x in p.bytes){*x=9;return;}`, ""},
		{"shared-owner-write", `for(let u8 x in p.bytes){p.other=9;break;}`, "E2101"},
		{"shared-field-write", `for(let ref u8 x in p.bytes){p.bytes[0]=9;return;}`, "E2101"},
		{"mutable-owner-read", `for(let ref mut u8 x in p.bytes){*x=9;assert(p.other==0);}`, "E2101"},
		{"mutable-owner-write", `for(let ref mut u8 x in p.bytes){*x=9;p.other=9;}`, "E2101"},
		{"bound-write", `for(let u8 x in p.bytes[..{p.other=9;1}]){}`, "E2101"},
		{"nested-mut", `for(let ref mut u8 x in p.bytes[..1]){for(let ref mut u8 y in p.bytes[1..]){*y=9;}*x=9;}`, "E2101"},
		{"prior-alias", `let ref u8[] a=ref p.bytes;for(let ref mut u8 x in p.bytes){*x=9;}assert(a[0]==1);`, "E2101"},
		{"retained-item", `var Option<ref mut u8> saved=.none;for(let ref mut u8 x in p.bytes){switch(saved){case .some(y):{assert(*y>0);}case .none:{}};saved=.some(x);}switch(saved){case .some(y):{assert(*y>0);}case .none:{}};`, "E2101"},
		{"parent-read", `let ref mut P root=ref mut p;for(let ref mut u8 x in root.bytes){*x=9;assert(root.other==0);}`, "E2101"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": base + tc.body + `}`})
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
			p, e := mir.Lower(r.HIR)
			if e != nil {
				t.Fatal(e)
			}
			if d := mir.Verify(p); len(d) != 0 {
				t.Fatal(d)
			}
		})
	}
}

func TestArrayFieldForBoundary(t *testing.T) {
	base := `module main;struct P{u8[2] bytes;i32 other;}`
	for _, tc := range []struct{ name, source, code string }{
		{"immutable", `fn void bad(){let P p=P{.bytes=[1,2],.other=0};for(let ref mut u8 x in p.bytes){}}`, "E1008"},
		{"shared-param", `fn void bad(ref P p){for(let ref mut u8 x in p.bytes[..]){}}`, "E1008"},
		{"width", `fn void bad(ref P p){for(let i64 x in p.bytes){}}`, "E1006"},
		{"bound", `fn void bad(ref P p){for(let u8 x in p.bytes[true..]){}}`, "E1006"},
		{"escape-local", `fn ref u8 bad(){let P p=P{.bytes=[1,2],.other=0};for(let ref u8 x in p.bytes){return x;}return ref p.bytes[0];}`, "E2102"},
		{"escape-value", `fn ref u8 bad(P p){for(let ref u8 x in p.bytes[..]){return x;}return ref p.bytes[0];}`, "E2102"},
		{"temporary", `fn P make(){return P{.bytes=[1,2],.other=0};}fn void bad(){for(let u8 x in make().bytes){}}`, ""},
		{"indexed", `fn void bad(){let P[1] ps=[P{.bytes=[1,2],.other=0}];for(let u8 x in ps[0].bytes){}}`, ""},
		{"nested-slice", `fn void bad(ref P p){for(let u8 x in p.bytes[..][..]){}}`, ""},
		{"tagged", `struct Tagged{u8[2] bytes;Option<i32> value;}fn void bad(ref Tagged p){for(let u8 x in p.bytes){}}`, ""},
		{"scalar-field", `fn void bad(ref P p){for(let i32 x in p.other){}}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": base + tc.source})
			if tc.code == "" {
				if len(r.Diagnostics)+len(r.Unsupported) > 0 {
					return
				}
			} else {
				for _, d := range r.Diagnostics {
					if d.Code == tc.code {
						return
					}
				}
			}
			t.Fatalf("expected %s: %+v %+v", tc.code, r.Diagnostics, r.Unsupported)
		})
	}
}
