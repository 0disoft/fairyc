package semantic_test

import "testing"

func TestArrayFieldViewBoundary(t *testing.T) {
	base := `module main;struct P{u8[2] bytes;i64[2] totals;i32 other;}fn void fill(ref mut u8[] v){}fn void pair(ref mut u8[] a,ref u8[] b){}fn ref u8[] relay(ref u8[] v){return v;}fn Option<ref u8[]> wrap(ref u8[] v){return .some(v);}`
	for _, tc := range []struct{ name, source, code string }{
		{"immutable-store", `fn void bad(){let P p=P{.bytes=[1,2],.totals=[1,2],.other=0};p.bytes[0]=3;}`, "E1008"},
		{"immutable-view", `fn void bad(){let P p=P{.bytes=[1,2],.totals=[1,2],.other=0};fill(ref mut p.bytes[..]);}`, "E1008"},
		{"shared-parameter", `fn void bad(ref P p){p.bytes[0]=3;}`, "E1008"},
		{"shared-parameter-view", `fn void bad(ref P p){fill(ref mut p.bytes);}`, "E1008"},
		{"width", `fn void bad(ref P p){let ref i32[] v=ref p.totals[..];}`, "E1006"},
		{"bound-type", `fn void bad(ref P p){let ref u8[] v=ref p.bytes[true..];}`, "E1006"},
		{"local-escape", `fn ref u8[] bad(){let P p=P{.bytes=[1,2],.totals=[1,2],.other=0};return ref p.bytes[..];}`, "E2102"},
		{"value-parameter-escape", `fn ref u8[] bad(P p){return ref p.bytes;}`, "E2102"},
		{"element-escape", `fn ref u8 bad(P p){return ref p.bytes[0];}`, "E2102"},
		{"temporary", `fn P make(){return P{.bytes=[1,2],.totals=[1,2],.other=0};}fn void bad(){fill(ref mut make().bytes);}`, "E1008"},
		{"nested-slice", `fn void bad(ref P p){let ref u8[] v=ref p.bytes[..][..];}`, ""},
		{"ordinary-slice", `fn void bad(ref P p){let ref u8[] v=p.bytes[..];}`, ""},
		{"indexed-owner", `fn void bad(){var P[1] p=[P{.bytes=[1,2],.totals=[1,2],.other=0}];let ref u8[] v=ref p[0].bytes;}`, ""},
		{"tagged-owner", `struct Tagged{u8[2] bytes;Option<i32> value;}fn void bad(ref Tagged p){let ref u8[] v=ref p.bytes;}`, ""},
		{"shared-owner-write", `let ref u8[] v=ref p.bytes[..];p.other=9;assert(v[0]==1);`, "E2101"},
		{"exclusive-owner-read", `let ref mut u8[] v=ref mut p.bytes[..];assert(p.other==0);v[0]=9;`, "E2101"},
		{"exclusive-owner-write", `let ref mut u8[] v=ref mut p.bytes[..];p.other=9;v[0]=8;`, "E2101"},
		{"sibling", `let ref mut u8[] a=ref mut p.bytes[..1];let ref mut u8[] b=ref mut p.bytes[1..];a[0]=b[0];`, "E2101"},
		{"call-held", `pair(ref mut p.bytes[..1],ref p.bytes[1..]);`, "E2101"},
		{"shared-live-direct-store", `let ref u8[] v=ref p.bytes;p.bytes[0]=3;assert(v[0]==1);`, "E2101"},
		{"element-live-store", `let ref u8 v=ref p.bytes[1];p.bytes[0]=3;assert(*v==2);`, "E2101"},
		{"bound-write", `let ref u8[] v=ref p.bytes[..{p.other=9;1}];assert(v.len==1);`, "E2101"},
		{"index-write", `p.bytes[{p.other=9;0}]=3;`, "E2101"},
		{"read-index-write", `let u8 value=p.bytes[{p.other=9;0}];`, "E2101"},
		{"rhs-read", `p.bytes[0]=p.bytes[1];`, "E2101"},
		{"rhs-replace", `p.bytes[0]={p.bytes=[3,4];5};`, "E2101"},
		{"returned-alias", `let ref u8[] v=relay(ref p.bytes[..1]);p.bytes=[3,4];assert(v[0]==1);`, "E2101"},
		{"tagged-alias", `let Option<ref u8[]> v=wrap(ref p.bytes[..1]);p.other=9;switch(v){case .some(value):{assert(value[0]==1);}case .none:{}};`, "E2101"},
		{"borrowed-child-parent", `let ref mut P root=ref mut p;let ref mut u8[] v=ref mut root.bytes[..];assert(root.other==0);v[0]=9;`, "E2101"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := tc.source
			if tc.code == "E2101" {
				source = `fn void bad(){var P p=P{.bytes=[1,2],.totals=[1,2],.other=0};` + source + `}`
			}
			r := analyzeText(t, map[string]string{"main.fy": base + source})
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
			t.Fatalf("expected %s rejection: %+v %+v", tc.code, r.Diagnostics, r.Unsupported)
		})
	}
}
