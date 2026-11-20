package cli

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func TestExperimentalTestPlanCalls(t *testing.T) {
	const packageUID = "example.stable-test.app"
	const selectedTest = "tests::plan::checks_answer"
	for _, tc := range []struct {
		name      string
		library   string
		call      string
		otherCall string
		selector  string
		path      []string
		reason    string
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
			path: []string{selectedTest, "library::bridge", "library::Token::cleanup", "library::answer"},
		},
		{
			name: "nested-cleanup",
			library: `resource Token {
    i32 value;
    cleanup { discard answer(); }
}
resource Wrapper { Token token; cleanup {} }
pub fn i32 bridge() {
    let Option<Wrapper> wrapped = .some(Wrapper { .token = Token { .value = 42 } });
    discard wrapped;
    return 42;
}`,
			call: "library::bridge()",
			path: []string{selectedTest, "library::bridge", "library::Token::cleanup", "library::answer"},
		},
		{
			name: "cleanup-with-ffi",
			library: `unsafe extern C fn i32 host_value();
resource Token {
    i32 value;
    cleanup { discard answer(); discard unsafe { host_value() }; }
}
pub fn i32 bridge() {
    let Token token = Token { .value = 42 };
    return token.value;
}`,
			call:   "library::bridge()",
			path:   []string{selectedTest, "library::bridge", "library::Token::cleanup", "library::answer"},
			reason: "incomplete-call-coverage",
		},
		{
			name: "copied-closure",
			library: `pub fn i32 bridge() {
    let i32 offset = 40;
    let fn i32(i32) action = fn [i32 offset] i32(i32 x) { return offset + x; };
    let copy = action;
    return copy(2);
}`,
			call:     "library::bridge()",
			selector: "library::_fy_closure_bridge_1",
			path:     []string{selectedTest, "library::bridge", "library::_fy_closure_bridge_1"},
		},
		{
			name: "closure-field",
			library: `struct Box { i32 arg; fn i32(i32) action; }
pub fn i32 bridge() {
    let i32 offset = 40;
    let fn i32(i32) action = fn [i32 offset] i32(i32 x) { return offset + x; };
    let Box stored = Box { .action = action, .arg = 2 };
    let Box copy = stored;
    let fn i32(i32) extracted = copy.action;
    return extracted(copy.arg);
}`,
			call:     "library::bridge()",
			selector: "library::_fy_closure_bridge_1",
			path:     []string{selectedTest, "library::bridge", "library::_fy_closure_bridge_1"},
		},
		{
			name: "closure-parameter",
			library: `fn i32 apply(fn i32(i32) action) { return action(2); }
pub fn i32 bridge() {
    let i32 offset = 40;
    let fn i32(i32) action = fn [i32 offset] i32(i32 x) { return offset + x; };
    return apply(action);
}`,
			call:     "library::bridge()",
			selector: "library::_fy_closure_bridge_1",
			path:     []string{selectedTest, "library::bridge", "library::apply", "library::_fy_closure_bridge_1"},
		},
		{
			name: "closure-return",
			library: `fn fn i32(i32) make(i32 offset) {
    let fn i32(i32) action = fn [i32 offset] i32(i32 x) { return offset + x; };
    return action;
}
fn fn i32(i32) relay(fn i32(i32) action) { return action; }
pub fn i32 bridge() {
    let action = relay(make(40));
    return action(2);
}`,
			call:     "library::bridge()",
			selector: "library::_fy_closure_make_1",
			path:     []string{selectedTest, "library::bridge", "library::_fy_closure_make_1"},
		},
		{
			name: "closure-mixed-same-caller",
			library: `fn i32 apply(fn i32(i32) action) { return action(2); }
pub fn i32 bridge() {
    let i32 offset = 40;
    let fn i32(i32) action = fn [i32 offset] i32(i32 x) { return offset + x; };
    let copy = action;
    let fn i32(i32) other = fn [] i32(i32 x) { return 40 + x; };
    return copy(2) + apply(copy) + apply(other) - 84;
}`,
			call:     "library::bridge()",
			selector: "library::_fy_closure_bridge_1",
			path:     []string{selectedTest, "library::bridge", "library::_fy_closure_bridge_1"},
		},
		{
			name: "closure-distinct-callers",
			library: `fn i32 apply(fn i32(i32) action) { return action(2); }
fn i32 relay(fn i32(i32) action) { return apply(action); }
pub fn i32 bridge() {
    let fn i32(i32) action = fn [] i32(i32 x) { return 40 + x; };
    return relay(action);
}
pub fn i32 alternate() {
    let fn i32(i32) action = fn [] i32(i32 x) { return 44 - x; };
    return relay(action);
}`,
			call:      "library::bridge()",
			otherCall: "library::alternate() == 42",
			selector:  "library::_fy_closure_bridge_1",
			path:      []string{selectedTest, "library::bridge", "library::relay", "library::apply", "library::_fy_closure_bridge_1"},
		},
		{
			name: "closure-ambiguous-return",
			library: `fn fn i32(i32) choose(bool first, fn i32(i32) left, fn i32(i32) right) {
    if (first) { return left; }
    return right;
}
pub fn i32 bridge(bool first) {
    let fn i32(i32) left = fn [] i32(i32 x) { return 40 + x; };
    let fn i32(i32) right = fn [] i32(i32 x) { return 44 - x; };
    let action = choose(first, left, right);
    return left(2) + action(2) - 42;
}`,
			call:     "library::bridge(true)",
			selector: "library::_fy_closure_bridge_1",
			path:     []string{selectedTest, "library::bridge", "library::_fy_closure_bridge_1"},
			reason:   "incomplete-call-coverage",
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
			call:   "library::left(2) + library::left(0) - 42",
			path:   []string{selectedTest, "library::left", "library::answer"},
			reason: "external-ffi",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeStableTestWorkspace(t, root)
			writeWorkspaceFile(t, root, "src/library.fy",
				"module library;\npub fn i32 answer() { return 42; }\n"+tc.library+"\n")
			otherCall := tc.otherCall
			if otherCall == "" {
				otherCall = "true"
			}
			writeWorkspaceFile(t, root, "tests/plan.fy", "module tests::plan;\nimport library as library;\n"+
				"@test\nfn void checks_answer() { assert("+tc.call+" == 42); }\n"+
				"@test\nfn void unrelated() { assert("+otherCall+"); }\n")
			selector := "library::answer"
			if tc.selector != "" {
				selector = tc.selector
			}
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
			if len(got.Tests) != 1 || got.Tests[0].Test != selectedTest || !slices.Equal(got.Tests[0].Path, tc.path) {
				t.Fatalf("unexpected plan: %s; want path %v", out, tc.path)
			}
			if tc.reason != "" {
				if got.Recommend != "all-tests" || !slices.Contains(got.Reasons, tc.reason) {
					t.Fatalf("expected full suite recommendation with %s: %s", tc.reason, out)
				}
			} else if got.Recommend != "selected-tests" || len(got.Reasons) != 0 {
				t.Fatalf("expected a selected-test recommendation: %s", out)
			}
		})
	}
}

