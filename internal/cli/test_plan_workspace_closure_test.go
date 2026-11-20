package cli

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestExperimentalTestPlanClosuresAcrossPackages(t *testing.T) {
	root := inlineCLIWorkspace(t)
	writeWorkspaceFile(t, root, "src/main.fy", `module main;
pub fn fn i32(i32) make_second() {
    let fn i32(i32) action = fn [] i32(i32 x) { return 44 - x; };
    return action;
}
`)
	writeWorkspaceFile(t, root, "tests/inline.fy", `module tests::inline;
import answer as a;
import main as m;
@test fn void calls_first() { assert(a::forward(a::make_first()) == 42); }
@test fn void calls_second() { assert(a::forward(m::make_second()) == 42); }
`)
	const provider = "libs/answer/src/answer.fy"
	const source = `module answer;
pub fn fn i32(i32) make_first() {
    let fn i32(i32) action = fn [] i32(i32 x) { return 40 + x; };
    return action;
}
fn i32 apply(fn i32(i32) action) { return action(2); }
pub fn i32 forward(fn i32(i32) action) { return apply(action); }
`
	for _, tc := range []struct {
		name, uid, selector, test, target, extra string
	}{
		{"dependency target", "test.inline-provider", "answer::_fy_closure_make_first_1", "calls_first", "test.inline-provider::answer::_fy_closure_make_first_1", ""},
		{"root target", "test.inline-cli", "main::_fy_closure_make_second_1", "calls_second", "main::_fy_closure_make_second_1", ""},
		{"unknown caller", "test.inline-provider", "answer::_fy_closure_make_first_1", "calls_first", "test.inline-provider::answer::_fy_closure_make_first_1",
			"fn i32 unknown(fn i32(i32) action) { return action(1); }"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writeWorkspaceFile(t, root, provider, source+tc.extra)
			test := "tests::inline::" + tc.test
			want := []string{test, "test.inline-provider::answer::forward", "test.inline-provider::answer::apply", tc.target}
			code, out, stderr := runForTest([]string{
				"experimental", "test-plan", "--symbol-package", tc.uid, "--symbol", tc.selector,
			}, root)
			var plan testPlanReport
			if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &plan) != nil ||
				len(plan.Tests) != 1 || plan.Tests[0].Test != test || !slices.Equal(plan.Tests[0].Path, want) {
				t.Fatalf("want test path %v, got (%d, %s, %s)", want, code, out, stderr)
			}
			if tc.extra == "" {
				if plan.Recommend != "selected-tests" || len(plan.Reasons) != 0 {
					t.Fatalf("want selected-tests with complete closure coverage, got %s", out)
				}
			} else if plan.Recommend != "all-tests" || !slices.Contains(plan.Reasons, "incomplete-call-coverage") {
				t.Fatalf("want all-tests for an unresolved closure call, got %s", out)
			}
		})
	}
}
