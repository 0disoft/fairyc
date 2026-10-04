package semantic_test

import "testing"

func TestStructArrayFieldBorrowRejections(t *testing.T) {
	base := `module main;struct Pos{i64 x;i64 y;}struct Point{Pos pos;}
fn void edit(ref mut i64 x){*x=9;}fn void pair(ref mut i64 x,ref i64 y){}
fn ref i64 relay(ref i64 x){return x;}fn Option<ref i64> wrap(ref i64 x){return .some(x);}`
	for _, tc := range []struct{ name, source, code string }{
		{"immutable", `fn void bad(){let Point[1] a=[Point{.pos=Pos{.x=1,.y=2}}];edit(ref mut a[0].pos.x);}`, "E1008"},
		{"temporary-shared", `fn Point[1] make(){return [Point{.pos=Pos{.x=1,.y=2}}];}fn void bad(){let ref i64 p=ref make()[0].pos.x;}`, ""},
		{"temporary-exclusive", `fn Point[1] make(){return [Point{.pos=Pos{.x=1,.y=2}}];}fn void bad(){edit(ref mut make()[0].pos.x);}`, "E1008"},
		{"local-escape", `fn ref i64 bad(){var Point[1] a=[Point{.pos=Pos{.x=1,.y=2}}];return ref a[0].pos.x;}`, "E2102"},
		{"mutable-escape", `fn ref mut i64 bad(){var Point[1] a=[Point{.pos=Pos{.x=1,.y=2}}];return ref mut a[0].pos.x;}`, "E2102"},
		{"parameter-escape", `fn ref i64 bad(Point[1] a){return ref a[0].pos.x;}`, "E2102"},
		{"wrong-field-type", `fn void bad(Point[1] a){let ref i32 p=ref a[0].pos.x;}`, "E1006"},
		{"missing-field", `fn void bad(Point[1] a){let ref i64 p=ref a[0].missing;}`, ""},
		{"shared-owner-write", `let ref i64 p=ref a[0].pos.x;a[1].pos.y=9;assert(*p==1);`, "E2101"},
		{"exclusive-owner-read", `let ref mut i64 p=ref mut a[0].pos.x;assert(a[1].pos.y==2);*p=9;`, "E2101"},
		{"exclusive-owner-write", `let ref mut i64 p=ref mut a[0].pos.x;a[1].pos.y=9;*p=8;`, "E2101"},
		{"sibling-fields", `let ref mut i64 p=ref mut a[0].pos.x;let ref mut i64 q=ref mut a[0].pos.y;*p=*q;`, "E2101"},
		{"sibling-elements", `let ref mut i64 p=ref mut a[0].pos.x;let ref mut i64 q=ref mut a[1].pos.x;*p=*q;`, "E2101"},
		{"shared-exclusive", `let ref i64 p=ref a[0].pos.x;edit(ref mut a[1].pos.x);assert(*p==1);`, "E2101"},
		{"call-held", `pair(ref mut a[0].pos.x,ref a[1].pos.y);`, "E2101"},
		{"index-write", `edit(ref mut a[{a[1].pos.y=9;0}].pos.x);`, "E2101"},
		{"index-borrow", `edit(ref mut a[{edit(ref mut a[1].pos.y);0}].pos.x);`, "E2101"},
		{"grouped-shared", `let ref i64 p=ref ((a)[0]).pos.x;(a)[1].pos.y=9;assert(*p==1);`, "E2101"},
		{"array-view-alias", `let ref mut Point[] view=ref mut a;let ref mut i64 p=ref mut a[0].pos.x;assert(view.len==2);*p=9;`, "E2101"},
		{"returned-alias", `let ref i64 p=relay(ref a[0].pos.x);let ref i64 copy=p;a[1].pos.y=9;assert(*copy==1);`, "E2101"},
		{"tagged-alias", `let Option<ref i64> p=wrap(ref a[0].pos.x);a[1].pos.y=9;switch(p){case .some(v):{assert(*v==1);}case .none:{}};`, "E2101"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := tc.source
			if tc.code == "E2101" {
				source = `fn void bad(){var Point[2] a=[Point{.pos=Pos{.x=1,.y=2}};2];` + source + `}`
			}
			result := analyzeText(t, map[string]string{"main.fy": base + source})
			if tc.code == "" {
				if len(result.Diagnostics)+len(result.Unsupported) > 0 {
					return
				}
			} else {
				for _, d := range result.Diagnostics {
					if d.Code == tc.code {
						return
					}
				}
			}
			t.Fatalf("expected rejection %s: %+v %+v", tc.code, result.Diagnostics, result.Unsupported)
		})
	}
}
