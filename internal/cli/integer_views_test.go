package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIntegerViewsNative(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "integer_views", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, source, trap string }{{"example", string(source), ""}}
	for _, element := range []string{"i32", "u32", "usize"} {
		base := "module main; fn i32 main() { var " + element + "[2] data = [1,2]; let ref mut " + element + "[] view = ref mut data; "
		for _, item := range []struct{ name, body string }{
			{"end-read", "let " + element + " unused = view[2];"},
			{"end-store", "view[2] = 3;"},
			{"empty-store", "view[2..][0] = 3;"},
			{"reverse-slice", "let ref mut " + element + "[] unused = view[2..1];"},
			{"past-end", "let ref mut " + element + "[] unused = view[..3];"},
		} {
			cases = append(cases, struct{ name, source, trap string }{element + "-" + item.name, base + item.body + "return 0; }", "bounds"})
		}
	}
	runI64ViewNativeCases(t, "integer-views", cases)
}
