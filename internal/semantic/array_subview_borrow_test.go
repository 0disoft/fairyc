package semantic_test

import "testing"

func TestArraySubviewBorrowRejections(t *testing.T) {
	base := `module main;struct Point{i64 x;}fn void pair(ref mut i64[] a,ref i64[] b){}fn void edit(ref mut i64[] a){}fn ref i64[] relay(ref i64[] a){return a;}fn Option<ref i64[]> wrap(ref i64[] a){return .some(a);}`
	for _, tc := range []struct{ name, source, code string }{
		{"immutable", `fn void bad(){let i64[2] a=[1,2];edit(ref mut a[..]);}`, "E1008"},
		{"local-escape", `fn ref i64[] bad(){let i64[2] a=[1,2];return ref a[1..];}`, "E2102"},
		{"mutable-escape", `fn ref mut i64[] bad(){var i64[2] a=[1,2];return ref mut a[..1];}`, "E2102"},
		{"parameter-escape", `fn ref i64[] bad(i64[2] a){return ref a[..];}`, "E2102"},
		{"width", `fn void bad(){let i64[2] a=[1,2];let ref i32[] v=ref a[..];}`, "E1006"},
		{"bound-type", `fn void bad(){let i64[2] a=[1,2];let ref i64[] v=ref a[true..];}`, "E1006"},
		{"temporary", `fn i64[2] make(){return [1,2];}fn void bad(){let ref i64[] v=ref make()[..];}`, ""},
		{"temporary-mut", `fn i64[2] make(){return [1,2];}fn void bad(){edit(ref mut make()[..]);}`, ""},
		{"literal", `fn void bad(){let ref i64[] v=ref [1,2][..];}`, ""},
		{"nested", `fn void bad(){let i64[2] a=[1,2];let ref i64[] v=ref a[..][..];}`, ""},
		{"ordinary-slice", `fn void bad(){let i64[2] a=[1,2];let ref i64[] v=a[..];}`, ""},
		{"field", `struct Box{ i64[2] data; }fn void bad(Box b){let ref i64[] v=ref b.data[..];}`, ""},
		{"shared-write", `let ref i64[] v=ref a[..1];a[1]=9;assert(v[0]==1);`, "E2101"},
		{"exclusive-read", `let ref mut i64[] v=ref mut a[..1];assert(a[1]==2);v[0]=9;`, "E2101"},
		{"exclusive-write", `let ref mut i64[] v=ref mut a[..1];a[1]=9;v[0]=8;`, "E2101"},
		{"sibling", `let ref mut i64[] v=ref mut a[..1];let ref mut i64[] w=ref mut a[1..];v[0]=w[0];`, "E2101"},
		{"call-held", `pair(ref mut a[..1],ref a[1..]);`, "E2101"},
		{"bound-write", `let ref i64[] v=ref a[..{a[1]=9;1}];assert(v.len==1);`, "E2101"},
		{"bound-mut-write", `edit(ref mut a[{a[1]=9;0}..1]);`, "E2101"},
		{"bound-borrow", `edit(ref mut a[..{edit(ref mut a[1..]);1}]);`, "E2101"},
		{"returned-alias", `let ref i64[] v=relay(ref a[..1]);let ref i64[] w=v;a[1]=9;assert(w[0]==1);`, "E2101"},
		{"tagged-alias", `let Option<ref i64[]> v=wrap(ref a[..1]);a[1]=9;switch(v){case .some(w):{assert(w[0]==1);}case .none:{}};`, "E2101"},
		{"element-alias", `let ref i64[] v=ref a[..1];let ref mut i64 p=ref mut a[1];assert(v[0]==1);*p=9;`, "E2101"},
		{"grouped-owner", `let ref i64[] v=ref ((a)[..1]);(a)[1]=9;assert(v[0]==1);`, "E2101"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := tc.source
			if tc.code == "E2101" {
				source = `fn void bad(){var i64[2] a=[1,2];` + source + `}`
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
