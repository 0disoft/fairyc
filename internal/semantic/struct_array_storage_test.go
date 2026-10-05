package semantic_test

import "testing"

func TestStructArrayStorageBoundary(t *testing.T) {
	for _, tc := range []struct{ name, source, code string }{
		{"immutable", `struct P{u8[2] bytes;}fn void bad(){let P p=P{.bytes=[1,2]};p.bytes=[3,4];}`, "E1008"},
		{"width", `struct P{u8[2] bytes;}fn void bad(){var P p=P{.bytes=[1,2]};let i32[2] b=[1,2];p.bytes=b;}`, "E1006"},
		{"length", `struct P{u8[2] bytes;}fn void bad(){var P p=P{.bytes=[1,2]};p.bytes=[1,2,3];}`, ""},
		{"repr-c", `@repr(C) struct P{u8[2] bytes;}`, ""},
		{"zero", `struct P{u8[0] bytes;}`, ""},
		{"oversize", `struct P{u8[4097] bytes;}`, ""},
		{"nominal", `struct T{i32 x;}struct P{T[2] data;}`, ""},
		{"resource", `resource R{cleanup{}}struct P{R[2] data;}`, ""},
		{"nested-array", `struct P{u8[2][2] data;}`, ""},
		{"shared-owner", `struct P{u8[2] bytes;i32 other;}fn void bad(){var P p=P{.bytes=[1,2],.other=0};let ref P v=ref p;p.bytes=[3,4];assert(v.other==0);}`, "E2101"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + tc.source})
			if tc.code == "" {
				if len(r.Diagnostics)+len(r.Unsupported) > 0 {
					return
				}
			} else {
				for _, d := range r.Diagnostics {
					if d.Code == tc.code {
						return
					}
				}
			}
			t.Fatalf("expected %s rejection: %+v %+v", tc.code, r.Diagnostics, r.Unsupported)
		})
	}
}
