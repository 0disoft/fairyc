package semantic_test

import "testing"

func TestArgumentLabelsSemantics(t *testing.T) {
	for _, test := range []struct{ name, declaration, expression, code string }{
		{"mixed", "fn i32 f(i32 first, i32 second) { return first + second; }", "f(.first = 20, 22)", ""},
		{"all", "fn i32 f(i32 first, i32 second) { return first + second; }", "f(.first = 20, .second = 22,)", ""},
		{"unknown", "fn i32 f(i32 first, i32 second) { return first + second; }", "f(.wrong = 20, 22)", "E1014"},
		{"reordered", "fn i32 f(i32 first, i32 second) { return first + second; }", "f(.second = 20, .first = 22)", "E1014"},
		{"duplicate", "fn i32 f(i32 first, i32 second) { return first + second; }", "f(.first = 20, .first = 22)", "E1014"},
		{"arity", "fn i32 f(i32 first, i32 second) { return first + second; }", "f(.first = 20)", "E1010"},
		{"public", "pub fn i32 f(i32 first) { return first; }", "f(.first = 42)", "E1014"},
		{"grouped target", "fn i32 f(i32 first) { return first; }", "(f)(.first = 42)", "E1014"},
		{"type", "fn i32 f(i32 first) { return first; }", "f(.first = true)", "E1006"},
		{"foreign", "unsafe extern C fn i32 f(i32 first);", "f(.first = 42)", "E1014"},
		{"owner function", "struct Box { i32 value; } fn i32 Box::f(i32 first) { return first; }", "Box::f(.first = 42)", "E1014"},
		{"intrinsic", "", "source_location(.value = 42)", "E1014"},
		{"constructor", "", ".some(.value = 42)", "E1014"},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main; " + test.declaration + " fn i32 main() { return " + test.expression + "; }"})
			if test.code == "" {
				assertCleanTypeResult(t, result)
				return
			}
			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Code == test.code {
					return
				}
			}
			t.Fatalf("missing %s: %+v", test.code, result)
		})
	}
}

func TestArgumentLabelsMethodAndModule(t *testing.T) {
	result := analyzeText(t, map[string]string{"main.fy": "module main; struct Box { i32 value; } fn i32 Box::f(ref Box this, i32 amount) { return this.value + amount; } fn i32 main() { let Box box = Box { .value = 1 }; return box.f(.amount = 41); }"})
	found := false
	for _, d := range result.Diagnostics {
		found = found || d.Code == "E1014"
	}
	if !found {
		t.Fatalf("method admitted: %+v", result)
	}
	result = analyzeText(t, map[string]string{"main.fy": "module main; import other as other; fn i32 main() { return other::f(.amount = 42); }", "other.fy": "module other; pub fn i32 f(i32 amount) { return amount; }"})
	found = false
	for _, d := range result.Diagnostics {
		found = found || d.Code == "E1014"
	}
	if !found {
		t.Fatalf("module call admitted: %+v", result)
	}
}

func TestArgumentLabelsConstantAndMove(t *testing.T) {
	for _, label := range []string{"value", "wrong"} {
		result := analyzeText(t, map[string]string{"main.fy": "module main; @const_eval fn i32 twice(i32 value) { return value + value; } static_assert(twice(." + label + " = 21) == 42); fn i32 main() { return 0; }"})
		if label == "value" {
			assertCleanTypeResult(t, result)
		} else if len(result.Diagnostics) == 0 {
			t.Fatal("const evaluator ignored label")
		}
	}
	for _, moved := range []bool{false, true} {
		tail := "return take(.value = item);"
		if moved {
			tail = "discard take(.value = item); return take(.value = item);"
		}
		result := analyzeText(t, map[string]string{"main.fy": "module main; resource Item { i32 number; cleanup {} } fn i32 take(Item value) { return value.number; } fn i32 main() { let Item item = Item { .number = 42 }; " + tail + " }"})
		if !moved {
			assertCleanTypeResult(t, result)
		} else {
			found := false
			for _, d := range result.Diagnostics {
				found = found || d.Code == "E1101"
			}
			if !found {
				t.Fatalf("missing move rejection: %+v", result)
			}
		}
	}
}
