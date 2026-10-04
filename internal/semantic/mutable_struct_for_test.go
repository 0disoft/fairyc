package semantic_test

import (
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"testing"
)

func TestMutableStructForFlow(t *testing.T) {
	for _, tc := range []struct{ name, body, code string }{
		{"stored-field", "for(let ref mut Token item in view){item.value+=1;}", ""},
		{"field-child", "for(let ref mut Token item in view){let ref mut i64 value=ref mut item.value;*value=9;}", ""},
		{"after-last-use", "for(let ref mut Token item in view){item.value=9;assert(view[0].value==9);}", ""},
		{"parent-live", "for(let ref mut Token item in view){assert(view.len>0);item.value=9;}", "E2101"},
		{"alias-live", "let ref mut Token[] alias=view;for(let ref mut Token item in view){alias[0].value=9;item.value=1;}", "E2101"},
		{"parent-before-break", "for(let ref mut Token item in view){view[0].value=9;assert(item.value>0);break;}", "E2101"},
		{"parent-before-return", "for(let ref mut Token item in view){view[0].value=9;assert(item.value>0);return;}", "E2101"},
		{"nested-same", "for(let ref mut Token outer in view){for(let ref mut Token inner in view){inner.value=9;}outer.value=1;}", "E2101"},
		{"retained-field", "for(let ref mut Token item in view){let ref mut i64 child=ref mut item.value;item.value=9;assert(*child==1);}", "E2101"},
		{"after-loop", "for(let ref mut Token item in view){item.value=9;break;}view[0].value=1;", ""},
		{"scope", "for(let ref mut Token item in view){}item.value=1;", "E0101"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;struct Token{i64 value;}fn void edit(ref mut Token[] view){" + tc.body + "}"})
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
func TestMutableStructForSourceBoundary(t *testing.T) {
	for name, body := range map[string]string{
		"shared-source":      "fn void edit(ref Token[] view){for(let ref mut Token item in view){}}",
		"copied-item":        "fn void edit(ref mut Token[] view){for(let Token item in view){}}",
		"mutable-descriptor": "fn void edit(ref mut Token[] view){var ref mut Token[] alias=view;for(let ref mut Token item in alias){}}",
		"temporary":          "fn void edit(ref mut Token[] view){for(let ref mut Token item in view[..]){}}",
		"wrong-nominal":      "fn void edit(ref mut Token[] view){for(let ref mut Other item in view){}}",
		"integer-ref":        "fn void edit(ref mut i64[] view){for(let ref mut i64 item in view){}}",
		"whole-deref-store":  "fn void edit(ref mut Token[] view){for(let ref mut Token item in view){*item=Token{.value=1};}}",
		"direct-element":     "fn void edit(ref mut Token[] view){let ref mut Token item=ref mut view[0];}",
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;struct Token{i64 value;}struct Other{i64 value;}" + body})
			if len(r.Diagnostics) == 0 && len(r.Unsupported) == 0 {
				t.Fatal("unadmitted source accepted")
			}
		})
	}
}
