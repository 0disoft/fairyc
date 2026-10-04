package semantic_test

import "testing"

func TestStructArrayFieldRejections(t *testing.T) {
	base := `struct Pos{i64 x;}struct Point{Pos pos;}fn void edit(ref mut Point p){p.pos.x=9;}`
	for _, tc := range []struct{ name, source, code string }{
		{"immutable", `fn void bad(){let Point[2] a=[Point{.pos=Pos{.x=1}};2];a[0].pos.x=9;}`, "E1008"},
		{"temporary", `fn Point[1] make(){return [Point{.pos=Pos{.x=1}}];}fn void bad(){make()[0].pos.x=9;}`, "E1008"},
		{"shared-view", `fn void bad(ref Point[] a){a[0].pos.x=9;}`, "E1008"},
		{"nominal-field", `struct Other{i64 x;}fn void bad(){var Point[1] a=[Point{.pos=Pos{.x=1}}];a[0].pos=Other{.x=2};}`, "E1006"},
		{"shared-live", `let ref Point item=ref a[1];a[0].pos.x=9;assert(item.pos.x==1);`, "E2101"},
		{"grouped-shared-live", `let ref Point item=ref a[1];((a)[0]).pos.x=9;assert(item.pos.x==1);`, "E2101"},
		{"grouped-index-last-use", `let ref Point item=ref a[1];(a)[{assert(item.pos.x==1);0}].pos.x=9;`, "E2101"},
		{"exclusive-live", `let ref mut Point item=ref mut a[1];a[0].pos.x=9;assert(item.pos.x==1);`, "E2101"},
		{"shared-rhs-last-use", `let ref Point item=ref a[1];a[0].pos.x=item.pos.x;`, "E2101"},
		{"shared-index-last-use", `let ref Point item=ref a[1];a[{assert(item.pos.x==1);0}].pos.x=9;`, "E2101"},
		{"view-live", `let ref Point[] view=ref a;a[0].pos.x=9;assert(view.len==2);`, "E2101"},
		{"index-write", `a[{a[1].pos.x=9;0}].pos.x=2;`, "E2101"},
		{"index-borrow", `a[{edit(ref mut a[1]);0}].pos.x=2;`, "E2101"},
		{"rhs-read", `a[0].pos.x=a[1].pos.x;`, "E2101"},
		{"rhs-write", `a[0].pos.x={a[1].pos.x=9;2};`, "E2101"},
		{"rhs-array-replace", `a[0].pos.x={a=[Point{.pos=Pos{.x=9}};2];2};`, "E2101"},
		{"rhs-borrow", `a[0].pos.x={edit(ref mut a[1]);2};`, "E2101"},
		{"loop-live", `for(let ref Point p in a){a[0].pos.x=9;assert(p.pos.x==1);}`, "E2101"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := tc.source
			if tc.code == "E2101" {
				src = `fn void bad(){var Point[2] a=[Point{.pos=Pos{.x=1}};2];` + src + `}`
			}
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + base + src})
			for _, d := range r.Diagnostics {
				if d.Code == tc.code {
					return
				}
			}
			t.Fatalf("missing %s: %+v %+v", tc.code, r.Diagnostics, r.Unsupported)
		})
	}
}

func TestStructArrayFieldLoanEnds(t *testing.T) {
	r := analyzeText(t, map[string]string{"main.fy": `module main;struct Point{i64 x;}
fn void run(){var Point[2] a=[Point{.x=1};2];let ref Point p=ref a[1];assert(p.x==1);a[0].x=9;assert(a[0].x==9);}`})
	if len(r.Diagnostics)+len(r.Unsupported) != 0 {
		t.Fatalf("analysis: %+v %+v", r.Diagnostics, r.Unsupported)
	}
}
