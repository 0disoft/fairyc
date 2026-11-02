package cli

import (
	"strings"
	"testing"
)

// Only the caller is generic. Both specializations call the same nongeneric
// inline exports, including four ordered side-effecting arguments.
func genericInlineCLIWorkspace(t *testing.T) string {
	t.Helper()
	root := inlineCLIWorkspace(t)
	caller := inlineCLIPick + "fn i32 evaluate<T>(T marker){discard marker;" + inlineCLIChecks + "return a::add(40);}"
	writeTestSource(t, root, "src/main.fy", "module main;import answer as a;"+caller+`
fn i32 main(){let i32 first=evaluate<i32>(1);let i32 second=evaluate<bool>(true);assert(first==second);return first;}`)
	writeTestSource(t, root, "tests/inline.fy", "module tests::inline;import answer as a;"+caller+`
@test fn void inline_contract(){assert(evaluate<i32>(1)==42);assert(evaluate<bool>(true)==42);}`)
	return root
}

func TestCandidateGenericInlineCLI(t *testing.T) {
	runCandidateInlineCLI(t, genericInlineCLIWorkspace)
}

func TestCandidateGenericInlineCLIReference(t *testing.T) {
	root := genericInlineCLIWorkspace(t)
	for _, value := range []int{42, 43} {
		if value == 43 {
			writeTestSource(t, root, "libs/answer/src/answer.fy", strings.Replace(inlineCLIProvider, "return x+2;", "return x+3;", 1))
		}
		code, out, diagnostic := runForTest([]string{"run"}, root)
		if code != value || out != "" || diagnostic != "" {
			t.Fatalf("Go reference: exit=%d want=%d stdout=%q stderr=%q", code, value, out, diagnostic)
		}
	}
}
