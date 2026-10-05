package semantic_test

import "testing"

func TestInlineNominalArrayFieldBoundary(t *testing.T) {
	base := `module main;struct ZToken{i64 value;}struct APage{ZToken[2] tokens;} `
	for _, tc := range []struct{ name, source, code string }{
		{"immutable", `fn void bad(){let APage p=APage{.tokens=[ZToken{.value=1};2]};p.tokens=[ZToken{.value=2};2];}`, "E1008"},
		{"shared-owner", `fn void bad(ref APage p){p.tokens=[ZToken{.value=2};2];}`, "E1008"},
		{"nominal-mismatch", `struct Other{i64 value;}fn void bad(){var APage p=APage{.tokens=[ZToken{.value=1};2]};p.tokens=[Other{.value=2};2];}`, "E1006"},
		{"missing-field", `fn void bad(){let APage p=APage{};}`, ""},
		{"wrong-length", `fn void bad(){let APage p=APage{.tokens=[ZToken{.value=1}]};}`, ""},
		{"shared-live", `fn void bad(){var APage p=APage{.tokens=[ZToken{.value=1};2]};let ref APage v=ref p;p.tokens=[ZToken{.value=2};2];let ZToken[2] a=v.tokens;assert(a[0].value==1);}`, "E2101"},
		{"field-reference", `fn void bad(ref APage p){let ref ZToken[2] v=ref p.tokens;}`, ""},
		{"mutable-field-reference", `fn void bad(ref mut APage p){let ref mut ZToken[2] v=ref mut p.tokens;}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": base + tc.source})
			for _, d := range r.Diagnostics {
				if tc.code != "" && d.Code == tc.code {
					return
				}
			}
			if tc.code == "" && len(r.Diagnostics)+len(r.Unsupported) > 0 {
				return
			}
			t.Fatalf("expected %s rejection: %+v %+v", tc.code, r.Diagnostics, r.Unsupported)
		})
	}
	for _, source := range []string{
		`@repr(C) struct P{ZToken[2] tokens;}`,
		`struct P{ZToken[0] tokens;}`,
		`struct P{ZToken[4097] tokens;}`,
		`struct P{ZToken[2][2] tokens;}`,
		`struct Pointer{ptr u8 p;}struct P{Pointer[2] tokens;}`,
		`struct Tagged{Option<i32> p;}struct P{Tagged[2] tokens;}`,
		`struct Empty{}struct P{Empty[2] tokens;}`,
		`struct Cyclic{Cyclic next;}struct P{Cyclic[2] tokens;}`,
		`struct Recursive{Recursive[2] tokens;}`,
		`resource R{cleanup{}}struct P{R[2] tokens;}`,
		`struct WithArray{u8[2] bytes;}struct P{WithArray[2] tokens;}`,
	} {
		r := analyzeText(t, map[string]string{"main.fy": base + source})
		if len(r.Diagnostics)+len(r.Unsupported) == 0 {
			t.Fatalf("accepted closed shape: %s", source)
		}
	}
}

func TestInlineNominalArrayFieldDeclarationOrder(t *testing.T) {
	for _, source := range []string{
		`struct A{Z[1] values;}struct Z{i64 x;}`,
		`struct Z{i64 x;}struct A{Z[1] values;}`,
		`struct A{Z[4096] values;}struct Z{i64 x;}`,
		`struct A{(Z[1]) values;}struct Z{i64 x;}`,
	} {
		r := analyzeText(t, map[string]string{"main.fy": "module main;" + source})
		if len(r.Diagnostics)+len(r.Unsupported) != 0 {
			t.Fatalf("analysis: %+v %+v", r.Diagnostics, r.Unsupported)
		}
	}
}
