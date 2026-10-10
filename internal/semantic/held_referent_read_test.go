package semantic_test

import (
	"strings"
	"testing"
)

func TestHeldExclusiveReferentReads(t *testing.T) {
	const scalar = `module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn i32 read(ref i32 p){return *p;}`
	const record = `module main;struct S{i32 x;}fn void take(ref mut S p,i32 n){}fn void share(ref S p,i32 n){}`
	const view = `module main;fn void take(ref mut u8[] p,u8 n){}fn void share(ref u8[] p,u8 n){}fn void size(ref mut u8[] p,usize n){}`
	for _, tc := range []struct {
		name, declarations, function string
		conflict                     bool
	}{
		{"scalar local", scalar, `fn void f(){var i32 x=1;let ref mut i32 p=ref mut x;take(p,*p);}`, true},
		{"scalar parameter", scalar, `fn void f(ref mut i32 p){take(p,*p);}`, true},
		{"scalar alias", scalar, `fn void f(ref mut i32 p){let ref mut i32 q=p;take(p,*q);}`, true},
		{"scalar groups", scalar, `fn void f(ref mut i32 p){take((p),*((p)));}`, true},
		{"scalar shared", scalar, `fn void f(ref mut i32 p){share(p,*p);}`, false},
		{"scalar earlier read", scalar + `fn void reversed(i32 n,ref mut i32 p){}`, `fn void f(ref mut i32 p){reversed(*p,p);}`, false},
		{"scalar released", scalar, `fn void f(ref mut i32 p){take(p,0);discard *p;}`, false},
		{"scalar selected store", scalar, `fn void f(ref mut i32 p){*p=*p+1;}`, false},
		{"scalar RHS shared call", scalar, `fn void f(ref mut i32 p){*p=read(p);}`, false},
		{"scalar unrelated", scalar, `fn void f(ref mut i32 p,ref i32 q){take(p,*q);}`, false},
		{"field local", record, `fn void f(){var S x=S{.x=1};let ref mut S p=ref mut x;take(p,p.x);}`, true},
		{"field parameter", record, `fn void f(ref mut S p){take(p,p.x);}`, true},
		{"field shared", record, `fn void f(ref mut S p){share(p,p.x);}`, false},
		{"field selected store", record, `fn void f(ref mut S p){p.x=p.x+1;}`, false},
		{"field named len", `module main;struct S{usize len;}fn void take(ref mut S p,usize n){}`, `fn void f(ref mut S p){take(p,p.len);}`, true},
		{"index local", view, `fn void f(){var u8[2] x=[1,2];let ref mut u8[] p=ref mut x;take(p,p[0]);}`, true},
		{"index parameter", view, `fn void f(ref mut u8[] p){take(p,p[0]);}`, true},
		{"index shared", view, `fn void f(ref mut u8[] p){share(p,p[0]);}`, false},
		{"index selected store", view, `fn void f(ref mut u8[] p){p[0]=p[1];}`, false},
		{"length metadata", view, `fn void f(ref mut u8[] p){size(p,p.len);}`, false},
		{"descriptor copy", scalar, `fn void f(ref mut i32 p){take(p,{let ref mut i32 q=p;0});}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := tc.declarations + tc.function
			result := analyzeText(t, map[string]string{"main.fy": text})
			if !tc.conflict {
				assertCleanTypeResult(t, result)
				return
			}
			if len(result.Unsupported) != 0 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2101" {
				t.Fatalf("diagnostics=%+v unsupported=%+v", result.Diagnostics, result.Unsupported)
			}
			if tc.name == "scalar parameter" {
				d := result.Diagnostics[0]
				read := strings.LastIndex(text, "*p")
				loan := strings.LastIndex(text, "take(p") + len("take(")
				if d.Primary.StartByte != read || d.Primary.EndByte != read+2 || len(d.Related) != 2 {
					t.Fatalf("read diagnostic=%+v", d)
				}
				for _, related := range d.Related {
					start := loan
					if related.Role == "last-use" {
						start = read + 1
					}
					if related.Span.StartByte != start || related.Span.EndByte != start+1 {
						t.Fatalf("%s=%+v", related.Role, related.Span)
					}
				}
			}
		})
	}
}
