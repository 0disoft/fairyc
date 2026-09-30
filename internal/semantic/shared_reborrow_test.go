package semantic_test

import "testing"

const sharedReborrowPrelude = `module main;
struct State { i32 number; }
fn i32 read(ref i32 value) { return *value; }
fn void edit(ref mut i32 value) { *value = 7; }
fn ref i32 retain(ref i32 value) { return value; }
fn Option<ref i32> optional(ref i32 value) { return .some(value); }
fn void both(ref i32 left, ref mut i32 right) {}
fn void reverse(ref mut i32 left, ref i32 right) {}
fn void shared(ref i32 left, ref i32 right) {}
`

func TestAnalyzeSharedReborrow(t *testing.T) {
	for name, body := range map[string]string{
		"call resumes":           "assert(read(view) == 1); edit(view);",
		"shared pair":            "shared(view, view); edit(view);",
		"group":                  "assert(read((view)) == 1); edit(view);",
		"return expires":         "let ref i32 child = retain(view); assert(*child == 1); edit(view);",
		"returned alias expires": "let ref i32 child = retain(view); let ref i32 alias = child; assert(*alias == 1); edit(view);",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": sharedReborrowPrelude + "fn i32 main() { var i32 owner = 1; let ref mut i32 view = ref mut owner; " + body + " return 0; }"})
			assertCleanTypeResult(t, result)
		})
	}
}

func TestAnalyzeSharedReborrowConflicts(t *testing.T) {
	for name, body := range map[string]string{
		"shared then mutable argument": "both(view, view);",
		"mutable then shared argument": "reverse(view, view);",
		"copied mutable argument":      "let ref mut i32 copy = view; both(view, copy);",
		"nested later mutation":        "shared(view, retain({ edit(view); view }));",
		"nested owner mutation":        "shared(view, retain({ owner = 2; view }));",
		"returned reference":           "let ref i32 child = retain(view); edit(view); assert(*child == 1);",
		"returned alias":               "let ref i32 child = retain(view); let ref i32 alias = child; edit(view); assert(*alias == 1);",
		"deref store":                  "let ref i32 child = retain(view); *view = 2; assert(*child == 1);",
		"owner store":                  "let ref i32 child = retain(view); owner = 2; assert(*child == 1);",
		"tagged result":                "let Option<ref i32> child = optional(view); edit(view); switch (child) { case .some(value): { assert(*value == 1); } case .none: {} };",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": sharedReborrowPrelude + "fn i32 main() { var i32 owner = 1; let ref mut i32 view = ref mut owner; " + body + " return 0; }"})
			found := false
			for _, d := range result.Diagnostics {
				if d.Code == "E2101" {
					found = true
				}
			}
			if !found || len(result.Unsupported) != 0 {
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

func TestAnalyzeSharedReborrowEscape(t *testing.T) {
	result := analyzeText(t, map[string]string{"main.fy": sharedReborrowPrelude + "fn ref i32 escape() { var i32 owner = 1; let ref mut i32 view = ref mut owner; return retain(view); }"})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2102" || len(result.Unsupported) != 0 {
		t.Fatalf("local escape: %+v %+v", result.Diagnostics, result.Unsupported)
	}
	result = analyzeText(t, map[string]string{"main.fy": sharedReborrowPrelude + "fn ref i32 forward(ref mut i32 value) { return retain(value); }"})
	assertCleanTypeResult(t, result)
}

func TestAnalyzeSharedReborrowReferentWrites(t *testing.T) {
	for name, source := range map[string]string{
		"parameter": sharedReborrowPrelude + "fn void check(ref mut i32 view) { let ref i32 child = retain(view); *view = 2; read(child); }",
		"byte view": `module main; fn ref u8[] retain(ref u8[] view) { return view; }
fn i32 main() { var u8[1] data = [1]; let ref mut u8[] view = ref mut data;
let ref u8[] child = retain(view); view[0] = 2; assert(child[0] == 1); return 0; }`,
		"whole struct": sharedReborrowPrelude + `fn ref State retain_state(ref State view) { return view; }
fn i32 main() { var State state = State { .number = 1 }; let ref mut State view = ref mut state;
let ref State child = retain_state(view); view.number = 2; assert(child.number == 1); return 0; }`,
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": source})
			if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2101" || len(result.Unsupported) != 0 {
				t.Fatalf("want one referent write conflict: %+v %+v", result.Diagnostics, result.Unsupported)
			}
		})
	}
}
