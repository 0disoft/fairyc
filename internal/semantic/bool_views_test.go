package semantic_test

import (
	"github.com/0disoft/fairyc/internal/mir"
	"testing"
)

func TestBoolViewFlow(t *testing.T) {
	for _, tc := range []struct{ name, source, code string }{
		{"shared-return", `fn ref bool first(ref bool[] flags){for(let ref bool item in flags){return item;}return ref flags[0];}`, ""},
		{"direct-named-slice", `fn void visit(ref bool[] flags){for(let bool flag in flags[..]){assert(flag);}}`, ""},
		{"reborrow", `fn void edit(ref mut bool[] flags){ {let ref mut bool[] child=flags[..];child[0]=true;}flags[0]=false;}`, ""},
		{"nested-shared", `fn void visit(){let bool[1] flags=[true];for(let ref bool a in flags){for(let ref bool b in flags){assert(*a==*b);}}}`, ""},
		{"shared-store", `fn void bad(ref bool[] flags){flags[0]=true;}`, "E1008"},
		{"shared-item-store", `fn void bad(ref bool[] flags){for(let ref bool item in flags){*item=false;}}`, "E1008"},
		{"shared-loop-write", `fn void bad(ref mut bool[] flags){for(let ref bool item in flags){assert(*item);flags[0]=false;break;}}`, "E2101"},
		{"array-loop-write", `fn void bad(){var bool[1] flags=[true];for(let bool item in flags){flags[0]=false;assert(item);}}`, "E2101"},
		{"exclusive-array-read", `fn void bad(){var bool[1] flags=[true];for(let ref mut bool item in flags){*item=false;assert(flags[0]);}}`, "E2101"},
		{"local-escape", `fn ref bool bad(){let bool[1] flags=[true];for(let ref bool item in flags){return item;}return ref flags[0];}`, "E2102"},
		{"view-escape", `fn ref bool[] bad(){let bool[1] flags=[true];return ref flags;}`, "E2102"},
		{"parent-use", `fn void bad(ref mut bool[] flags){let ref mut bool[] child=flags[1..];flags[0]=true;child[0]=false;}`, "E2101"},
		{"siblings", `fn void bad(ref mut bool[] flags){let ref mut bool[] a=flags[..1];let ref mut bool[] b=flags[1..];a[0]=true;b[0]=false;}`, "E2101"},
		{"numeric-store", `fn void bad(ref mut bool[] flags){flags[0]=1;}`, "E1006"},
		{"wrong-element", `fn void take(ref bool[] flags){}fn void bad(){let i32[1] data=[1];take(ref data);}`, "E1006"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main;" + tc.source})
			if tc.code != "" {
				for _, d := range result.Diagnostics {
					if d.Code == tc.code {
						return
					}
				}
				t.Fatalf("missing %s: %+v %+v", tc.code, result.Diagnostics, result.Unsupported)
			}
			if len(result.Diagnostics)+len(result.Unsupported) != 0 {
				t.Fatalf("analysis: %+v %+v", result.Diagnostics, result.Unsupported)
			}
			p, err := mir.Lower(result.HIR)
			if err != nil {
				t.Fatal(err)
			}
			if d := mir.Verify(p); len(d) != 0 {
				t.Fatal(d)
			}
		})
	}
}

func TestBoolViewBoundary(t *testing.T) {
	for name, source := range map[string]string{
		"owned-slice":        `fn void bad(bool[] flags){}`,
		"mutable-descriptor": `fn void bad(ref bool[] flags){var ref bool[] alias=flags;for(let bool flag in alias){}}`,
		"computed-root":      `fn ref bool[] same(ref bool[] flags){return flags;}fn void bad(ref bool[] flags){for(let bool flag in same(flags)[..]){}}`,
		"immutable-array":    `fn void bad(){let bool[1] flags=[true];for(let ref mut bool flag in flags){*flag=false;}}`,
		"zero-array":         `fn void bad(){var bool[0] flags=[false;0];}`,
		"oversized-array":    `fn void bad(){var bool[4097] flags=[false;4097];}`,
		"wrong-loop-item":    `fn void bad(ref bool[] flags){for(let i32 flag in flags){}}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + source})
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("unadmitted bool view accepted")
			}
		})
	}
}
