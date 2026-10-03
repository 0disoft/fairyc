package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// Also runs in the deferred Linux batch through the existing native harness.
func TestI64MutableViewNative(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "i64_mut_view", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, source, trap string }{
		{"subview-reverse", "module main;\nfn void reverse(ref mut i64[] values) {\n    var usize left = 0;\n    var usize right = values.len;\n    while (left < right) {\n        right -= 1;\n        let i64 saved = values[left];\n        values[left] = values[right];\n        values[right] = saved;\n        left += 1;\n    }\n}\nfn i32 main() { var i64[5] data = [9223372036854775807,4000000000,-7,-9223372036854775808,9]; let ref mut i64[] view = ref mut data; let ref mut i64[] child = view[1..4]; reverse(child); if (view[0] != 9223372036854775807) { return 11; } if (view[1] != -9223372036854775808) { return 12; } if (view[2] != -7) { return 13; } if (view[3] != 4000000000) { return 14; } if (view[4] != 9) { return 15; } return 0; }", ""},
		{"local-reverse", "module main; fn i32 main() { var i64[5] data = [9223372036854775807,4000000000,-7,-9223372036854775808,9]; let ref mut i64[] view = ref mut data; let ref mut i64[] child = view[1..4]; \n    var usize left = 0;\n    var usize right = child.len;\n    while (left < right) {\n        right -= 1;\n        let i64 saved = child[left];\n        child[left] = child[right];\n        child[right] = saved;\n        left += 1;\n    } if (view[0] != 9223372036854775807) { return 11; } if (view[1] != -9223372036854775808) { return 12; } if (view[2] != -7) { return 13; } if (view[3] != 4000000000) { return 14; } if (view[4] != 9) { return 15; } return 0; }", ""},
		{"example", string(source), ""},
		{"subview-store", "module main; fn void put(ref mut i64[] values) { values[0] = 4000000000; } fn i32 main() { var i64[2] data = [1,2]; let ref mut i64[] view = ref mut data; let ref mut i64[] child = view[1..]; put(child); assert(view[1] == 4000000000); return 0; }", ""},
	}
	for _, item := range []struct{ name, action, trap string }{
		{"index-end", "view[2] = 7;", "bounds"},
		{"index-huge", "view[18446744073709551615] = 7;", "bounds"},
		{"empty", "view[2..][0] = 7;", "bounds"},
		{"reversed", "let ref mut i64[] child = view[2..1];", "bounds"},
		{"past-end", "let ref mut i64[] child = view[..3];", "bounds"},
		{"overflow", "view[0] += 1;", "integer-overflow"},
		{"underflow", "view[1] -= 1;", "integer-overflow"},
	} {
		cases = append(cases, struct{ name, source, trap string }{item.name, "module main; fn i32 main() { var i64[2] data = [9223372036854775807,-9223372036854775808]; let ref mut i64[] view = ref mut data; " + item.action + " return 0; }", item.trap})
	}
	runI64ViewNativeCases(t, "i64-mut-view", cases)
}
