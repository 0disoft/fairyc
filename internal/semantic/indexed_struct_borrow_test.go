package semantic_test

import (
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"testing"
)

func TestIndexedStructBorrowFlow(t *testing.T) {
	for _, tc := range []struct{ name, body, code string }{
		{"mutable-item", "let ref mut Token item=ref mut view[0];item.value=4000000000;assert(view[0].value==4000000000);", ""},
		{"shared-item", "let ref Token item=ref view[0];let ref Token other=ref view[1];assert(item.value==other.value);view[0].value=9;", ""},
		{"grouped", "let ref mut Token item=ref mut ((view)[0]);item.position.start+=1;", ""},
		{"direct-argument", "edit(ref mut view[0]);read(ref view[0]);", ""},
		{"copied-value", "let ref mut Token item=ref mut view[0];var Token copy=*item;copy.value=9;assert(item.value!=9);", ""},
		{"parent-read", "let ref mut Token item=ref mut view[0];assert(view.len>0);item.value=9;", "E2101"},
		{"parent-write", "let ref mut Token item=ref mut view[0];view[1].value=1;assert(item.value>0);", "E2101"},
		{"alias-read", "let ref mut Token[] alias=view;let ref mut Token item=ref mut view[0];assert(alias.len>0);item.value=9;", "E2101"},
		{"exclusive-siblings", "let ref mut Token one=ref mut view[0];let ref mut Token two=ref mut view[1];assert(one.value==two.value);", "E2101"},
		{"shared-write", "let ref Token item=ref view[0];view[1].value=1;assert(item.value>0);", "E2101"},
		{"shared-exclusive", "let ref Token item=ref view[0];let ref mut Token other=ref mut view[1];assert(item.value==other.value);", "E2101"},
		{"field-child", "let ref mut Token item=ref mut view[0];let ref mut usize start=ref mut item.position.start;item.value=9;assert(*start>0);", "E2101"},
		{"later-call-argument", "pair(ref mut view[0],ref view[1]);", "E2101"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := `module main;struct Span{usize start;usize end;}struct Token{i64 value;Span position;}
 fn void edit(ref mut Token item){item.value+=1;}fn void read(ref Token item){}fn void pair(ref mut Token a,ref Token b){}
 fn void run(ref mut Token[] view){` + tc.body + `}`
			r := analyzeText(t, map[string]string{"main.fy": text})
			if tc.code != "" {
				for _, d := range r.Diagnostics {
					if d.Code == tc.code {
						return
					}
				}
				t.Fatalf("missing %s: %+v %+v", tc.code, r.Diagnostics, r.Unsupported)
			}
			if len(r.Diagnostics) != 0 || len(r.Unsupported) != 0 || r.CompletedThrough != semantic.PhaseFlow {
				t.Fatalf("analysis: %+v %+v", r.Diagnostics, r.Unsupported)
			}
			p, e := mir.Lower(r.HIR)
			if e != nil {
				t.Fatal(e)
			}
			if errs := mir.Verify(p); len(errs) != 0 {
				t.Fatal(errs)
			}
		})
	}
}

func TestIndexedStructBorrowSourceBoundary(t *testing.T) {
	for name, body := range map[string]string{
		"shared-exclusive":   "fn void bad(ref Token[] view){let ref mut Token item=ref mut view[0];}",
		"shared-field-store": "fn void bad(ref Token[] view){let ref Token item=ref view[0];item.value=1;}",
		"temporary-shared":   "fn ref Token[] relay(ref Token[] view){return view;}fn void bad(ref Token[] view){let ref Token item=ref relay(view)[0];}",
		"temporary-mutable":  "fn ref mut Token[] relay(ref mut Token[] view){return view;}fn void bad(ref mut Token[] view){let ref mut Token item=ref mut relay(view)[0];}",
		"wrong-nominal":      "fn void bad(ref mut Token[] view){let ref mut Other item=ref mut view[0];}",
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;struct Token{i64 value;}struct Other{i64 value;}" + body})
			if len(r.Diagnostics) == 0 && len(r.Unsupported) == 0 {
				t.Fatal("unadmitted source accepted")
			}
		})
	}
}