func TestExperimentalTestPlanGenericSpecializations(t *testing.T) {
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "src/library.fy", `module library;
fn T target<T>(T value) { return value; }
fn T relay<T>(T value) { return target<T>(value); }
fn T unused<T>(T value) { return value; }
pub fn i32 signed() { return relay<i32>(42); }
pub fn bool boolean() { return relay<bool>(true); }
`)
	writeWorkspaceFile(t, root, "tests/plan.fy", `module tests::plan;
import library as library;
@test fn void checks_bool() { assert(library::boolean()); }
@test fn void checks_i32() { assert(library::signed() == 42); }
`)
	signed := atlasmap.TestPlanPath{Test: "tests::plan::checks_i32", Path: []string{"tests::plan::checks_i32", "library::signed", "library::relay<i32>", "library::target<i32>"}}
	boolean := atlasmap.TestPlanPath{Test: "tests::plan::checks_bool", Path: []string{"tests::plan::checks_bool", "library::boolean", "library::relay<bool>", "library::target<bool>"}}
	for _, tc := range []struct {
		selector string
		want     []atlasmap.TestPlanPath
		reason   string
	}{
		{selector: "library::target<i32>", want: []atlasmap.TestPlanPath{signed}},
		{selector: "library::target<bool>", want: []atlasmap.TestPlanPath{boolean}},
		{selector: "library::target", want: []atlasmap.TestPlanPath{boolean, signed}},
		{selector: "library::unused", reason: "generic-call-coverage"},
	} {
		t.Run(tc.selector, func(t *testing.T) {
			code, out, stderr := runForTest([]string{"experimental", "test-plan", "--symbol", tc.selector}, root)
			var got testPlanReport
			if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &got) != nil {
				t.Fatalf("test-plan = (%d, %s, %s)", code, out, stderr)
			}
			if !slices.EqualFunc(got.Tests, tc.want, func(a, b atlasmap.TestPlanPath) bool {
				return a.Test == b.Test && slices.Equal(a.Path, b.Path)
			}) {
				t.Fatalf("unexpected paths: %s, want %+v", out, tc.want)
			}
			if tc.reason == "" {
				if got.Recommend != "selected-tests" || len(got.Reasons) != 0 {
					t.Fatalf("expected selected tests: %s", out)
				}
			} else if got.Recommend != "all-tests" || !slices.Contains(got.Reasons, tc.reason) {
				t.Fatalf("expected full suite with %s: %s", tc.reason, out)
			}
		})
	}
}

func TestExperimentalTestPlanGenericCleanup(t *testing.T) {
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "src/library.fy", `module library;
fn T observe<T>(T value) { return value; }
resource Token<T> { T value; cleanup { discard observe<T>(this.value); } }
fn T exercise<T>(T value) {
    let Token<T> token = Token<T> { .value = value };
    return token.value;
}
pub fn i32 signed() { return exercise<i32>(42); }
pub fn bool boolean() { return exercise<bool>(true); }
`)
	writeWorkspaceFile(t, root, "tests/plan.fy", `module tests::plan;
import library as library;
@test fn void checks_bool() { assert(library::boolean()); }
@test fn void checks_i32() { assert(library::signed() == 42); }
`)
	for _, tc := range []struct{ typ, entry string }{{"i32", "signed"}, {"bool", "boolean"}} {
		t.Run(tc.typ, func(t *testing.T) {
			selector := "library::observe<" + tc.typ + ">"
			code, out, stderr := runForTest([]string{"experimental", "test-plan", "--symbol", selector}, root)
			var got testPlanReport
			if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &got) != nil {
				t.Fatalf("test-plan = (%d, %s, %s)", code, out, stderr)
			}
			test := "tests::plan::checks_" + tc.typ
			want := []string{test, "library::" + tc.entry, "library::exercise<" + tc.typ + ">", "library::Token<" + tc.typ + ">::cleanup", selector}
			if got.Recommend != "selected-tests" || len(got.Reasons) != 0 || len(got.Tests) != 1 || got.Tests[0].Test != test || !slices.Equal(got.Tests[0].Path, want) {
				t.Fatalf("want selected test with path %v, got %s", want, out)
			}
		})
	}
}
