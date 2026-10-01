package semantic_test

import "testing"

func TestAnalyzeRepeatedArray(t *testing.T) {
	for name, body := range map[string]string{
		"inferred":          "let values = [42; 3]; assert(values[2] == 42);",
		"scalar set":        "let bool[1] a = [true; 1]; let u8[2] b = [255; 2]; let i32[2] c = [-7; 2]; let i64[2] d = [2147483648; 2]; let u32[2] e = [4294967295; 2]; let usize[2] f = [8; 2];",
		"maximum":           "let u8[4096] data = [0; 4096]; assert(data[4095] == 0);",
		"literal bases":     "let u8[4] a = [1; 0x4]; let u8[4] b = [1; 0b100]; let u8[4096] c = [0; 4_096];",
		"computed constant": "let i32[4] a = [3; COUNT];",
		"effect once":       "var i32 count = 0; let i32[4] a = [{ count += 1; count }; 4]; assert(count == 1);",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main; const i32 COUNT = 2 + 2; fn i32 main() { " + body + " return 0; }"})
			assertCleanTypeResult(t, result)
		})
	}
}

func TestAnalyzeRepeatedArrayRejectsUnsupported(t *testing.T) {
	for name, body := range map[string]string{
		"zero":              "let values = [1; 0];",
		"too many":          "let values = [1; 4097];",
		"runtime count":     "let usize count = 2; let values = [1; count];",
		"negative constant": "let values = [1; NEGATIVE];",
		"boolean constant":  "let values = [1; FLAG];",
		"resource":          "let values = [Owner { .value = 1 }; 1];",
		"nested array":      "let values = [[1; 2]; 2];",
		"borrow":            "var i32 owner = 1; let values = [ref mut owner; 2];",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main; resource Owner { i32 value; cleanup {} } const i32 NEGATIVE = -1; const bool FLAG = true; fn i32 main() { " + body + " return 0; }"})
			if len(result.Unsupported) == 0 || len(result.Diagnostics) != 0 {
				t.Fatalf("want unsupported without unrelated diagnostics: %+v %+v", result.Diagnostics, result.Unsupported)
			}
		})
	}
}

func TestAnalyzeRepeatedArrayTypeMismatch(t *testing.T) {
	for _, body := range []string{"let i32[3] values = [1; 2];", "let u8[2] values = [true; 2];", "let u8[2] values = [256; 2];"} {
		result := analyzeText(t, map[string]string{"main.fy": "module main; fn i32 main() { " + body + " return 0; }"})
		if len(result.Diagnostics) != 1 || len(result.Unsupported) != 0 {
			t.Fatalf("want one type/range diagnostic: %+v %+v", result.Diagnostics, result.Unsupported)
		}
		if code := result.Diagnostics[0].Code; code != "E1006" && code != "E1013" {
			t.Fatalf("wrong diagnostic: %s", code)
		}
	}
}
