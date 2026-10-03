package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestI64ForNative(t *testing.T) {
	var cases []struct{ name, source, trap string }
	for _, name := range []string{"i64_for", "vec_i64_view"} {
		source, err := os.ReadFile(filepath.Join("..", "..", "examples", name, "src", "main.fy"))
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, struct{ name, source, trap string }{name, string(source), ""})
	}
	for name, values := range map[string]string{"overflow": "9223372036854775807,1", "underflow": "-9223372036854775808,-1"} {
		source := "module main; fn i32 main() { let i64[2] data = [" + values + "]; let ref i64[] view = ref data; var i64 total = 0; for (let i64 value in view) { total += value; } return 0; }"
		cases = append(cases, struct{ name, source, trap string }{name, source, "integer-overflow"})
	}
	runI64ViewNativeCases(t, "i64-for", cases)
}

func TestI64ForVecOwnerBorrow(t *testing.T) {
	for _, item := range []struct{ name, action, code string }{
		{"push", "let Result<usize,i64> unused = v::push<i64>(ref mut values, 42);", "E2101"},
		{"reserve", "let Result<usize,u32> unused = v::reserve<i64>(ref mut values, 100);", "E2101"},
		{"clear", "let usize unused = v::clear<i64>(ref mut values);", "E2101"},
		{"move", "let v::Vec<i64> moved = values;", "E2105"},
		{"call-move", "let i32 unused = take(values);", "E2105"},
		{"replace", "values = v::empty<i64>();", "E2101"},
		{"push-break", "let Result<usize,i64> unused = v::push<i64>(ref mut values, 42); break;", "E2101"},
		{"move-break", "let v::Vec<i64> moved = values; break;", "E1105"},
	} {
		t.Run(item.name, func(t *testing.T) {
			text := strings.Replace(vecI64ViewBase, "ACTION", "for (let i64 value in view) { "+item.action+" }", 1)
			// The loop is the view's last use; the hold must survive early exit.
			text = strings.Replace(text, "assert(view[0] == 4000000000);", "", 1)
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, text)
			code, _, stderr := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(stderr, item.code) {
				t.Fatalf("expected %s: %d %s", item.code, code, stderr)
			}
		})
	}
}
