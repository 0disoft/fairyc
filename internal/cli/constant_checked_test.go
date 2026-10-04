package cli

import (
	"os"
	"testing"
)

func TestConstantCheckedTrapNative(t *testing.T) {
	data, err := os.ReadFile("../compiler/testdata/constant_checked_trap.fy")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, source, trap string }{
		{"aggregate-rhs", string(data), "integer-overflow"},
		{"aggregate-return", `module main;struct Span{i64 start;i64 end;}fn Span bad(){var i64 n=9223372036854775807;n+=1;return Span{.start=n,.end=0};}fn i32 main(){let Span s=bad();return 0;}`, "integer-overflow"},
		{"dynamic-safe", `module main;struct Span{i64 start;i64 end;}struct Owner{Span position;}fn Span next(i64 seed){var i64 n=seed;n+=1;return Span{.start=n,.end=0};}fn i32 main(){var Owner s=Owner{.position=Span{.start=0,.end=0}};s.position=next(41);assert(s.position.start==42);return 0;}`, ""},
		{"alias-rewrite", `module main;fn i32 main(){var i64 n=9223372036854775807;let ref mut i64 slot=ref mut n;*slot=40;n+=2;assert(n==42);return 0;}`, ""},
	}
	for _, item := range []struct{ name, typ, initial, operation string }{
		{"i64-sub", "i64", "-9223372036854775808", "-=1"},
		{"i64-mul", "i64", "9223372036854775807", "*=2"},
		{"i32-add", "i32", "2147483647", "+=1"},
		{"u8-add", "u8", "255", "+=1"},
		{"u32-sub", "u32", "0", "-=1"},
		{"usize-mul", "usize", "18446744073709551615", "*=2"},
	} {
		cases = append(cases, struct{ name, source, trap string }{item.name, "module main;fn i32 main(){var " + item.typ + " n=" + item.initial + ";n" + item.operation + ";return 0;}", "integer-overflow"})
	}
	runI64ViewNativeCases(t, "constant-checked-trap", cases)
}
