package semantic_test

import (
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"testing"
)

func TestStructDerefStoreFlow(t *testing.T) {
	for _, tc := range []struct{ name, body, code string }{
		{"named", `let ref mut Token item=ref mut view[0];*(item)=Token{.value=4000000000,.span=Span{.start=2}};assert(view[0].value==4000000000);`, ""},
		{"iteration", `for(let ref mut Token item in view){*item=Token{.value=9,.span=Span{.start=2}};}`, ""},
		{"self-copy", `let ref mut Token item=ref mut view[0];*item=*item;`, ""},
		{"child-last-use", `let ref mut Token item=ref mut view[0];let ref usize child=ref item.span.start;assert(*child==1);*item=Token{.value=9,.span=Span{.start=2}};`, ""},
		{"live-child", `let ref mut Token item=ref mut view[0];let ref mut usize child=ref mut item.span.start;*item=Token{.value=9,.span=Span{.start=2}};assert(*child==1);`, "E2101"},
		{"live-shared", `let ref mut Token item=ref mut view[0];let ref Span shared=ref item.span;*item=Token{.value=9,.span=Span{.start=2}};assert(shared.start==1);`, "E2101"},
		{"rhs-parent", `let ref mut Token item=ref mut view[0];*item={view[0].value=1;Token{.value=9,.span=Span{.start=2}}};`, "E2101"},
		{"rhs-shared-read", `let ref mut Token item=ref mut view[0];*item={let ref Span shared=ref item.span;assert(shared.start==1);Token{.value=9,.span=Span{.start=2}}};`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": `module main;struct Span{usize start;}struct Token{i64 value;Span span;}fn void edit(ref mut Token[] view){` + tc.body + `}`})
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
func TestStructDerefStoreBoundary(t *testing.T) {
	for name, src := range map[string]string{
		"shared":        `struct T{i64 value;}fn void edit(ref T item){*item=T{.value=1};}`,
		"nominal":       `struct T{i64 value;}struct U{i64 value;}fn void edit(ref mut T item){*item=U{.value=1};}`,
		"pointer-field": `struct T{ptr void value;}fn void edit(ref mut T item){*item=T{.value=null};}`,
		"resource":      `resource T{i64 value;cleanup{}}fn void edit(ref mut T item){*item=T{.value=1};}`,
		"empty":         `struct T{}fn void edit(ref mut T item){*item=T{};}`,
		"temporary":     `struct T{i64 value;}fn ref mut T relay(ref mut T item){return item;}fn void edit(ref mut T item){*relay(item)=T{.value=1};}`,
		"compound":      `struct T{i64 value;}fn void edit(ref mut T item){*item+=T{.value=1};}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + src})
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("closed replacement admitted")
			}
		})
	}
}
