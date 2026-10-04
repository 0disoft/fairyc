package semantic_test

import (
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

func TestSliceForFlow(t *testing.T) {
	for _, tc := range []struct{ name, body, code string }{
		{"copy-array", `for(let i64 item in data[1..2]){assert(item>0);}data[0]=9;`, ""},
		{"shared-array", `for(let ref i64 item in data[..]){assert(*item>0);}data[0]=9;`, ""},
		{"mut-array", `for(let ref mut i64 item in data[1..]){*item=9;}data[0]=9;`, ""},
		{"shared-view", `let ref i64[] view=ref data;for(let ref i64 item in view[..]){assert(*item==view[0]);}`, ""},
		{"mut-view", `let ref mut i64[] view=ref mut data;for(let ref mut i64 item in view[..]){*item=9;}view[0]=9;`, ""},
		{"copy-mut-view", `let ref mut i64[] view=ref mut data;for(let i64 item in view[..]){assert(item>0);}view[0]=9;`, ""},
		{"shared-mut-view", `let ref mut i64[] view=ref mut data;for(let ref i64 item in view[..]){assert(*item>0);}view[0]=9;`, ""},
		{"shadow", `for(let ref mut i64 data in data[..]){*data=9;}`, ""},
		{"break", `for(let ref mut i64 item in data[..]){*item=9;break;}data[0]=9;`, ""},
		{"continue", `for(let ref mut i64 item in data[..]){*item=9;continue;}data[0]=9;`, ""},
		{"return", `for(let ref mut i64 item in data[..]){*item=9;return;}`, ""},
		{"copy-owner-write", `for(let i64 item in data[..]){data[0]=item;break;}`, "E2101"},
		{"mut-owner-read", `for(let ref mut i64 item in data[..]){*item=9;assert(data[0]==9);}`, "E2101"},
		{"mut-parent-read", `let ref mut i64[] view=ref mut data;for(let ref mut i64 item in view[..]){*item=9;assert(view[0]==9);}`, "E2101"},
		{"copy-mut-parent-write", `let ref mut i64[] view=ref mut data;for(let i64 item in view[..]){view[0]=item;}`, "E2101"},
		{"shared-parent-write", `let ref mut i64[] view=ref mut data;for(let ref i64 item in view[..]){assert(*item>0);view[0]=9;return;}`, "E2101"},
		{"bound-write", `for(let i64 item in data[..{data[0]=9;1}]){}`, "E2101"},
		{"nested-mut", `for(let ref mut i64 a in data[..]){for(let ref mut i64 b in data[..]){*b=9;}*a=9;}`, "E2101"},
		{"escape", `var Option<ref mut i64> saved=.none;for(let ref mut i64 item in data[..]){switch(saved){case .some(old):{assert(*old>0);}case .none:{}};saved=.some(item);}switch(saved){case .some(old):{assert(*old>0);}case .none:{}};`, "E2101"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;fn void run(){var i64[2] data=[1,2];" + tc.body + "}"})
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
func TestSliceForBoundary(t *testing.T) {
	for name, src := range map[string]string{
		"immutable-array":    `fn void bad(){let i64[1] data=[1];for(let ref mut i64 x in data[..]){}}`,
		"shared-mutable":     `fn void bad(ref i64[] data){for(let ref mut i64 x in data[..]){}}`,
		"mutable-descriptor": `fn void bad(ref i64[] data){var ref i64[] alias=data;for(let i64 x in alias[..]){}}`,
		"width":              `fn void bad(ref i32[] data){for(let i64 x in data[..]){}}`,
		"nested-slice":       `fn void bad(ref i64[] data){for(let i64 x in data[..][..]){}}`,
		"call-root":          `fn ref i64[] relay(ref i64[] data){return data;}fn void bad(ref i64[] data){for(let i64 x in relay(data)[..]){}}`,
		"literal-root":       `fn void bad(){for(let i64 x in [1,2][..]){}}`,
		"bad-bound":          `fn void bad(ref i64[] data){for(let i64 x in data[true..]){}}`,
		"local-escape":       `fn ref i64 bad(){let i64[1] data=[1];for(let ref i64 x in data[..]){return x;}return ref data[0];}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + src})
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("invalid slice loop accepted")
			}
			if strings.Contains(name, "escape") {
				for _, d := range r.Diagnostics {
					if d.Code == "E2102" {
						return
					}
				}
				t.Fatalf("missing escape diagnostic: %+v %+v", r.Diagnostics, r.Unsupported)
			}
		})
	}
}
