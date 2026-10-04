package semantic_test

import (
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

func TestMutableIntegerForFlow(t *testing.T) {
	for _, typ := range []string{"u8", "i32", "i64", "u32", "usize"} {
		for _, tc := range []struct{ name, body, code string }{
			{"stored", "for(let ref mut T item in view){*item=*item+1;}", ""},
			{"after-last-use", "for(let ref mut T item in view){*item=9;assert(view[0]==9);}", ""},
			{"parent-live", "for(let ref mut T item in view){assert(view.len>0);*item=9;}", "E2101"},
			{"alias-live", "let ref mut T[] alias=view;for(let ref mut T item in view){alias[0]=9;*item=1;}", "E2101"},
			{"before-break", "for(let ref mut T item in view){view[0]=9;assert(*item>0);break;}", "E2101"},
			{"before-return", "for(let ref mut T item in view){view[0]=9;assert(*item>0);return;}", "E2101"},
			{"before-continue", "for(let ref mut T item in view){view[0]=9;assert(*item>0);continue;}", "E2101"},
			{"nested-same", "for(let ref mut T outer in view){for(let ref mut T inner in view){*inner=9;}*outer=1;}", "E2101"},
			{"after-loop", "for(let ref mut T item in view){*item=9;break;}view[0]=1;", ""},
			{"scope", "for(let ref mut T item in view){}*item=1;", "E0101"},
		} {
			t.Run(typ+"/"+tc.name, func(t *testing.T) {
				body := strings.ReplaceAll(tc.body, "T", typ)
				r := analyzeText(t, map[string]string{"main.fy": "module main;fn void edit(ref mut " + typ + "[] view){" + body + "}"})
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
				if d := mir.Verify(p); len(d) != 0 {
					t.Fatal(d)
				}
			})
		}
	}
}

func TestMutableIntegerForSourceBoundary(t *testing.T) {
	for _, typ := range []string{"u8", "i32", "i64", "u32", "usize"} {
		for name, body := range map[string]string{
			"shared-source":      "fn void edit(ref T[] view){for(let ref mut T item in view){}}",
			"shared-item":        "fn void edit(ref mut T[] view){for(let ref T item in view){}}",
			"mutable-descriptor": "fn void edit(ref mut T[] view){var ref mut T[] alias=view;for(let ref mut T item in alias){}}",
			"temporary":          "fn void edit(ref mut T[] view){for(let ref mut T item in view[..]){}}",
			"integer-mismatch":   "fn void edit(ref mut T[] view){for(let ref mut Other item in view){}}",
			"local-escape":       "fn ref mut T edit(){var T[1] data=[1];let ref mut T[] view=ref mut data;for(let ref mut T item in view){return item;}return ref mut view[0];}",
			"wrong-width":        "fn void edit(ref mut T[] view){for(let ref mut bool item in view){}}",
			"immutable-array":    "fn void edit(){let T[1] data=[1];for(let ref mut T item in data){}}",
		} {
			t.Run(typ+"/"+name, func(t *testing.T) {
				other := "i64"
				if typ == "i64" {
					other = "u32"
				}
				r := analyzeText(t, map[string]string{"main.fy": "module main;" + strings.ReplaceAll(strings.ReplaceAll(body, "T", typ), "Other", other)})
				if name == "local-escape" {
					for _, d := range r.Diagnostics {
						if d.Code == "E2102" {
							return
						}
					}
					t.Fatalf("missing local-owner escape diagnostic: %+v %+v", r.Diagnostics, r.Unsupported)
				}
				if len(r.Diagnostics) == 0 && len(r.Unsupported) == 0 {
					t.Fatal("unadmitted source accepted")
				}
			})
		}
	}
}
