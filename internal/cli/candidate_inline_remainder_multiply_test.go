package cli

import (
	"runtime"
	"testing"
)

// Keep this matrix separate from the accepted division/addition cases so a
// focused replay does not repeat their unchanged native CLI observations.
var inlineRemainderMultiplyTraps = inlineTrapFixture{
	cases: []inlineTrapCase{
		{"remzero", "remainder(10,0)", "division-by-zero", 8},
		{"remsigned", "remainder(-2147483648,-1)", "signed-division-overflow", 8},
		{"multiply", "multiply(46341,46341)", "integer-overflow", 9},
		{"zero_remzero", "zero_remainder()", "division-by-zero", 10},
		{"zero_remsigned", "zero_signed_remainder()", "signed-division-overflow", 11},
		{"zero_multiply", "zero_multiply()", "integer-overflow", 12},
	},
	provider: inlineCLIProvider + `@inline pub fn i32 remainder(i32 a,i32 b){return a%b;}
@inline pub fn i32 multiply(i32 a,i32 b){return a*b;}
@inline pub fn i32 zero_remainder(){return 10%0;}
@inline pub fn i32 zero_signed_remainder(){return -2147483648%-1;}
@inline pub fn i32 zero_multiply(){return 46341*46341;}
@inline pub fn i32 zero_remainder_value(){return 85%43;}
@inline pub fn i32 zero_negative_remainder(){return -85%43;}
@inline pub fn i32 zero_product(){return 6*7;}
`,
	assertions: `assert(a::remainder(85,43)==42);assert(a::remainder(-85,43)==-42);
assert(a::multiply(6,7)==42);assert(a::multiply(-6,-7)==42);
assert(a::zero_remainder_value()==42);assert(a::zero_negative_remainder()==-42);assert(a::zero_product()==42);`,
}

func TestCandidateInlineRemainderMultiplyReference(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC inline trap reference; native Linux remains deferred")
	}
	inlineRemainderMultiplyTraps.reference(t)
}

func TestCandidateInlineRemainderMultiplyCLI(t *testing.T) {
	runCandidateInlineTrapCLI(t, inlineRemainderMultiplyTraps)
}
