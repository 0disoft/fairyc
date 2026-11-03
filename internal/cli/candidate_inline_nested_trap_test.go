package cli

import (
	"runtime"
	"testing"
)

var inlineNestedTraps = inlineTrapFixture{
	cases: []inlineTrapCase{
		{"inner", "nested(2147483647,0)", "integer-overflow", 8},
		{"division", "nested(9,0)", "division-by-zero", 8},
		{"outer", "nested(9,2)", "integer-overflow", 8},
		{"zero_inner", "zero_inner()", "integer-overflow", 9},
		{"zero_division", "zero_division()", "division-by-zero", 10},
		{"zero_outer", "zero_outer()", "integer-overflow", 11},
	},
	provider: inlineCLIProvider + `@inline pub fn i32 nested(i32 a,i32 b){return (a+1)/b+(2147483647+1);}
@inline pub fn i32 zero_inner(){return (2147483647+1)/0+(2147483647+1);}
@inline pub fn i32 zero_division(){return (9+1)/0+(2147483647+1);}
@inline pub fn i32 zero_outer(){return (9+1)/2+(2147483647+1);}
@inline pub fn i32 nested_value(i32 a,i32 b){return -(-((a+1)/b))+(39-2);}
@inline pub fn i32 zero_nested_value(){return -(-((9+1)/2))+(39-2);}
`,
	// One-based source columns, counted independently of either compiler's spans.
	columns:    []int{48, 47, 56, 41, 43, 49},
	assertions: "assert(a::nested_value(9,2)==42);assert(a::zero_nested_value()==42);",
}

func TestCandidateInlineNestedTrapReference(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC nested inline traps; native Linux remains deferred")
	}
	inlineNestedTraps.reference(t)
}

func TestCandidateInlineNestedTrapCLI(t *testing.T) {
	runCandidateInlineTrapCLI(t, inlineNestedTraps)
}
