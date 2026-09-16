package semantic_test

import (
	"strings"
	"testing"
)

func TestFieldShorthandSemanticParity(t *testing.T) {
	for _, test := range []struct {
		name, body string
		valid      bool
	}{
		{"copy", "let i32 value = 42; let Box box = Box { FIELD }; return box.value;", true},
		{"shadow", "let i32 value = 1; { let i32 value = 42; let Box box = Box { FIELD }; assert(box.value == 42); } return value;", true},
		{"unknown", "let Box box = Box { FIELD }; return 0;", false},
		{"wrong type", "let bool value = true; let Box box = Box { FIELD }; return 0;", false},
		{"duplicate", "let i32 value = 42; let Box box = Box { FIELD, .value = 0 }; return 0;", false},
		{"missing", "let i32 value = 42; let Wide box = Wide { FIELD }; return 0;", false},
		{"move", "let Leaf value = Leaf { .number = 42 }; let Owner box = Owner { FIELD }; return 0;", true},
		{"use after move", "let Leaf value = Leaf { .number = 42 }; let Owner box = Owner { FIELD }; let Leaf again = value; return 0;", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			prefix := "module main; struct Box { i32 value; } struct Wide { i32 value; i32 other; } resource Leaf { i32 number; cleanup {} } resource Owner { Leaf value; cleanup {} } fn i32 main() { "
			full := analyzeText(t, map[string]string{"main.fy": prefix + strings.ReplaceAll(test.body, "FIELD", ".value = value") + " }"})
			short := analyzeText(t, map[string]string{"main.fy": prefix + strings.ReplaceAll(test.body, "FIELD", ".value") + " }"})
			if test.valid {
				assertCleanTypeResult(t, full)
				assertCleanTypeResult(t, short)
				return
			}
			if len(full.Diagnostics)+len(full.Unsupported) == 0 || len(short.Diagnostics) != len(full.Diagnostics) || len(short.Unsupported) != len(full.Unsupported) {
				t.Fatalf("rejection differs: full=%+v short=%+v unsupported=%+v", full.Diagnostics, short.Diagnostics, short.Unsupported)
			}
			for i := range full.Diagnostics {
				if full.Diagnostics[i].Code != short.Diagnostics[i].Code {
					t.Fatalf("diagnostic changed: %+v / %+v", full.Diagnostics, short.Diagnostics)
				}
			}
			for i := range full.Unsupported {
				if full.Unsupported[i].Feature != short.Unsupported[i].Feature {
					t.Fatalf("unsupported rejection changed: %+v / %+v", full.Unsupported, short.Unsupported)
				}
			}
		})
	}
}
