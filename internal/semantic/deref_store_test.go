package semantic_test

import "testing"

func TestAnalyzeFieldStoreScalarDeref(t *testing.T) {
	for name, source := range map[string]string{
		"local": `module main; fn i32 main() {
    var i32 value = 1; let ref mut i32 view = ref mut value;
    *view = 42; assert(value == 42); return 0;
}`,
		"parameter": `module main;
fn void set(ref mut i32 view) { *view = 42; }
fn i32 main() { var i32 value = 1; set(ref mut value); return value; }`,
		"read through same reference": `module main; fn i32 main() {
    var i32 value = 1; let ref mut i32 view = ref mut value;
    *view = *view + 1; return value;
}`,
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": source})
			assertCleanTypeResult(t, result)
		})
	}
}

func TestAnalyzeFieldStoreScalarDerefRejections(t *testing.T) {
	const prelude = `module main; struct Record { i32 value; }
fn i32 main() { var i32 value = 1; var Record record = Record { .value = 1 }; `
	for name, test := range map[string]struct{ body, code string }{
		"shared":        {"let ref i32 view = ref value; *view = 42;", "E1008"},
		"mismatch":      {"let ref mut i32 view = ref mut value; *view = true;", "E1006"},
		"alias":         {"let ref mut i32 view = ref mut value; let ref i32 alias = ref value; *view = 42; assert(*alias == 1);", "E2101"},
		"owner read":    {"let ref mut i32 view = ref mut value; *view = value;", "E2101"},
		"rhs reborrow":  {"let ref mut i32 view = ref mut value; *view = *(ref value);", "E2101"},
		"compound bool": {"var bool flag = true; let ref mut bool view = ref mut flag; *view += true;", "unsupported"},
		"temporary":     {"*(ref mut value) = 42;", "unsupported"},
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": prelude + test.body + " return 0; }"})
			if test.code == "unsupported" {
				if len(result.Unsupported) == 0 {
					t.Fatalf("unsupported scalar dereference assignment admitted: %+v", result)
				}
				return
			}
			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Code == test.code {
					return
				}
			}
			t.Fatalf("missing %s: diagnostics=%+v unsupported=%+v", test.code, result.Diagnostics, result.Unsupported)
		})
	}
}
