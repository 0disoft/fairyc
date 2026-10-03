package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const vecU8ViewBase = `module main;
import std::vec as v;
fn i32 take(v::Vec<u8> values) { return 0; }
fn ref u8[] tail(ref v::Vec<u8> values) { return v::view_u8(values)[..]; }
fn Option<ref u8[]> tagged(ref v::Vec<u8> values) { return .some(tail(values)); }
fn i32 main() {
    var v::Vec<u8> values = v::empty<u8>();
    assert(switch (v::push<u8>(ref mut values, 127)) { case .ok(_): { true } case .err(_): { false } });
    let ref u8[] view = v::view_u8(ref values);
    ACTION
    assert(view[0] == 127);
    return 0;
}`

var vecU8MutViewBase = strings.NewReplacer("ref u8[]", "ref mut u8[]", "ref v::Vec<u8>", "ref mut v::Vec<u8>", "v::view_u8", "v::view_mut_u8", "ref values", "ref mut values").Replace(vecU8ViewBase)

func TestVecU8ViewCheck(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "vec_u8_view", "src", "main.fy"))
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

func TestVecU8ViewBorrowRejections(t *testing.T) {
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
				writeVecI64ViewWorkspace(t, root, source)
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
			writeVecI64ViewWorkspace(t, root, item.source)
			code, _, stderr := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || stderr == "" || item.code != "" && !strings.Contains(stderr, item.code) {
				t.Fatalf("rejection %s: %d %s", item.code, code, stderr)
			}
		})
	}
}

func TestVecU8ViewNative(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "vec_u8_view", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, source, trap string }{{"example", string(source), ""}}
	// The shared storage path must preserve the existing wide-element backend.
	wide, err := os.ReadFile(filepath.Join("..", "..", "examples", "vec_i64_mut_view", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	cases = append(cases, struct{ name, source, trap string }{"i64-regression", string(wide), ""})
	for _, item := range []struct {
		name, action string
		mutable      bool
		trap         string
	}{
		{"read-end", "let u8 unused = view[1];", false, "bounds"},
		{"read-empty", "let u8 unused = view[1..][0];", false, "bounds"},
		{"read-huge", "let u8 unused = view[18446744073709551615];", false, "bounds"},
		{"reversed", "let ref u8[] child = view[1..0];", false, "bounds"},
		{"past-end", "let ref mut u8[] child = view[..2];", true, "bounds"},
		{"store-end", "view[1] = 7;", true, "bounds"},
		{"store-empty", "view[1..][0] = 7;", true, "bounds"},
		{"overflow", "view[0] = 255; view[0] += 1;", true, "integer-overflow"},
		{"underflow", "view[0] = 0; view[0] -= 1;", true, "integer-overflow"},
	} {
		base := vecU8ViewBase
		if item.mutable {
			base = vecU8MutViewBase
		}
		cases = append(cases, struct{ name, source, trap string }{item.name, strings.Replace(base, "ACTION", item.action, 1), item.trap})
	}
	runI64ViewNativeCases(t, "vec-u8-view", cases)
}
