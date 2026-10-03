package semantic_test

import (
	"strings"
	"testing"
)

func TestAnalyzeI64MutableViewConflicts(t *testing.T) {
	prelude := strings.ReplaceAll(exclusiveSubviewPrelude, "u8", "i64")
	for name, body := range map[string]string{
		"parent write":     "let ref mut i64[] child = view[1..]; view[0] = 2; read(child);",
		"parent read":      "let ref mut i64[] child = view[1..]; assert(view[0] == 1); read(child);",
		"owner read":       "let ref mut i64[] child = view[1..]; assert(data[0] == 1); read(child);",
		"owner write":      "data[0] = 2; read(view);",
		"copied parent":    "let ref mut i64[] alias = view; let ref mut i64[] child = view[1..]; edit(alias); read(child);",
		"child alias":      "let ref mut i64[] child = view[1..]; let ref mut i64[] alias = child; edit(view); read(alias);",
		"sibling":          "let ref mut i64[] child = view[..2]; let ref mut i64[] sibling = view[2..]; read(child); read(sibling);",
		"nested parent":    "let ref mut i64[] child = view[1..]; let ref mut i64[] leaf = child[..1]; edit(child); read(leaf);",
		"earlier argument": "both(view[1..], view);",
		"later argument":   "both(view, view[1..]);",
		"bounds mutation":  "let ref mut i64[] child = view[0..{ edit(view); 2 }]; read(child);",
		"held target":      "let ref mut i64[] child = view[1..]; child[0] = view[0];",
		"returned result":  "let ref mut i64[] child = tail(view); edit(view); read(child);",
		"tagged result":    "let Option<ref mut i64[]> child = tagged(view); edit(view); switch (child) { case .some(value): { read(value); } case .none: {} };",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": prelude + "fn i32 main() { var i64[4] data = [1,2,3,4]; let ref mut i64[] view = ref mut data; " + body + " return 0; }"})
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

func TestAnalyzeI64MutableViewEscape(t *testing.T) {
	result := analyzeText(t, map[string]string{"main.fy": "module main; fn ref mut i64[] escape() { var i64[2] data = [1,2]; let ref mut i64[] view = ref mut data; return view[1..]; } fn i32 main() { return 0; }"})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2102" || len(result.Unsupported) != 0 {
		t.Fatalf("local escape: %+v %+v", result.Diagnostics, result.Unsupported)
	}
}

func TestAnalyzeI64MutableViewRejectedForms(t *testing.T) {
	for name, source := range map[string]string{
		"immutable owner":  "let i64[2] data = [1,2]; let ref mut i64[] view = ref mut data;",
		"shared promotion": "let i64[2] data = [1,2]; let ref i64[] shared = ref data; let ref mut i64[] edit = shared;",
		"wrong element":    "var u8[2] data = [1,2]; let ref mut i64[] edit = ref mut data;",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main; fn i32 main() { " + source + " return 0; }"})
			if len(result.Diagnostics) == 0 && len(result.Unsupported) == 0 {
				t.Fatal("unsafe mutable view accepted")
			}
		})
	}
}
