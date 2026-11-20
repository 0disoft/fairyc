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

func TestExperimentalTestPlanCleanupAcrossPackages(t *testing.T) {
	root := inlineCLIWorkspace(t)
	writeWorkspaceFile(t, root, "src/main.fy", `module main; fn i32 main() { return 0; }`)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", `module answer;
fn void observe_plain(i32 value) { discard value; }
fn void observe<T>(T value) { discard value; }
pub resource Plain { i32 value; cleanup { observe_plain(this.value); } }
pub resource Token<T> { T value; cleanup { observe<T>(this.value); } }
pub fn Plain plain() { return Plain { .value = 1 }; }
pub fn Token<i32> integer() { return Token<i32> { .value = 2 }; }
pub fn Token<bool> boolean() { return Token<bool> { .value = true }; }
`)
	writeWorkspaceFile(t, root, "tests/inline.fy", `module tests::inline;
import answer as a;
@test fn void drops_plain() { discard a::plain(); }
@test fn void drops_integer() { discard a::integer(); }
@test fn void drops_boolean() { discard a::boolean(); }
`)
	const prefix = "test.inline-provider::"
	for _, tc := range []struct {
		selector, test, cleanup, callee string
	}{
		{"answer::Plain::cleanup", "drops_plain", "answer::Plain::cleanup", ""},
		{"answer::observe_plain", "drops_plain", "answer::Plain::cleanup", "answer::observe_plain"},
		{"answer::Token<i32>::cleanup", "drops_integer", "answer::Token<i32>::cleanup", ""},
		{"answer::observe<i32>", "drops_integer", "answer::Token<i32>::cleanup", "answer::observe<i32>"},
		{"answer::observe<bool>", "drops_boolean", "answer::Token<bool>::cleanup", "answer::observe<bool>"},
	} {
		t.Run(tc.selector, func(t *testing.T) {
			test := "tests::inline::" + tc.test
			path := []string{test, prefix + tc.cleanup}
			if tc.callee != "" {
				path = append(path, prefix+tc.callee)
			}
			code, out, stderr := runForTest([]string{
				"experimental", "test-plan", "--symbol-package", "test.inline-provider", "--symbol", tc.selector,
			}, root)
			var plan testPlanReport
			if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &plan) != nil ||
				plan.Recommend != "selected-tests" || len(plan.Reasons) != 0 || len(plan.Tests) != 1 ||
				plan.Tests[0].Test != test || !slices.Equal(plan.Tests[0].Path, path) {
				t.Fatalf("want one selected test with path %v, got (%d, %s, %s)", path, code, out, stderr)
			}
		})
	}
}
