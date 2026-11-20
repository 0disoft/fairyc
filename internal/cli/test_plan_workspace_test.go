package cli

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestExperimentalTestPlanAcrossPackages(t *testing.T) {
	root := inlineCLIWorkspace(t)
	const provider = "libs/answer/src/answer.fy"
	const source = `module answer;
fn T target<T>(T value) { return value; }
fn i32 cycle(i32 n) { return relay(n); }
pub fn i32 relay(i32 n) {
    if (n == 0) { return target<i32>(42); }
    return cycle(n - 1);
}
pub fn bool other() { return target<bool>(true); }
`
	writeWorkspaceFile(t, root, provider, source)
	writeWorkspaceFile(t, root, "src/main.fy", `module main; fn i32 main() { return 0; }`)
	writeWorkspaceFile(t, root, "tests/inline.fy", `module tests::inline;
import answer as a;
fn i32 bridge() { return a::relay(2); }
@test fn void calls_target() { assert(bridge() == 42); }
@test fn void calls_other() { assert(a::other()); }
`)
	writeWorkspaceFile(t, root, "libs/answer/tests/answer.fy", `module tests::answer;
import answer as a;
@test fn void provider_test() { assert(a::relay(0) == 42); }
`)
	const test = "tests::inline::calls_target"
	path := []string{test, "tests::inline::bridge", "test.inline-provider::answer::relay", "test.inline-provider::answer::target<i32>"}
	for _, tc := range []struct {
		name, extra, recommendation string
	}{
		{"exact", "", "selected-tests"},
		{"opaque", "unsafe extern C fn i32 foreign(); fn i32 opaque() { return unsafe { foreign() }; }", "all-tests"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writeWorkspaceFile(t, root, provider, source+tc.extra)
			code, out, stderr := runForTest([]string{
				"experimental", "test-plan", "--symbol-package", "test.inline-provider", "--symbol", "answer::target<i32>",
			}, root)
			var plan testPlanReport
			if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &plan) != nil ||
				plan.Package != "test.inline-cli" || plan.SymbolPackage != "test.inline-provider" || plan.Recommend != tc.recommendation ||
				len(plan.Tests) != 1 || plan.Tests[0].Test != test || !slices.Equal(plan.Tests[0].Path, path) {
				t.Fatalf("want %s and path %v, got (%d, %s, %s)", tc.recommendation, path, code, out, stderr)
			}
			if tc.extra != "" && !slices.Contains(plan.Reasons, "external-ffi") {
				t.Fatalf("foreign coverage reason lost: %s", out)
			}
		})
	}
}
