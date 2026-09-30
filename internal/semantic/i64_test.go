package semantic_test

import (
	"github.com/0disoft/fairyc/internal/hir"
	"testing"
)

func TestI64LiteralAndCompileTimeBoundaries(t *testing.T) {
	result := analyzeText(t, map[string]string{"main.fy": `module main;
const i64 MINIMUM = -9223372036854775808;
const i64 MAXIMUM = 0x7fff_ffff_ffff_ffff;
const i64 WIDE = 2147483647 + 1;
const i64 NEGATIVE = -5000000000;
const i64 SHIFT = -1 << 63;
static_assert(MINIMUM < NEGATIVE && MAXIMUM > WIDE);
fn i64 wide() { return MAXIMUM; }
fn i32 main() { return 0; }
`})
	assertCleanTypeResult(t, result)
	want := map[string]string{"MINIMUM": "-9223372036854775808", "MAXIMUM": "9223372036854775807", "WIDE": "2147483648", "NEGATIVE": "-5000000000", "SHIFT": "-9223372036854775808"}
	for _, constant := range result.Constants {
		if constant.Type != hir.TypeI64 || constant.Value != want[constant.Name] {
			t.Fatalf("constant = %+v", constant)
		}
		delete(want, constant.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing constants: %v", want)
	}
}

func TestI64RejectsOutOfRangeAndCompileTimeTraps(t *testing.T) {
	for _, body := range []string{
		"const i64 BAD = 9223372036854775808;",
		"const i64 BAD = -9223372036854775809;",
		"const i64 BAD = 9223372036854775807 + 1;",
		"const i64 BAD = -9223372036854775808 - 1;",
		"const i64 BAD = 3037000500 * 3037000500;",
		"const i64 BAD = -9223372036854775808 / -1;",
		"const i64 BAD = -9223372036854775808 % -1;",
		"const i64 BAD = 1 << 63;",
		"const i64 BAD = 1 >> 64;",
	} {
		t.Run(body, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main;\n" + body + "\nfn i32 main() { return 0; }"})
			if !hasDiagnosticCode(result, "E4303") {
				t.Fatalf("expected E4303, got %+v", result.Diagnostics)
			}
		})
	}
}
