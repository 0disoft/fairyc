package semantic_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestArrayForFlow(t *testing.T) {
	for _, typ := range []string{"u8", "i32", "i64", "u32", "usize"} {
		for _, tc := range []struct{ name, body, code string }{
			{"copy", "for(let T item in data){assert(item>0);}data[0]=9;", ""},
			{"mutable", "for(let ref mut T item in data){*item=*item+1;}data[0]=9;", ""},
			{"shadow", "for(let ref mut T data in data){*data=9;}", ""},
			{"break", "for(let ref mut T item in data){*item=9;break;}data[0]=1;", ""},
			{"continue", "for(let ref mut T item in data){*item=9;continue;}data[0]=1;", ""},
			{"independent-nested", "var T[1] other=[2];for(let ref mut T item in data){for(let ref mut T nested in other){*nested=1;}*item=9;}", ""},
			{"copy-nested-shared", "for(let T a in data){for(let T b in data){assert(a==b);}}", ""},
			{"copy-owner-write", "for(let T item in data){data[0]=item;}", "E2101"},
			{"copy-before-break", "for(let T item in data){data[0]=item;break;}", "E2101"},
			{"mut-owner-write", "for(let ref mut T item in data){*item=9;data[0]=1;}", "E2101"},
			{"mut-owner-read", "for(let ref mut T item in data){assert(data[0]==1);*item=9;}", "E2101"},
			{"mut-owner-read-after-item", "for(let ref mut T item in data){*item=9;assert(data[0]==9);}", "E2101"},
			{"mut-before-return", "for(let ref mut T item in data){*item=9;data[0]=1;return;}", "E2101"},
			{"mut-before-continue", "for(let ref mut T item in data){*item=9;data[0]=1;continue;}", "E2101"},
			{"replacement", "var T[2] other=[2,2];for(let ref mut T item in data){data=other;}", "E2101"},
			{"nested-same", "for(let ref mut T a in data){for(let ref mut T b in data){*b=9;}*a=1;}", "E2101"},
			{"alias", "let ref T[] alias=ref data;for(let ref mut T item in data){*item=9;assert(alias.len>0);}", "E2101"},
			{"scope", "for(let T item in data){}assert(item==1);", "E0101"},
		} {
			t.Run(typ+"/"+tc.name, func(t *testing.T) {
				source := strings.ReplaceAll("module main;fn void run(){var T[2] data=[1,1];"+tc.body+"}", "T", typ)
				r := analyzeText(t, map[string]string{"main.fy": source})
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

func TestArrayForBoundary(t *testing.T) {
	for name, source := range map[string]string{
		"immutable-mut":     "fn void bad(){let i64[1] data=[1];for(let ref mut i64 item in data){}}",
		"wrong-width":       "fn void bad(){var i32[1] data=[1];for(let i64 item in data){}}",
		"literal":           "fn void bad(){for(let i64 item in [1,2]){}}",
		"call":              "fn i64[1] make(){return [1];}fn void bad(){for(let i64 item in make()){}}",
		"shared-item-store": "fn void bad(){var i64[1] data=[1];for(let ref i64 item in data){*item=9;}}",
		"field":             "struct Box{i64 value;}fn void bad(){let Box data=Box{.value=1};for(let i64 item in data.value){}}",
		"local-escape":      "fn ref mut i64 bad(){var i64[1] data=[1];for(let ref mut i64 item in data){return item;}return ref mut data[0];}",
		"parameter-escape":  "fn ref i64 bad(i64[1] data){for(let i64 item in data){return ref data[0];}return ref data[0];}",
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + source})
			if strings.Contains(name, "escape") {
				for _, d := range r.Diagnostics {
					if d.Code == "E2102" {
						return
					}
				}
				t.Fatalf("missing E2102: %+v %+v", r.Diagnostics, r.Unsupported)
			}
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("unadmitted array loop accepted")
			}
		})
	}
}
