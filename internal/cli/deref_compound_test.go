package cli

import (
	"os"
	"testing"
)

func TestDerefCompoundNative(t *testing.T) {
	src, e := os.ReadFile("../compiler/testdata/deref_compound.fy")
	if e != nil {
		t.Fatal(e)
	}
	cases := []struct{ name, source, trap string }{
		{"matrix", string(src), ""},
		{"generic", `module main;fn void add<T>(ref mut T item,T delta){*item+=delta;}fn i32 main(){var i64 x=4000000000;add<i64>(ref mut x,2);assert(x==4000000002);return 0;}`, ""},
		{"div-zero", `module main;fn i32 main(){var i64 x=10;let ref mut i64 r=ref mut x;*r/=0;return 0;}`, "division-by-zero"},
		{"mod-zero", `module main;fn i32 main(){var u8 x=10;let ref mut u8 r=ref mut x;*r%=0;return 0;}`, "division-by-zero"},
		{"signed-div-overflow", `module main;fn i32 main(){var i64 x=-9223372036854775808;let ref mut i64 r=ref mut x;*r/=-1;return 0;}`, "signed-division-overflow"},
		{"signed-mod-overflow", `module main;fn i32 main(){var i32 x=-2147483648;let ref mut i32 r=ref mut x;*r%=-1;return 0;}`, "signed-division-overflow"},
		{"invalid-shift", `module main;fn i32 main(){var u8 x=1;let ref mut u8 r=ref mut x;*r>>=8;return 0;}`, "invalid-shift"},
		{"left-shift-overflow", `module main;fn i32 main(){var u8 x=128;let ref mut u8 r=ref mut x;*r<<=1;return 0;}`, "integer-overflow"},
		{"rhs-trap", `module main;fn i32 main(){var i64 x=1;let ref mut i64 r=ref mut x;*r+={assert(false);1};return 0;}`, "assertion-failed"},
	}
	for _, tc := range []struct{ typ, max string }{{"u8", "255"}, {"i32", "2147483647"}, {"i64", "9223372036854775807"}, {"u32", "4294967295"}, {"usize", "18446744073709551615"}} {
		cases = append(cases, struct{ name, source, trap string }{tc.typ + "-overflow", `module main;fn i32 main(){var ` + tc.typ + ` x=` + tc.max + `;let ref mut ` + tc.typ + ` r=ref mut x;*r+=1;return 0;}`, "integer-overflow"})
	}
	runI64ViewNativeCases(t, "deref-compound", cases)
}
func TestDerefCompoundCache(t *testing.T) {
	src, e := os.ReadFile("../compiler/testdata/deref_compound.fy")
	if e != nil {
		t.Fatal(e)
	}
	checkStructVecViewCache(t, string(src))
}
