package semantic_test

import (
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

func TestArrayElementBorrowFlow(t *testing.T) {
	for _, typ := range []string{"u8", "i32", "i64", "u32", "usize"} {
		for _, tc := range []struct{ name, body, code string }{
			{"stored", "let ref mut T item=ref mut data[0];*item=*item+1;assert(data[0]==2);", ""},
			{"shared", "let ref T a=ref data[0];let ref T b=ref data[1];assert(*a==*b && data[0]==1);data[0]=9;", ""},
			{"grouped", "let ref mut T item=ref mut ((data)[0]);*item=9;", ""},
			{"call", "edit(ref mut data[0]);read(ref data[0]);", ""},
			{"index-length", "let ref mut T item=ref mut data[data.len-1];*item=9;", ""},
			{"once", "var usize calls=0;let ref mut T item=ref mut data[{calls+=1;0}];*item=9;assert(calls==1);", ""},
			{"parent-read", "let ref mut T item=ref mut data[0];assert(data.len>0);*item=9;", "E2101"},
			{"parent-write", "let ref mut T item=ref mut data[0];data[1]=9;assert(*item==1);", "E2101"},
			{"shared-write", "let ref T item=ref data[0];data[1]=9;assert(*item==1);", "E2101"},
			{"siblings", "let ref mut T a=ref mut data[0];let ref mut T b=ref mut data[1];assert(*a==*b);", "E2101"},
			{"earlier-argument", "pair(ref mut data[0],ref data[1]);", "E2101"},
			{"index-store", "let ref mut T item=ref mut data[{data[0]=9;0}];*item=1;", "E2101"},
			{"index-borrow", "let ref mut T item=ref mut data[{edit(ref mut data[0]);0}];*item=1;", "E2101"},
			{"view-live", "let ref mut T[] alias=ref mut data;let ref mut T item=ref mut data[0];assert(alias.len>0);*item=9;", "E2101"},
			{"returned-live", "let ref mut T item=identity(ref mut data[0]);data[1]=9;assert(*item==1);", "E2101"},
			{"tagged-live", "let Option<ref mut T> item=.some(ref mut data[0]);data[1]=9;switch(item){case .some(value):{assert(*value==1);}case .none:{}};", "E2101"},
			{"copied-alias", "let ref mut T a=ref mut data[0];let ref mut T b=a;data[1]=9;assert(*b==1);", "E2101"},
			{"rhs-owner-read", "let ref mut T item=ref mut data[0];*item=data[1];", "E2101"},
		} {
			t.Run(typ+"/"+tc.name, func(t *testing.T) {
				src := "module main;fn void edit(ref mut T item){*item=9;}fn void read(ref T item){}fn void pair(ref mut T a,ref T b){}fn ref mut T identity(ref mut T value){return value;}fn void run(){var T[2] data=[1,1];" + tc.body + "}"
				r := analyzeText(t, map[string]string{"main.fy": strings.ReplaceAll(src, "T", typ)})
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
}
func TestArrayElementBorrowBoundary(t *testing.T) {
	for name, src := range map[string]string{
		"immutable-mut":    "fn void bad(){let i64[2] data=[1,2];let ref mut i64 item=ref mut data[0];}",
		"wrong-width":      "fn void bad(){var i32[2] data=[1,2];let ref i64 item=ref data[0];}",
		"bool":             "fn void bad(){var bool[2] data=[true,false];let ref mut bool item=ref mut data[0];}",
		"call-root":        "fn i64[2] make(){return [1,2];}fn void bad(){let ref i64 item=ref make()[0];}",
		"parameter-escape": "fn ref i64 bad(i64[2] data){return ref data[0];}",
		"literal":          "fn void bad(){let ref i64 item=ref [1,2][0];}",
		"shared-store":     "fn void bad(){var i64[2] data=[1,2];let ref i64 item=ref data[0];*item=9;}",
		"local-escape":     "fn ref i64 bad(){var i64[2] data=[1,2];return ref data[0];}",
		"mutable-escape":   "fn ref mut i64 bad(){var i64[2] data=[1,2];return ref mut data[0];}",
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + src})
			if strings.Contains(name, "escape") {
				for _, d := range r.Diagnostics {
					if d.Code == "E2102" {
						return
					}
				}
				t.Fatalf("missing E2102: %+v %+v", r.Diagnostics, r.Unsupported)
			}
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("unadmitted array reference accepted")
			}
		})
	}
}
