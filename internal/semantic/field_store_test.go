package semantic_test

import "testing"

func TestAnalyzeFieldStoreRejectsBorrowedRootMutation(t *testing.T) {
	for name, body := range map[string]string{
		"shared":                "let ref i32 view = ref state.count; state.count = 2; assert(*view == 1);",
		"other field same root": "let ref i32 view = ref state.count; state.other = 2; assert(*view == 1);",
		"exclusive":             "let ref mut State view = ref mut state; state.count = 2; consume(view);",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": `module main;
struct State { i32 count; i32 other; }
fn void consume(ref mut State view) {}
fn i32 main() { var State state = State { .count = 1, .other = 0 }; ` + body + ` return 0; }`})
			found := false
			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Code == "E2101" {
					found = true
				}
			}
			if !found {
				t.Fatalf("field store did not protect root: %+v", result.Diagnostics)
			}
		})
	}
}
