package cli

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestExperimentalTestPlanCalls(t *testing.T) {
	const packageUID = "example.stable-test.app"
	const selectedTest = "tests::plan::checks_answer"
	for _, tc := range []struct {
		name    string
		library string
		call    string
		path    []string
	}{
		{
			name: "direct",
			call: "library::answer()",
			path: []string{selectedTest, "library::answer"},
		},
		{
			name:    "indirect",
			library: "pub fn i32 bridge() { return answer(); }",
			call:    "library::bridge()",
			path:    []string{selectedTest, "library::bridge", "library::answer"},
		},
		{
			name: "cleanup",
			library: `resource Token {
    i32 value;
    cleanup { discard answer(); }
}
pub fn i32 bridge() {
    let Token token = Token { .value = 42 };
    return token.value;
}`,
			call: "library::bridge()",
		},
		{
			name: "cycle",
			library: `unsafe extern C fn i32 host_value();
pub fn i32 left(i32 n) {
    if (n == 0) { return answer(); }
    return right(n - 1);
}
fn i32 right(i32 n) {
    if (n < 0) { return unsafe { host_value() }; }
    return left(n);
}`,
			call: "library::left(2) + library::left(0) - 42",
			path: []string{selectedTest, "library::left", "library::answer"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeStableTestWorkspace(t, root)
			writeWorkspaceFile(t, root, "src/library.fy",
				"module library;\npub fn i32 answer() { return 42; }\n"+tc.library+"\n")
			writeWorkspaceFile(t, root, "tests/plan.fy", "module tests::plan;\nimport library as library;\n"+
				"@test\nfn void checks_answer() { assert("+tc.call+" == 42); }\n"+
				"@test\nfn void unrelated() { assert(true); }\n")
			selector := "library::answer"
			if tc.name == "direct" {
				selector = atlasFunctionSymbolForPackage(t, root, packageUID, "answer").Visible
			}
			args := []string{"experimental", "test-plan", "--symbol", selector, "--package", packageUID}
			code, out, stderr := runForTest(args, root)
			var got testPlanReport
			if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &got) != nil {
				t.Fatalf("test-plan = (%d, %s, %s)", code, out, stderr)
			}
			if got.Schema != "fairyc.test-plan/v1" || got.Package != packageUID || got.Symbol != selector {
				t.Fatalf("unexpected plan identity: %s", out)
			}
			if tc.name == "cleanup" {
				if len(got.Tests) != 0 || got.Recommend != "all-tests" || !slices.Contains(got.Reasons, "incomplete-call-coverage") {
					t.Fatalf("cleanup call did not request the full suite: %s", out)
				}
				return
			}
			if len(got.Tests) != 1 || got.Tests[0].Test != selectedTest || !slices.Equal(got.Tests[0].Path, tc.path) {
				t.Fatalf("unexpected plan: %s; want path %v", out, tc.path)
			}
			if tc.name == "cycle" {
				if got.Recommend != "all-tests" || !slices.Contains(got.Reasons, "external-ffi") {
					t.Fatalf("opaque branch in cycle did not request the full suite: %s", out)
				}
			} else if got.Recommend != "selected-tests" || len(got.Reasons) != 0 {
				t.Fatalf("expected a selected-test recommendation: %s", out)
			}
		})
	}
}
