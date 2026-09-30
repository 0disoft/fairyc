package semantic_test

import "testing"

const exclusiveSubviewPrelude = `module main;
fn void edit(ref mut u8[] view) { view[0] = 9; }
fn void read(ref u8[] view) { assert(view.len > 0); }
fn void both(ref mut u8[] left, ref mut u8[] right) {}
fn ref mut u8[] tail(ref mut u8[] view) { return view[1..]; }
fn Option<ref mut u8[]> tagged(ref mut u8[] view) { return .some(view[1..]); }
`

func TestAnalyzeExclusiveSubview(t *testing.T) {
	for name, body := range map[string]string{
		"last use":          "let ref mut u8[] child = view[1..3]; child[0] = 9; assert(child[0] == 9); assert(view[1] == 9);",
		"omitted start":     "let ref mut u8[] child = view[..2]; edit(child); assert(view[0] == 9);",
		"omitted end":       "let ref mut u8[] child = view[1..]; edit(child); assert(view[1] == 9);",
		"full length bound": "let ref mut u8[] child = view[0..view.len]; edit(child); assert(view[0] == 9);",
		"nested":            "let ref mut u8[] child = view[1..]; let ref mut u8[] leaf = child[..1]; edit(leaf); read(child); edit(view);",
		"returned":          "let ref mut u8[] child = tail(view); edit(child); edit(view);",
		"shared reader":     "let ref mut u8[] child = view[..]; read(child); edit(child); edit(view);",
		"loop":              "var usize i = 0; while (i < 2) { let ref mut u8[] child = view[1..]; edit(child); i += 1; } edit(view);",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": exclusiveSubviewPrelude + "fn i32 main() { var u8[4] data = [1,2,3,4]; let ref mut u8[] view = ref mut data; " + body + " return 0; }"})
			assertCleanTypeResult(t, result)
		})
	}
}

func TestAnalyzeExclusiveSubviewConflicts(t *testing.T) {
	for name, body := range map[string]string{
		"parent write":     "let ref mut u8[] child = view[1..]; view[0] = 2; read(child);",
		"parent read":      "let ref mut u8[] child = view[1..]; assert(view[0] == 1); read(child);",
		"owner read":       "let ref mut u8[] child = view[1..]; assert(data[0] == 1); read(child);",
		"copied parent":    "let ref mut u8[] alias = view; let ref mut u8[] child = view[1..]; edit(alias); read(child);",
		"child alias":      "let ref mut u8[] child = view[1..]; let ref mut u8[] alias = child; edit(view); read(alias);",
		"sibling":          "let ref mut u8[] child = view[..2]; let ref mut u8[] sibling = view[2..]; read(child); read(sibling);",
		"nested parent":    "let ref mut u8[] child = view[1..]; let ref mut u8[] leaf = child[..1]; edit(child); read(leaf);",
		"earlier argument": "both(view[1..], view);",
		"later argument":   "both(view, view[1..]);",
		"bounds mutation":  "let ref mut u8[] child = view[0..{ edit(view); 2 }]; read(child);",
		"held target":      "let ref mut u8[] child = view[1..]; child[0] = view[0];",
		"returned result":  "let ref mut u8[] child = tail(view); edit(view); read(child);",
		"tagged result":    "let Option<ref mut u8[]> child = tagged(view); edit(view); switch (child) { case .some(value): { read(value); } case .none: {} };",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": exclusiveSubviewPrelude + "fn i32 main() { var u8[4] data = [1,2,3,4]; let ref mut u8[] view = ref mut data; " + body + " return 0; }"})
			if len(result.Diagnostics) == 0 || len(result.Unsupported) != 0 {
				t.Fatalf("want conflict: %+v %+v", result.Diagnostics, result.Unsupported)
			}
			for _, d := range result.Diagnostics {
				if d.Code != "E2101" {
					t.Fatalf("unrelated error: %+v", d)
				}
			}
		})
	}
}

func TestAnalyzeExclusiveSubviewEscape(t *testing.T) {
	result := analyzeText(t, map[string]string{"main.fy": exclusiveSubviewPrelude + "fn ref mut u8[] escape() { var u8[2] data = [1,2]; let ref mut u8[] view = ref mut data; return view[1..]; }"})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2102" || len(result.Unsupported) != 0 {
		t.Fatalf("local escape: %+v %+v", result.Diagnostics, result.Unsupported)
	}
}
