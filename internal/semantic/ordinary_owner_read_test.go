package semantic_test

import (
	"strings"
	"testing"
)

func TestOrdinaryOwnerReadBorrowLifetime(t *testing.T) {
	for _, tc := range []struct {
		name, declarations, body, code string
	}{
		{"scalar live", "", "var i32 x=1;let ref mut i32 p=ref mut x;discard x;discard *p;", "E2101"},
		{"last use", "", "var i32 x=1;let ref mut i32 p=ref mut x;discard *p;discard x;", ""},
		{"shared read", "", "var i32 x=1;let ref i32 p=ref x;discard x;discard *p;", ""},
		{"own reference read", "", "var i32 x=1;let ref mut i32 p=ref mut x;discard *p;", ""},
		{"alias retained", "", "var i32 x=1;let ref mut i32 p=ref mut x;let ref mut i32 q=p;discard x;discard *q;", "E2101"},
		{"unrelated root", "", "var i32 x=1;var i32 y=2;let ref mut i32 p=ref mut x;discard y;discard *p;", ""},
		{"tagged loan", "", "var i32 x=1;let Option<ref mut i32> p=.some(ref mut x);discard x;switch(p){case .some(q):{discard *q;}case .none:{}};", "E2101"},
		{"call held", "fn void both(ref mut i32 p,i32 n){}", "var i32 x=1;both(ref mut x,x);", "E2101"},
		{"shared call held", "fn void both(ref i32 p,i32 n){}", "var i32 x=1;both(ref x,x);", ""},
		{"array index", "", "var u8[2] x=[1,2];let ref mut u8[] p=ref mut x;discard x[0];discard p[0];", "E2101"},
		{"struct field", "struct S{i32 n;}", "var S x=S{.n=1};let ref mut S p=ref mut x;discard x.n;discard p.n;", "E2101"},
		{"resource field", "resource R{i32 n;cleanup{}}", "var R x=R{.n=1};let ref mut R p=ref mut x;discard x.n;discard p.n;", "E2101"},
		{"move uses move diagnostic", "resource R{cleanup{}}fn void take(R r){}", "let R x=R{};let ref R p=ref x;take(x);discard p;", "E2105"},
		{"mutable move uses move diagnostic", "resource R{cleanup{}}fn void take(R r){}", "var R x=R{};let ref mut R p=ref mut x;take(x);discard p;", "E2105"},
		{"field store single diagnostic", "struct S{i32 n;}", "var S x=S{.n=1};let ref mut S p=ref mut x;x.n=2;discard p.n;", "E2101"},
		{"array store single diagnostic", "", "var u8[2] x=[1,2];let ref mut u8[] p=ref mut x;x[0]=3;discard p[0];", "E2101"},
		{"new borrow single diagnostic", "fn void read(ref i32 p){}", "var i32 x=1;let ref mut i32 p=ref mut x;read(ref x);discard *p;", "E2101"},
		{"store own reference", "", "var i32 x=1;let ref mut i32 p=ref mut x;*p=*p+1;", ""},
		{"new array range borrow single diagnostic", "", "var u8[2] x=[1,2];let ref mut u8[] p=ref mut x;let ref u8[] q=ref x[..1];discard p[0];discard q[0];", "E2101"},
		{"selector reads held root", "", "var u8[2] x=[1,2];let ref mut u8 item=ref mut x[{discard x[0];0}];discard *item;", "E2101"},
		{"array length metadata", "", "var u8[2] x=[1,2];let ref mut u8 item=ref mut x[x.len-1];discard *item;", ""},
		{"constructor owner metadata", "resource R{ptr mut void data;usize length;cleanup{}}", "var R x=R{.data=null,.length=0};let p=unsafe{byte_slice::from_raw_parts_mut(ref mut x,x.data,x.length)};discard p;", ""},
		{"constructor effectful read", "resource R{ptr mut void data;usize length;cleanup{}}", "var R x=R{.data=null,.length=0};let p=unsafe{byte_slice::from_raw_parts_mut(ref mut x,x.data,{discard x.length;0})};discard p;", "E2101"},
		{"raw cast is not a safe call loan", "resource R{usize length;cleanup{}}unsafe extern C fn u32 probe(ptr mut void data,usize length);", "var R x=R{.length=0};discard unsafe{probe(ref mut x as ptr mut void,x.length)};", ""},
		{"raw cast checks borrow creation", "resource R{usize length;cleanup{}}unsafe extern C fn u32 probe(ptr mut void data,usize length);", "var R x=R{.length=0};let ref R p=ref x;discard unsafe{probe(ref mut x as ptr mut void,x.length)};discard p;", "E2101"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "module main;" + tc.declarations + "fn void f(){" + tc.body + "}"
			result := analyzeText(t, map[string]string{"main.fy": source})
			if tc.code == "" {
				assertCleanTypeResult(t, result)
				return
			}
			if len(result.Unsupported) != 0 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != tc.code {
				t.Fatalf("diagnostics=%+v unsupported=%+v", result.Diagnostics, result.Unsupported)
			}
			d := result.Diagnostics[0]
			if tc.name == "scalar live" {
				read := strings.Index(source, "discard x") + len("discard ")
				borrow := strings.Index(source, "ref mut x")
				last := strings.LastIndex(source, "*p") + 1
				if d.Primary.StartByte != read || d.Primary.EndByte != read+1 {
					t.Fatalf("read span=%+v", d.Primary)
				}
				if len(d.Related) != 2 {
					t.Fatalf("related=%+v", d.Related)
				}
				for _, related := range d.Related {
					wantStart, wantEnd := borrow, borrow+len("ref mut x")
					if related.Role == "last-use" {
						wantStart, wantEnd = last, last+1
					}
					if related.Span.StartByte != wantStart || related.Span.EndByte != wantEnd {
						t.Fatalf("%s span=%+v want=%d..%d", related.Role, related.Span, wantStart, wantEnd)
					}
				}
			}
		})
	}
}
