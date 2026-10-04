package semantic_test

import (
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

func TestDerefCompoundFlow(t *testing.T) {
	for _, typ := range []string{"u8", "i32", "i64", "u32", "usize"} {
		for _, op := range []string{"+=", "-=", "*=", "/=", "%=", "<<=", ">>=", "&=", "^=", "|="} {
			t.Run(typ+op, func(t *testing.T) {
				src := `module main;fn void edit(ref mut T value){*(value) OP 1;}fn void run(){var T[2] data=[12,12];for(let ref mut T item in data[1..]){edit(item);}}`
				src = strings.ReplaceAll(strings.ReplaceAll(src, "T", typ), "OP", op)
				r := analyzeText(t, map[string]string{"main.fy": src})
				if len(r.Diagnostics)+len(r.Unsupported) != 0 || r.CompletedThrough != semantic.PhaseFlow {
					t.Fatalf("analysis %+v %+v", r.Diagnostics, r.Unsupported)
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
	for _, tc := range []struct{ name, body, code string }{
		{"owner-read", `let ref mut i64 item=ref mut x;*item+=x;`, "E2101"},
		{"owner-write", `let ref mut i64 item=ref mut x;*item+={x=9;1};`, "E2101"},
		{"sibling", `let ref mut i64 item=ref mut x;let ref i64 sibling=ref x;*item+=1;assert(*sibling>0);`, "E2101"},
		{"shared", `let ref i64 item=ref x;*item+=1;`, "E1008"},
		{"width", `let ref mut i64 item=ref mut x;let i32 value=1;*item+=value;`, "E1006"},
		{"snapshot", `let ref mut i64 item=ref mut x;*item+={*item=100;2};`, ""},
		{"rebind", `var i64 y=20;var ref mut i64 item=ref mut x;*item+={item=ref mut y;3};assert(*item==20);`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": `module main;fn void run(){var i64 x=10;` + tc.body + `}`})
			if tc.code != "" {
				for _, d := range r.Diagnostics {
					if d.Code == tc.code {
						return
					}
				}
				t.Fatalf("missing %s: %+v %+v", tc.code, r.Diagnostics, r.Unsupported)
			}
			if len(r.Diagnostics)+len(r.Unsupported) != 0 {
				t.Fatalf("analysis %+v %+v", r.Diagnostics, r.Unsupported)
			}
		})
	}
}
func TestDerefCompoundBoundary(t *testing.T) {
	for name, src := range map[string]string{
		"bool":      `fn void bad(ref mut bool item){*item&=true;}`,
		"struct":    `struct T{i64 x;}fn void bad(ref mut T item){*item+=T{.x=1};}`,
		"temporary": `fn void bad(){var i64 x=1;*(ref mut x)+=1;}`,
		"call":      `fn ref mut i64 relay(ref mut i64 x){return x;}fn void bad(ref mut i64 item){*relay(item)+=1;}`,
		"raw":       `fn void bad(ptr mut i64 item){unsafe{*item+=1;};}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + src})
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("unadmitted compound target accepted")
			}
		})
	}
}
