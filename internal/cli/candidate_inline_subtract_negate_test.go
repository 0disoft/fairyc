package cli

import (
	"runtime"
	"testing"
)

var inlineSubtractNegateTraps = inlineTrapFixture{
	cases: []inlineTrapCase{
		{"subtract_low", "subtract(-2147483648,1)", "integer-overflow", 8},
		{"subtract_high", "subtract(2147483647,-1)", "integer-overflow", 8},
		{"negate", "negate(-2147483648)", "integer-overflow", 9},
		{"zero_subtract_low", "zero_subtract_low()", "integer-overflow", 10},
		{"zero_subtract_high", "zero_subtract_high()", "integer-overflow", 11},
		{"zero_negate", "zero_negate()", "integer-overflow", 12},
	},
	provider: inlineCLIProvider + `@inline pub fn i32 subtract(i32 a,i32 b){return a-b;}
@inline pub fn i32 negate(i32 a){return -a;}
@inline pub fn i32 zero_subtract_low(){return -2147483648-1;}
@inline pub fn i32 zero_subtract_high(){return 2147483647-(-1);}
@inline pub fn i32 zero_negate(){return -(-2147483648);}
@inline pub fn i32 zero_minimum(){return -2147483648-0;}
@inline pub fn i32 zero_maximum(){return 2147483647-0;}
@inline pub fn i32 zero_negation(){return -(-42);}
@inline pub fn i32 zero_max_negation(){return -(2147483647);}
@inline pub fn i32 zero_negate_zero(){return -0;}
`,
	assertions: `assert(a::subtract(-2147483648,0)==-2147483648);assert(a::subtract(2147483647,0)==2147483647);
assert(a::negate(-42)==42);assert(a::negate(2147483647)==-2147483647);assert(a::negate(0)==0);
assert(a::zero_minimum()==-2147483648);assert(a::zero_maximum()==2147483647);
assert(a::zero_negation()==42);assert(a::zero_max_negation()==-2147483647);assert(a::zero_negate_zero()==0);`,
}

func TestCandidateInlineSubtractNegateReference(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC inline trap reference; native Linux remains deferred")
	}
	inlineSubtractNegateTraps.reference(t)
}

func TestCandidateInlineSubtractNegateCLI(t *testing.T) {
	runCandidateInlineTrapCLI(t, inlineSubtractNegateTraps)
}
