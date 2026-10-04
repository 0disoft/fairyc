package semantic_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestSharedIntegerForFlow(t *testing.T) {
	for _, typ := range []string{"u8", "i32", "i64", "u32", "usize"} {
		for _, tc := range []struct{ name, source, code string }{
			{"shared-view", `fn void test(ref T[] values){for(let ref T item in values){assert(*item>0);assert(values.len>0);}}`, ""},
			{"mutable-view-read", `fn void test(ref mut T[] values){for(let ref T item in values){assert(*item==values[0]);}}`, ""},
			{"mutable-view-write", `fn void test(ref mut T[] values){for(let ref T item in values){assert(*item>0);values[0]=9;}}`, "E2101"},
			{"alias-write", `fn void test(ref mut T[] values){let ref mut T[] alias=values;for(let ref T item in values){assert(*item>0);alias[0]=9;break;}}`, "E2101"},
			{"mutable-reborrow", `fn void use(ref mut T value){}fn void test(ref mut T[] values){for(let ref T item in values){assert(*item>0);use(ref mut values[0]);}}`, "E2101"},
			{"shared-return", `fn ref T first(ref T[] values){for(let ref T item in values){return item;}return ref values[0];}`, ""},
			{"shared-return-mut-parent", `fn ref T first(ref mut T[] values){for(let ref T item in values){return item;}return ref values[0];}`, ""},
			{"arrays", `fn void test(){var T[2] data=[1,2];for(let ref T item in data){assert(*item>0);assert(data.len==2);}data[0]=9;}`, ""},
			{"immutable-array", `fn void test(){let T[1] data=[1];for(let ref T item in data){assert(*item==1);}}`, ""},
			{"array-owner-write", `fn void test(){var T[1] data=[1];for(let ref T item in data){assert(*item>0);data[0]=9;return;}}`, "E2101"},
			{"nested-shared", `fn void test(){var T[1] data=[1];for(let ref T a in data){for(let ref T b in data){assert(*a==*b);}}}`, ""},
			{"shared-write", `fn void test(ref T[] values){for(let ref T item in values){*item=9;}}`, "E1008"},
			{"local-escape", `fn ref T first(){var T[1] data=[1];for(let ref T item in data){return item;}return ref data[0];}`, "E2102"},
			{"parameter-escape", `fn ref T first(T[1] data){for(let ref T item in data){return item;}return ref data[0];}`, "E2102"},
			{"scope", `fn void test(ref T[] values){for(let ref T item in values){}assert(*item==1);}`, "E0101"},
		} {
			t.Run(typ+"/"+tc.name, func(t *testing.T) {
				r := analyzeText(t, map[string]string{"main.fy": "module main;" + strings.ReplaceAll(tc.source, "T", typ)})
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
}

func TestSharedIntegerForBoundary(t *testing.T) {
	for name, source := range map[string]string{
		"wrong-width":        `fn void test(ref i32[] values){for(let ref i64 item in values){}}`,
		"mutable-descriptor": `fn void test(ref i64[] values){var ref i64[] alias=values;for(let ref i64 item in alias){}}`,
		"temporary":          `fn void test(ref i64[] values){for(let ref i64 item in values[..]){}}`,
		"bool":               `fn void test(){var bool[1] values=[true];for(let ref bool item in values){}}`,
		"resource":           `resource Token{i32 code;cleanup{}}fn void test(ref Token[] values){for(let ref Token item in values){}}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + source})
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("unadmitted shared item accepted")
			}
		})
	}
}
