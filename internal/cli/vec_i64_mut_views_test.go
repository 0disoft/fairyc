package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var vecI64MutViewBase = strings.NewReplacer("ref i64[]", "ref mut i64[]", "ref v::Vec<i64>", "ref mut v::Vec<i64>", "v::view_i64", "v::view_mut_i64", "ref values", "ref mut values").Replace(vecI64ViewBase)

func TestVecI64MutableViewCheck(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "vec_i64_mut_view", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeVecI64ViewWorkspace(t, root, string(source))
	code, _, stderr := runForTest([]string{"check"}, root)
	if code != ExitSuccess {
		t.Fatalf("check: %d %s", code, stderr)
	}
}

func TestVecI64MutableViewBorrowRejections(t *testing.T) {
	for _, item := range []struct{ name, action, code string }{
		{"push", "let Result<usize,i64> unused = v::push<i64>(ref mut values, 42);", "E2101"},
		{"reserve", "let Result<usize,u32> unused = v::reserve<i64>(ref mut values, 0);", "E2101"},
		{"pop", "let Option<i64> unused = v::pop<i64>(ref mut values);", "E2101"},
		{"truncate", "let usize unused = v::truncate<i64>(ref mut values, 100);", "E2101"},
		{"clear", "let usize unused = v::clear<i64>(ref mut values);", "E2101"},
		{"read owner", "let usize unused = v::len<i64>(ref values);", "E2101"},
		{"shared view", "let ref i64[] unused = v::view_i64(ref values); assert(unused.len > 0);", "E2101"},
		{"element", "let ref mut i64 unused = v::at_mut<i64>(ref mut values, 0);", "E2101"},
		{"second view", "let ref mut i64[] other = v::view_mut_i64(ref mut values); assert(other.len > 0);", "E2101"},
		{"move", "let v::Vec<i64> unused = values;", "E2105"},
		{"move call", "let i32 unused = take(values);", "E2105"},
		{"replace", "values = v::empty<i64>();", "E2101"},
		{"parent read", "let ref mut i64[] child = view[..]; assert(view.len > 0); assert(child.len > 0);", "E2101"},
		{"parent write", "let ref mut i64[] child = view[..]; view[0] = 1; assert(child.len > 0);", "E2101"},
		{"siblings", "let ref mut i64[] child = view[..0]; let ref mut i64[] other = view[0..]; assert(child.is_empty); assert(other.len > 0);", "E2101"},
		{"bounds mutation", "let ref mut i64[] child = view[..{ let usize unused = v::clear<i64>(ref mut values); 0 }];", "E2101"},
	} {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, strings.Replace(vecI64MutViewBase, "ACTION", item.action, 1))
			code, _, stderr := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(stderr, item.code) {
				t.Fatalf("expected %s: %d %s", item.code, code, stderr)
			}
		})
	}
	for name, source := range map[string]string{
		"derived":        strings.Replace(strings.Replace(vecI64MutViewBase, "v::view_mut_i64(ref mut values);", "tail(ref mut values);", 1), "ACTION", "let usize unused = v::clear<i64>(ref mut values);", 1),
		"tagged":         strings.Replace(strings.Replace(vecI64MutViewBase, "v::view_mut_i64(ref mut values);", "switch (tagged(ref mut values)) { case .some(value): { value } case .none: { tail(ref mut values) } };", 1), "ACTION", "let usize unused = v::clear<i64>(ref mut values);", 1),
		"local escape":   "module main; import std::vec as v; fn ref mut i64[] bad() { var v::Vec<i64> values = v::empty<i64>(); return v::view_mut_i64(ref mut values); } fn i32 main() { return 0; }",
		"wrong element":  "module main; import std::vec as v; fn i32 main() { var v::Vec<i32> values = v::empty<i32>(); let ref mut i64[] view = v::view_mut_i64(ref mut values); assert(view.is_empty); return 0; }",
		"shared owner":   "module main; import std::vec as v; fn void bad(ref v::Vec<i64> values) { let ref mut i64[] view = v::view_mut_i64(values); } fn i32 main() { return 0; }",
		"user intrinsic": "module main; fn i32 main() { let ref mut i64[] view = unsafe { storage_borrow_mut<i64[]>(null,null,0) }; return 0; }",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, source)
			code, _, stderr := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || stderr == "" {
				t.Fatalf("unsafe mutable view accepted: %d %s", code, stderr)
			}
			want := "E2101"
			if name == "local escape" {
				want = "E2102"
			}
			if (name == "derived" || name == "tagged" || name == "local escape") && !strings.Contains(stderr, want) {
				t.Fatalf("want %s: %s", want, stderr)
			}
		})
	}
}

func TestVecI64MutableViewNative(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "vec_i64_mut_view", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, source, trap string }{{"example", string(source), ""}}
	for _, item := range []struct{ name, action, trap string }{
		{"end store", "view[1] = 7;", "bounds"},
		{"huge store", "view[18446744073709551615] = 7;", "bounds"},
		{"empty store", "view[1..][0] = 7;", "bounds"},
		{"reversed", "let ref mut i64[] child = view[1..0];", "bounds"},
		{"past end", "let ref mut i64[] child = view[..2];", "bounds"},
		{"overflow", "view[0] = 9223372036854775807; view[0] += 1;", "integer-overflow"},
		{"underflow", "view[0] = -9223372036854775808; view[0] -= 1;", "integer-overflow"},
	} {
		cases = append(cases, struct{ name, source, trap string }{item.name, strings.Replace(vecI64MutViewBase, "ACTION", item.action, 1), item.trap})
	}
	runI64ViewNativeCases(t, "vec-i64-mut-view", cases)
}
