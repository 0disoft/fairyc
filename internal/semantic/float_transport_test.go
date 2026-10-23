package semantic_test

import "testing"

func TestFloatingTransportBoundaries(t *testing.T) {
	for name, body := range map[string]string{
		"literal":    `fn f32 bad(){return 1;}`,
		"arithmetic": `fn f32 bad(f32 a,f32 b){return a+b;}`,
		"comparison": `fn bool bad(f64 a,f64 b){return a==b;}`,
		"width":      `fn f32 bad(f64 a){return a;}`,
		"c-scalar":   `fn f32 bad(c::float a){return a;}`,
		"cast":       `fn f32 bad(c::float a){return a as f32;}`,
		"array":      `struct P{f32[2] values;}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + body})
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("unsupported floating operation admitted")
			}
		})
	}
}
