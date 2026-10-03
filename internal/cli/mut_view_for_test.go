package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const mutableViewForVecSource = `module main;
import std::vec as v;
fn i32 main() {
    var v::Vec<i64> values = v::empty<i64>();
    let ref mut i64[] empty = v::view_mut_i64(ref mut values);
    for (let i64 value in empty) { assert(false); }
    assert(switch(v::push<i64>(ref mut values, 4000000000)) { case .ok(_): { true } case .err(_): { false } });
    assert(switch(v::push<i64>(ref mut values, -7)) { case .ok(_): { true } case .err(_): { false } });
    let ref mut i64[] view = v::view_mut_i64(ref mut values);
    var usize index = 0;
    var i64 total = 0;
    for (let i64 value in view) {
        total += value;
        view[index] = value + 1;
        index += 1;
    }
    assert(total == 3999999993 && index == 2);
    assert(switch(v::push<i64>(ref mut values, 9)) { case .ok(_): { true } case .err(_): { false } });
    let ref mut i64[] grown = v::view_mut_i64(ref mut values);
    let ref mut i64[] child = grown[1..];
    for (let i64 value in child) { assert(value == -6); child[0] = 8; break; }
    assert(grown[0] == 4000000001 && grown[1] == 8 && grown[2] == 9);
    let v::Vec<i64> moved = values;
    assert(v::len<i64>(ref moved) == 3);
    return 0;
}`

func TestMutableViewForNative(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "mut_view_for", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, source, trap string }{
		{"example", string(source), ""},
		{"vec", mutableViewForVecSource, ""},
	}
	for _, item := range []struct{ name, element, value, operator string }{
		{"u8-overflow", "u8", "255", "+"},
		{"u8-underflow", "u8", "0", "-"},
		{"i64-overflow", "i64", "9223372036854775807", "+"},
		{"i64-underflow", "i64", "-9223372036854775808", "-"},
	} {
		source := "module main; fn i32 main() { var ELEMENT[1] data = [VALUE]; let ref mut ELEMENT[] view = ref mut data; for (let ELEMENT value in view) { view[0] = value OP 1; } return 0; }"
		source = strings.NewReplacer("ELEMENT", item.element, "VALUE", item.value, "OP", item.operator).Replace(source)
		cases = append(cases, struct{ name, source, trap string }{item.name, source, "integer-overflow"})
	}
	runI64ViewNativeCases(t, "mut-view-for", cases)
}

func TestMutableViewForVecOwnerBorrow(t *testing.T) {
	for _, item := range []struct{ name, action, code string }{
		{"push", "discard v::push<i64>(ref mut values, 42);", "E2101"},
		{"reserve", "discard v::reserve<i64>(ref mut values, 0);", "E2101"},
		{"read", "discard v::len<i64>(ref values);", "E2101"},
		{"pop", "discard v::pop<i64>(ref mut values);", "E2101"},
		{"clear", "discard v::clear<i64>(ref mut values);", "E2101"},
		{"replace", "values = v::empty<i64>();", "E2101"},
		{"move-call", "discard take(values);", "E2105"},
	} {
		for _, exit := range []string{"", "break;", "return 0;"} {
			t.Run(item.name+"/"+exit, func(t *testing.T) {
				source := strings.Replace(vecI64MutViewBase, "ACTION", "for (let i64 value in view) { "+item.action+exit+" }", 1)
				source = strings.Replace(source, "assert(view[0] == 4000000000);", "", 1)
				root := t.TempDir()
				writeVecI64ViewWorkspace(t, root, source)
				code, _, stderr := runForTest([]string{"check"}, root)
				want := item.code
				if item.name == "move-call" && exit == "break;" {
					want = "E1105"
				}
				if code != ExitSourceDiagnostics || !strings.Contains(stderr, want) {
					t.Fatalf("expected %s: %d %s", item.code, code, stderr)
				}
			})
		}
	}
}

// The same held-loan rule covers existing shared loops on an early return.
func TestViewForOwnerMoveBeforeReturn(t *testing.T) {
	for _, base := range []string{vecI64ViewBase, vecI64MutViewBase} {
		for _, action := range []string{"discard take(values); return 0;", "let v::Vec<i64> moved = values; return 0;"} {
			source := strings.Replace(base, "ACTION", "for (let i64 value in view) { "+action+" }", 1)
			source = strings.Replace(source, "assert(view[0] == 4000000000);", "", 1)
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, source)
			code, _, stderr := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(stderr, "E2105") {
				t.Fatalf("held owner move: %d %s", code, stderr)
			}
		}
	}
}
