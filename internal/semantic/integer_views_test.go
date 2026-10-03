package semantic_test

import (
	"strings"
	"testing"
)

func TestAnalyzeIntegerViewRejections(t *testing.T) {
	for _, element := range []string{"i32", "u32", "usize"} {
		for name, body := range map[string]string{
			"local-escape":     "fn ref T[] bad() { let T[2] data = [1,2]; return ref data; }",
			"owned-slice":      "fn void bad(T[] view) {}",
			"shared-store":     "fn void bad(ref T[] view) { view[0] = 3; }",
			"wrong-element":    "fn void read(ref T[] view) {} fn void bad() { let u8[2] data = [1,2]; read(ref data); }",
			"live-owner-write": "fn void bad() { var T[2] data = [1,2]; let ref T[] view = ref data; data[0] = 3; assert(view[0] == 1); }",
			"parent-use":       "fn void bad(ref mut T[] view) { let ref mut T[] child = view[1..]; view[0] = 3; child[0] = 4; }",
			"siblings":         "fn void bad(ref mut T[] view) { let ref mut T[] a = view[..1]; let ref mut T[] b = view[1..]; a[0] = 3; b[0] = 4; }",
			"bounds-effect":    "fn void bad() { var T[2] data = [1,2]; let ref T[] view = ref data; let T n = view[{data[0] = 3; 0}]; }",
		} {
			t.Run(element+"/"+name, func(t *testing.T) {
				result := analyzeText(t, map[string]string{"main.fy": "module main; " + strings.ReplaceAll(body, "T", element) + " fn i32 main() {return 0;}"})
				if len(result.Diagnostics) == 0 && len(result.Unsupported) == 0 {
					t.Fatal("unsafe integer view accepted")
				}
			})
		}
	}
}
