package semantic_test

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

func TestIntegerElementBorrowFlow(t *testing.T) {
	for _, typ := range []string{"u8", "i32", "i64", "u32", "usize"} {
		for _, tc := range []struct{ name, body, code string }{
			{"stored", `let ref mut T item=ref mut view[0];*item=*item+1;assert(view[0]==2);`, ""},
			{"shared", `let ref T a=ref view[0];let ref T b=ref view[1];assert(*a==*b);view[0]=9;`, ""},
			{"grouped", `let ref mut T item=ref mut ((view)[0]);*item=9;`, ""},
			{"call", `edit(ref mut view[0]);read(ref view[0]);`, ""},
			{"parent-read", `let ref mut T item=ref mut view[0];assert(view.len>0);*item=9;`, "E2101"},
			{"parent-write", `let ref mut T item=ref mut view[0];view[1]=1;assert(*item==1);`, "E2101"},
			{"shared-write", `let ref T item=ref view[0];view[1]=1;assert(*item==1);`, "E2101"},
			{"siblings", `let ref mut T a=ref mut view[0];let ref mut T b=ref mut view[1];assert(*a==*b);`, "E2101"},
			{"alias", `let ref mut T[] alias=view;let ref mut T item=ref mut view[0];assert(alias.len>0);*item=9;`, "E2101"},
			{"later-call", `pair(ref mut view[0],ref view[1]);`, "E2101"},
			{"index-effect", `let ref mut T item=ref mut view[{view[0]=9;0}];*item=1;`, "E2101"},
		} {
			t.Run(typ+"/"+tc.name, func(t *testing.T) {
				src := fmt.Sprintf(`module main;fn void edit(ref mut %[1]s item){*item=9;}fn void read(ref %[1]s item){}fn void pair(ref mut %[1]s a,ref %[1]s b){}fn void run(ref mut %[1]s[] view){%[2]s}`, typ, strings.ReplaceAll(tc.body, "T", typ))
				r := analyzeText(t, map[string]string{"main.fy": src})
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
func TestIntegerElementBorrowBoundary(t *testing.T) {
	for name, src := range map[string]string{
		"shared-exclusive": `fn void bad(ref i64[] view){let ref mut i64 item=ref mut view[0];}`,
		"shared-store":     `fn void bad(ref i64[] view){let ref i64 item=ref view[0];*item=1;}`,
		"wrong-width":      `fn void bad(ref i32[] view){let ref i64 item=ref view[0];}`,
		"temporary":        `fn void bad(ref mut i64[] view){let ref mut i64 item=ref mut view[..][0];}`,
		"fixed-array-root": `fn void bad(){var i64[2] data=[1,2];let ref mut i64 item=ref mut data[0];}`,
		"local-escape":     `fn ref i64 bad(){var i64[2] data=[1,2];let ref i64[] view=ref data;return ref view[0];}`,
		"mutable-escape":   `fn ref mut i64 bad(){var i64[2] data=[1,2];let ref mut i64[] view=ref mut data;return ref mut view[0];}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + src})
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("unadmitted reference accepted")
			}
		})
	}
}
