package semantic_test

import "testing"

func TestNominalArrayFieldViewBoundary(t *testing.T) {
	base := `module main;struct T{i64 x;}struct Other{i64 x;}struct P{T[2] data;usize n;}fn void fill(ref mut T[] v){}fn void pair(ref mut T[] a,ref T[] b){}fn ref T[] relay(ref T[] v){return v;}fn Option<ref T[]> wrap(ref T[] v){return .some(v);}`
	init := `var P p=P{.data=[T{.x=1},T{.x=2}],.n=2};`
	for _, tc := range []struct{ name, source, code string }{
		{"immutable-element", `fn void bad(){let P p=P{.data=[T{.x=1};2],.n=2};p.data[0]=T{.x=3};}`, "E1008"},
		{"immutable-leaf", `fn void bad(ref P p){p.data[0].x=3;}`, "E1008"},
		{"immutable-view", `fn void bad(ref P p){fill(ref mut p.data);}`, "E1008"},
		{"nominal-store", `p.data[0]=Other{.x=3};`, "E1006"},
		{"nominal-view", `let ref Other[] v=ref p.data;`, "E1006"},
		{"nominal-for", `for(let Other x in p.data){}`, "E1006"},
		{"bound-type", `let ref T[] v=ref p.data[true..];`, "E1006"},
		{"local-escape", `fn ref T[] bad(){let P p=P{.data=[T{.x=1};2],.n=2};return ref p.data;}`, "E2102"},
		{"value-escape", `fn ref T[] bad(P p){return ref p.data[..];}`, "E2102"},
		{"element-escape", `fn ref T bad(P p){return ref p.data[0];}`, "E2102"},
		{"for-escape", `fn ref T bad(P p){for(let ref T t in p.data){return t;}return ref p.data[0];}`, "E2102"},
		{"shared-store", `let ref T[] v=ref p.data;p.data[0]=T{.x=3};assert(v[0].x==1);`, "E2101"},
		{"shared-leaf-store", `let ref T[] v=ref p.data;p.data[0].x=3;assert(v[0].x==1);`, "E2101"},
		{"shared-whole-store", `let ref T t=ref p.data[0];p.data=[T{.x=3};2];assert(t.x==1);`, "E2101"},
		{"exclusive-read", `let ref mut T[] v=ref mut p.data;assert(p.n==2);v[0].x=3;`, "E2101"},
		{"exclusive-write", `let ref mut T t=ref mut p.data[0];p.n=3;t.x=3;`, "E2101"},
		{"sibling", `let ref mut T[] a=ref mut p.data[..1];let ref mut T[] b=ref mut p.data[1..];a[0]=b[0];`, "E2101"},
		{"call-held", `pair(ref mut p.data[..1],ref p.data[1..]);`, "E2101"},
		{"bound-write", `let ref T[] v=ref p.data[..{p.n=3;1}];assert(v.len==1);`, "E2101"},
		{"read-index-write", `let T t=p.data[{p.n=3;0}];`, "E2101"},
		{"leaf-index-write", `p.data[{p.n=3;0}].x=3;`, "E2101"},
		{"element-index-write", `p.data[{p.n=3;0}]=T{.x=3};`, "E2101"},
		{"leaf-rhs-read", `p.data[0].x=p.data[1].x;`, "E2101"},
		{"element-rhs-read", `p.data[0]=p.data[1];`, "E2101"},
		{"leaf-rhs-write", `p.data[0].x={p.n=3;5};`, "E2101"},
		{"element-rhs-replace", `p.data[0]={p.data=[T{.x=3};2];T{.x=5}};`, "E2101"},
		{"returned-alias", `let ref T[] v=relay(ref p.data);p.n=3;assert(v[0].x==1);`, "E2101"},
		{"tagged-alias", `let Option<ref T[]> v=wrap(ref p.data);p.n=3;switch(v){case .some(a):{assert(a[0].x==1);}case .none:{}};`, "E2101"},
		{"parent-frozen", `let ref mut P root=ref mut p;let ref mut T[] v=ref mut root.data;assert(root.n==2);v[0].x=3;`, "E2101"},
		{"copied-for-write", `for(let T t in p.data){p.n=3;}`, "E2101"},
		{"shared-for-write", `for(let ref T t in p.data){p.data[0].x=3;}`, "E2101"},
		{"mutable-for-read", `for(let ref mut T t in p.data){assert(p.n==2);t.x=3;}`, "E2101"},
		{"loop-break", `for(let ref T t in p.data){p.n=3;break;}`, "E2101"},
		{"loop-return", `for(let ref mut T t in p.data){assert(p.n==2);return;}`, "E2101"},
		{"temporary", `fn P make(){return P{.data=[T{.x=1};2],.n=2};}fn void bad(){fill(ref mut make().data);}`, ""},
		{"indexed-owner", `var P[1] pages=[p];let ref T[] v=ref pages[0].data;`, ""},
		{"tagged-owner", `struct Q{T[2] data;Option<i32> flag;}fn void bad(ref Q q){let ref T[] v=ref q.data;}`, ""},
		{"nested-slice", `let ref T[] v=ref p.data[..][..];`, ""},
		{"ordinary-slice", `let ref T[] v=p.data[..];`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := tc.source
			if len(source) < 3 || source[:3] != "fn " && source[:7] != "struct " {
				source = `fn void bad(){` + init + source + `}`
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
