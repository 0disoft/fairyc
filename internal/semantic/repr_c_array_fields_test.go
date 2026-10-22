package semantic_test

import "testing"

func TestReprCArrayFieldBoundaries(t *testing.T) {
	for name, text := range map[string]string{
		"bool":           `@repr(C) struct P { bool[2] data; }`,
		"zero":           `@repr(C) struct P { u8[0] data; }`,
		"oversize":       `@repr(C) struct P { u8[4097] data; }`,
		"nested":         `@repr(C) struct P { u8[2][2] data; }`,
		"struct-element": `@repr(C) struct E { u8 tag; } @repr(C) struct P { E[2] data; }`,
		"shared-view":    `@repr(C) struct P { u8[2] data; } fn void bad(){ let P p=P{.data=[1,2]}; let ref u8[] v=ref p.data; }`,
		"mutable-view":   `@repr(C) struct P { u8[2] data; } fn void bad(){ var P p=P{.data=[1,2]}; let ref mut u8[] v=ref mut p.data; }`,
		"element-store":  `@repr(C) struct P { u8[2] data; } fn void bad(){ var P p=P{.data=[1,2]}; p.data[0]=3; }`,
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + text})
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("unsupported raw array shape or access accepted")
			}
		})
	}
}
