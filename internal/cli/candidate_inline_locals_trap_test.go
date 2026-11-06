package cli

import (
	"runtime"
	"testing"
)

// Both initializers are unused. Their first failure must still execute in
// declaration order, before the literal result, including after cache restore.
var inlineLocalsTraps = inlineTrapFixture{
	cases: []inlineTrapCase{
		{"initializer_division", "locals(2147483647,0)", "division-by-zero", 8},
		{"initializer_overflow", "locals(2147483647,1)", "integer-overflow", 8},
	},
	provider: inlineCLIProvider + `@inline pub fn i32 locals(i32 a,i32 b){let i32 unused=a/b;let i32 next=a+1;return 42;}
`,
	columns:    []int{55, 72},
	assertions: "assert(a::locals(40,2)==42);assert(a::add(40)==42);",
}

func TestCandidateInlineLocalsTrapReference(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows local initializer traps; Linux deferred")
	}
	inlineLocalsTraps.reference(t)
}
