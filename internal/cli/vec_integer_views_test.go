package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVecIntegerViewBorrowRejections(t *testing.T) {
	for _, element := range []string{"i32", "u32", "usize"} {
		t.Run(element, func(t *testing.T) {
			for _, mutable := range []bool{false, true} {
				base := vecU8ViewBase
				if mutable {
					base = vecU8MutViewBase
				}
				for _, item := range []struct{ name, action, code string }{
					{"push", "discard v::push<u8>(ref mut values, 42);", "E2101"},
					{"reserve-noop", "discard v::reserve<u8>(ref mut values, 0);", "E2101"},
					{"pop", "discard v::pop<u8>(ref mut values);", "E2101"},
					{"truncate-noop", "discard v::truncate<u8>(ref mut values, 100);", "E2101"},
					{"clear", "discard v::clear<u8>(ref mut values);", "E2101"},
					{"move", "let v::Vec<u8> moved = values;", "E2105"},
					{"move-call", "discard take(values);", "E2105"},
					{"replace", "values = v::empty<u8>();", "E2101"},
					{"derived", "let ref BORROWu8[] child = view[..]; discard v::clear<u8>(ref mut values); assert(child.len > 0);", "E2101"},
					{"address-effect", "let u8 unused = view[{ discard v::clear<u8>(ref mut values); 0 }];", "E2101"},
					{"bound-effect", "let ref BORROWu8[] child = view[..{ discard v::clear<u8>(ref mut values); 0 }];", "E2101"},
					{"loop-early-return", "for (let u8 byte in view) { discard take(values); return 0; }", "E2105"},
				} {
					t.Run(item.name+"/"+map[bool]string{false: "shared", true: "mutable"}[mutable], func(t *testing.T) {
						borrow := ""
						if mutable {
							borrow = "mut "
						}
						action := strings.ReplaceAll(item.action, "BORROW", borrow)
						source := strings.Replace(base, "ACTION", action, 1)
						if item.name == "loop-early-return" {
							source = strings.Replace(source, "assert(view[0] == 127);", "", 1)
						}
						root := t.TempDir()
						writeVecI64ViewWorkspace(t, root, strings.ReplaceAll(source, "u8", element))
						code, _, stderr := runForTest([]string{"check"}, root)
						if code != ExitSourceDiagnostics || !strings.Contains(stderr, item.code) {
							t.Fatalf("expected %s: %d %s", item.code, code, stderr)
						}
					})
				}
			}
			for _, item := range []struct{ name, source, code string }{
				{"shared-store", strings.Replace(vecU8ViewBase, "ACTION", "view[0] = 1;", 1), ""},
				{"exclusive-owner-read", strings.Replace(vecU8MutViewBase, "ACTION", "discard v::len<u8>(ref values);", 1), "E2101"},
				{"exclusive-competing-view", strings.Replace(vecU8MutViewBase, "ACTION", "let ref u8[] other = v::view_u8(ref values); assert(other.len > 0);", 1), "E2101"},
				{"parent", strings.Replace(vecU8MutViewBase, "ACTION", "let ref mut u8[] child = view[..]; assert(view.len > 0); assert(child.len > 0);", 1), "E2101"},
				{"siblings", strings.Replace(vecU8MutViewBase, "ACTION", "let ref mut u8[] child = view[..0]; let ref mut u8[] other = view[0..]; assert(child.is_empty); assert(other.len > 0);", 1), "E2101"},
				{"tagged-origin", strings.Replace(vecU8MutViewBase, "ACTION", "let Option<ref mut u8[]> result = tagged(ref mut values); discard v::clear<u8>(ref mut values); switch(result) { case .some(child): { assert(child.len > 0); } case .none: {} };", 1), "E2101"},
				{"local-escape", "module main; import std::vec as v; fn ref u8[] bad() { let v::Vec<u8> values = v::empty<u8>(); return v::view_u8(ref values); } fn i32 main() { return 0; }", "E2102"},
				{"mut-local-escape", "module main; import std::vec as v; fn ref mut u8[] bad() { var v::Vec<u8> values = v::empty<u8>(); return v::view_mut_u8(ref mut values); } fn i32 main() { return 0; }", "E2102"},
				{"wrong-element", "module main; import std::vec as v; fn i32 main() { let v::Vec<i64> values = v::empty<i64>(); let ref u8[] view = v::view_u8(ref values); assert(view.is_empty); return 0; }", "E1006"},
				{"user-private", "module main; fn i32 main() { let ref u8[] view = unsafe { storage_borrow<u8[]>(null,null,0) }; return 0; }", ""},
			} {
				t.Run(item.name, func(t *testing.T) {
					root := t.TempDir()
					writeVecI64ViewWorkspace(t, root, strings.ReplaceAll(item.source, "u8", element))
					code, _, stderr := runForTest([]string{"check"}, root)
					if code != ExitSourceDiagnostics || stderr == "" || item.code != "" && !strings.Contains(stderr, item.code) {
						t.Fatalf("rejection %s: %d %s", item.code, code, stderr)
					}
				})
			}
		})
	}
}

func TestVecIntegerViewsNative(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "vec_integer_views", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, source, trap string }{{"example", string(source), ""}}
	for _, element := range []string{"i32", "u32", "usize"} {
		hi := map[string]string{"i32": "2147483647", "u32": "4294967295", "usize": "18446744073709551615"}[element]
		low := map[string]string{"i32": "-2147483648", "u32": "0", "usize": "0"}[element]
		for _, item := range []struct{ name, action, trap string }{
			{"end-read", "let u8 n = view[1];", "bounds"},
			{"huge-read", "let u8 n = view[18446744073709551615];", "bounds"},
			{"empty-store", "view[1..][0] = 0;", "bounds"},
			{"past-end", "let ref mut u8[] child = view[..2];", "bounds"},
			{"reversed", "let ref mut u8[] child = view[1..0];", "bounds"},
			{"overflow", "view[0] = " + hi + "; for(let u8 item in view) { view[0] = item + 1; }", "integer-overflow"},
			{"underflow", "view[0] = " + low + "; for(let u8 item in view) { view[0] = item - 1; }", "integer-overflow"},
		} {
			s := strings.Replace(vecU8MutViewBase, "ACTION", item.action, 1)
			cases = append(cases, struct{ name, source, trap string }{element + "-" + item.name, strings.ReplaceAll(s, "u8", element), item.trap})
		}
	}
	runI64ViewNativeCases(t, "vec-integer-views", cases)
}
