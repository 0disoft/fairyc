package cli

import (
	"runtime"
	"strings"
	"testing"
)

func TestWindowsVectorTruncate(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.vector-truncate"
name = "truncate"
version = "1.0.0"
language = "fairyc-language/16"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", vectorTruncateSource)
	for _, release := range []bool{false, true} {
		args := []string{"run"}
		if release {
			args = append(args, "--release", "--locked")
		}
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stdout != "" || stderr != "" {
			t.Fatalf("truncate release=%v: %d %q %s", release, code, stdout, stderr)
		}
	}
	// Even a length-preserving truncate is a mutable operation: a live element
	// borrow must prevent it rather than relying on this call's numeric argument.
	invalid := strings.Replace(vectorTruncateSource, "assert(v::truncate<i32>(ref mut values, 2) == 2);", "let ref i32 borrowed = v::at<i32>(ref values, 0);\nassert(v::truncate<i32>(ref mut values, 4) == 0);\nassert(*borrowed == 10);", 1)
	writeTestSource(t, root, "src/main.fy", invalid)
	code, _, stderr := runForTest([]string{"check"}, root)
	if code != ExitSourceDiagnostics || !strings.Contains(stderr, "E2101") {
		t.Fatalf("live borrow must reject truncation: %d %s", code, stderr)
	}
}

const vectorTruncateSource = `module main;
import std::vec as v;
fn i32 main() {
    var v::Vec<i32> values = v::empty<i32>();
    assert(v::truncate<i32>(ref mut values, 1) == 0);
    assert(v::capacity<i32>(ref values) == 0);
    for (var i32 index = 10; index < 14; index += 1) {
        assert(switch (v::push<i32>(ref mut values, index)) { case .ok(_) : { true } case .err(_) : { false } });
    }
    let usize capacity = v::capacity<i32>(ref values);
    assert(v::truncate<i32>(ref mut values, 100) == 0);
    assert(v::truncate<i32>(ref mut values, 4) == 0);
    assert(v::truncate<i32>(ref mut values, 2) == 2);
    assert(v::len<i32>(ref values) == 2);
    assert(v::capacity<i32>(ref values) == capacity);
    assert(*v::at<i32>(ref values, 0) == 10 && *v::at<i32>(ref values, 1) == 11);
    assert(v::truncate<i32>(ref mut values, 2) == 0);
    assert(switch (v::push<i32>(ref mut values, 42)) { case .ok(n) : { n == 3 } case .err(_) : { false } });
    assert(*v::at<i32>(ref values, 2) == 42);
    assert(v::truncate<i32>(ref mut values, 0) == 3);
    assert(v::is_empty<i32>(ref values));
    assert(v::capacity<i32>(ref values) == capacity);
    assert(v::clear<i32>(ref mut values) == 0);
    return 0;
}
`
