package semantic_test

import "testing"

func TestAnalyzeI64ViewLifetime(t *testing.T) {
	for name, body := range map[string]string{
		"owner write":       "let ref i64[] child = view[1..]; data[0] = 5; assert(child[0] == 2);",
		"owner replace":     "let ref i64[] alias = view; data = [5,6]; assert(alias[0] == 1);",
		"exclusive overlap": "let ref mut i64[2] edit = ref mut data; assert(view[0] == 1);",
		"returned shared":   "let ref i64[] child = tail(view); data[0] = 5; assert(child[0] == 2);",
		"bound mutation":    "let ref i64[] child = view[0..{ data[0] = 5; 1 }]; assert(child[0] == 1);",
		"index mutation":    "let i64 value = view[{ data[0] = 5; 0 }]; assert(value == 1);",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main; fn ref i64[] tail(ref i64[] view) { return view[1..]; } fn i32 main() { var i64[2] data = [1,2]; let ref i64[] view = ref data; " + body + " return 0; }"})
			found := false
			for _, diagnostic := range result.Diagnostics {
				found = found || diagnostic.Code == "E2101"
			}
			if !found {
				t.Fatalf("missing loan conflict: %+v %+v", result.Diagnostics, result.Unsupported)
			}
		})
	}
}

func TestAnalyzeI64ViewRejectedForms(t *testing.T) {
	for name, source := range map[string]string{
		"local escape":           "fn ref i64[] bad() { let i64[2] data = [1,2]; return ref data; }",
		"value parameter escape": "fn ref i64[] bad(i64[2] data) { return ref data; }",
		"owned slice":            "fn void bad(i64[] values) {}",
		"wrong element":          "fn void read(ref i64[] values) {} fn void bad() { let u8[2] data = [1,2]; read(ref data); }",
		"write through view":     "fn void bad(ref i64[] values) { values[0] = 3; }",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main; " + source + " fn i32 main() { return 0; }"})
			if len(result.Diagnostics) == 0 && len(result.Unsupported) == 0 {
				t.Fatal("unsupported or unsafe view accepted")
			}
		})
	}
}

func TestAnalyzeSharedByteViewHeldDuringBounds(t *testing.T) {
	for name, body := range map[string]string{
		"index": "let u8 value = view[{ data[0] = 5; 0 }];",
		"slice": "let ref u8[] child = view[..{ data[0] = 5; 1 }];",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main; fn i32 main() { var u8[2] data = [1,2]; let ref u8[] view = ref data; " + body + " return 0; }"})
			found := false
			for _, diagnostic := range result.Diagnostics {
				found = found || diagnostic.Code == "E2101"
			}
			if !found {
				t.Fatalf("lost selected byte receiver loan: %+v %+v", result.Diagnostics, result.Unsupported)
			}
		})
	}
}
