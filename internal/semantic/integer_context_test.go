package semantic_test

import "testing"

func TestContextualIntegerOperands(t *testing.T) {
	for name, body := range map[string]string{
		"workload":        "let usize rows = 2; let usize size = 1048576 / rows;",
		"inferred peer":   "let usize rows = 2; let size = ((1048576)) / rows; assert(size == rows * 262144);",
		"literal subtree": "let u32 value = 2; let result = (2147483647 + 1) / value;",
		"destinations":    "let u8 byte = 200 + 1; let u32 word = 2147483647 + 1; let i64 wide = 9223372036854775806 + 1; let usize size = 18446744073709551615 / 1;",
		"comparisons":     "let usize size = 2; assert(1 < size); assert((1 + 1) == size); assert(3 != size); assert(2 <= size); assert(3 > size); assert(2 >= size);",
		"signed prefixes": "let i64 wide = 2; let i64 value = -5000000000 / wide; let minimum = -9223372036854775808 / wide;",
		"bitwise":         "let u32 word = 2; let u32 value = (~0 & 255) ^ word; let u32 shifted = 1 << word; let u32 back = 8 >> word;",
		"contexts":        "let u32[2] values = [2147483647 + 1; 2]; assert(values[1] == 2147483648); assert(take(2147483647 + 1) == 2147483648);",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main; fn u32 take(u32 value) { return value; } fn i32 main() { " + body + " return 0; }"})
			assertCleanTypeResult(t, result)
		})
	}
}

func TestContextualIntegerRejectsConversionsAndRanges(t *testing.T) {
	for name, item := range map[string]struct{ body, code string }{
		"mixed variables":             {"let u32 word = 1; let i32 signed = 1; let u32 value = signed + word;", "E1006"},
		"fixed constant":              {"let u32 word = 1; let u32 value = FIXED + word;", "E1006"},
		"peer beats destination":      {"let u8 byte = 1; let u32 value = 1 + byte;", "E1006"},
		"range at peer":               {"let u8 byte = 1; let value = 256 / byte;", "E1013"},
		"range at destination":        {"let u8 value = 256 + 1;", "E1013"},
		"comparison range":            {"let u8 byte = 1; assert(256 > byte);", "E1013"},
		"unconstrained default":       {"let value = 2147483648 + 1;", "E1013"},
		"bool is not integer context": {"assert(2147483648 == 2147483648);", "E1013"},
		"negative unsigned":           {"let u32 word = 1; let value = -1 / word;", "E1006"},
		"minimum group unchanged":     {"let i64 value = -(9223372036854775808) + 1;", "E1013"},
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main; const i32 FIXED = 1; fn i32 main() { " + item.body + " return 0; }"})
			if !hasDiagnosticCode(result, item.code) || len(result.Unsupported) != 0 {
				t.Fatalf("want %s: %+v %+v", item.code, result.Diagnostics, result.Unsupported)
			}
		})
	}
}

func TestContextualIntegerCompileTime(t *testing.T) {
	result := analyzeText(t, map[string]string{"main.fy": `module main;
const u32 WORD = 2;
const u32 QUOTIENT = (2147483647 + 1) / WORD;
const bool COMPARED = 1 < WORD;
const i64 WIDE = 2;
const i64 MINIMUM = -9223372036854775808 / WIDE;
static_assert(QUOTIENT == 1073741824 && COMPARED && MINIMUM == -4611686018427387904);
fn i32 main() { return 0; }
`})
	assertCleanTypeResult(t, result)
	for _, text := range []string{
		"const u32 WORD = 1; const u32 BAD = 4294967296 / WORD;",
		"const u32 WORD = 1; const i32 SIGNED = 1; const u32 BAD = SIGNED + WORD;",
		"const u8 BYTE = 255; const u8 BAD = 1 + BYTE;",
	} {
		result := analyzeText(t, map[string]string{"main.fy": "module main; " + text + " fn i32 main() { return 0; }"})
		if len(result.Diagnostics) == 0 || len(result.Unsupported) != 0 {
			t.Fatalf("want compile-time rejection: %+v %+v", result.Diagnostics, result.Unsupported)
		}
	}
}
