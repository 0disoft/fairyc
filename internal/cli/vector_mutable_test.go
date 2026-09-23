package cli

import (
	"runtime"
	"strings"
	"testing"
)

func TestWindowsVectorMutableElement(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.vector-mutable"
name = "vector-mutable"
version = "1.0.0"
language = "fairyc-language/19"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", vectorMutableSource)
	for _, release := range []bool{false, true} {
		args := []string{"run"}
		if release {
			args = append(args, "--release", "--locked")
		}
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stdout != "" || stderr != "" {
			t.Fatalf("mutable vector release=%v: %d %q %s", release, code, stdout, stderr)
		}
	}

	aliased := strings.Replace(vectorMutableSource,
		"selected.value = 43;",
		"let ref Cell alias = v::at<Cell>(ref values, 0); selected.value = 43; assert(alias.value == 41);", 1)
	writeTestSource(t, root, "src/main.fy", aliased)
	code, _, stderr := runForTest([]string{"check"}, root)
	if code != ExitSourceDiagnostics || !strings.Contains(stderr, "E2101") {
		t.Fatalf("live mutable element borrow must reject alias: %d %s", code, stderr)
	}

	moved := strings.Replace(vectorMutableSource, "selected.value = 43;", "let v::Vec<Cell> moved = values; selected.value = 43;", 1)
	writeTestSource(t, root, "src/main.fy", moved)
	code, _, stderr = runForTest([]string{"check"}, root)
	if code != ExitSourceDiagnostics || !strings.Contains(stderr, "E2105") {
		t.Fatalf("live mutable element borrow must reject owner move: %d %s", code, stderr)
	}

	outOfRange := strings.Replace(vectorMutableSource, "v::at_mut<Cell>(ref mut values, 0)", "v::at_mut<Cell>(ref mut values, 1)", 1)
	writeTestSource(t, root, "src/main.fy", outOfRange)
	code, _, stderr = runForTest([]string{"run"}, root)
	if code != ExitProgramTrap || !strings.Contains(stderr, "assertion-failed") {
		t.Fatalf("out-of-range mutable element borrow: %d %s", code, stderr)
	}

	writeTestSource(t, root, "src/main.fy", vectorMutableScalarSource)
	for _, release := range []bool{false, true} {
		args := []string{"run"}
		if release {
			args = append(args, "--release", "--locked")
		}
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stdout != "" || stderr != "" {
			t.Fatalf("mutable scalar vector release=%v: %d %q %s", release, code, stdout, stderr)
		}
	}

	aliasedScalar := strings.Replace(vectorMutableScalarSource,
		"*selected = *selected + 2;",
		"let ref i32 alias = v::at<i32>(ref values, 0); *selected = 43; assert(*alias == 41);", 1)
	writeTestSource(t, root, "src/main.fy", aliasedScalar)
	code, _, stderr = runForTest([]string{"check"}, root)
	if code != ExitSourceDiagnostics || !strings.Contains(stderr, "E2101") {
		t.Fatalf("live mutable scalar slot must reject alias: %d %s", code, stderr)
	}

	movedOnRHS := strings.Replace(vectorMutableScalarSource, "fn i32 main()",
		"fn i32 take(v::Vec<i32> items) { return 1; }\nfn i32 main()", 1)
	movedOnRHS = strings.Replace(movedOnRHS, "*selected = *selected + 2;", "*selected = take(values);", 1)
	writeTestSource(t, root, "src/main.fy", movedOnRHS)
	code, _, stderr = runForTest([]string{"check"}, root)
	if code != ExitSourceDiagnostics || !strings.Contains(stderr, "E2105") {
		t.Fatalf("scalar store RHS must reject owner move: %d %s", code, stderr)
	}
}

const vectorMutableSource = `module main;
import std::vec as v;
struct Cell { i32 value; }
fn i32 main() {
    var v::Vec<Cell> values = v::empty<Cell>();
    assert(switch (v::push<Cell>(ref mut values, Cell { .value = 41 })) { case .ok(n) : { n == 1 } case .err(_) : { false } });
    let ref mut Cell selected = v::at_mut<Cell>(ref mut values, 0);
    selected.value = 43;
    let ref Cell observed = v::at<Cell>(ref values, 0);
    assert(observed.value == 43);
    return 0;
}
`

const vectorMutableScalarSource = `module main;
import std::vec as v;
fn i32 main() {
    var v::Vec<i32> values = v::empty<i32>();
    assert(switch (v::push<i32>(ref mut values, 41)) { case .ok(n) : { n == 1 } case .err(_) : { false } });
    let ref mut i32 selected = v::at_mut<i32>(ref mut values, 0);
    *selected = *selected + 2;
    let ref i32 observed = v::at<i32>(ref values, 0);
    assert(*observed == 43);
    return 0;
}
`
