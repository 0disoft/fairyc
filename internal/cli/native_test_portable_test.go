package cli

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStableNativePortableExecution(t *testing.T) {
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "windows" && runtime.GOOS != "linux") {
		t.Skip("requires Windows MSVC or Linux GNU amd64")
	}
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/native.fy", `module tests::native;
import std::vec as vec;
@test
fn void a_fails() { assert(false); }
@test
fn void z_passes() {
    var vec::Vec<i32> values = vec::empty<i32>();
    assert(switch (vec::push<i32>(ref mut values, 42)) { case .ok(n) : { n == 1 } case .err(_) : { false } });
    assert(*vec::at<i32>(ref values, 0) == 42);
}
`)
	writeWorkspaceFile(t, root, "tests/io/streams.fy", `module tests::io::streams;
import std::io as io;
import std::process as process;
@effects(console)
fn void streams() {
    assert(process::arg_count() == 4);
    var u8[8] flag = [0; 8];
    assert(switch (process::arg(1, ref mut flag)) { case .ok(n) : { n == 8 && flag[0] == 45 && flag[7] == 100 } case .err(_) : { false } });
    var u8[1] empty = [7];
    assert(switch (process::arg(2, ref mut empty)) { case .ok(n) : { n == 0 && empty[0] == 7 } case .err(_) : { false } });
    var u8[8] utf8 = [0; 8];
    assert(switch (process::arg(3, ref mut utf8)) { case .ok(n) : { n == 8 && utf8[0] == 237 && utf8[7] == 187 } case .err(_) : { false } });
    var u8[2] input = [9; 2];
    assert(switch (io::read_stdin(ref mut input)) { case .ok(n) : { n == 2 && input[0] == 65 && input[1] == 0 } case .err(_) : { false } });
    var u8[2] output = [0, 255];
    assert(switch (io::write_bytes(ref output)) { case .ok(n) : { n == 2 } case .err(_) : { false } });
    assert(switch (io::write_stderr("warn\n")) { case .ok(_) : { true } case .err(_) : { false } });
}
@test
@effects(console)
fn void io_a() { streams(); }
@test
@effects(console)
fn void io_b() { streams(); }
`)
	writeWorkspaceFile(t, root, "tests/timeout/hang.fy", "module tests::timeout::hang;\n@test\nfn void waits() { while (true) {} }\n")
	if err := os.WriteFile(filepath.Join(root, "input.bin"), []byte{'A', 0}, 0600); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		profile := "development"
		prefix := []string{"test", "--native", "--json", "--capture-output"}
		if release {
			profile = "release"
			prefix = append(prefix, "--release", "--locked")
		}
		var out, diag strings.Builder
		args := append(append([]string{}, prefix...), "--locations", "--file", "tests/native.fy")
		code := Run(args, root, &out, &diag)
		report := portableNativeReport(t, code, ExitProgramTrap, out.String(), diag.String(), profile, 2)
		if report.Passed != 1 || report.Failed != 1 || report.Tests[0].Identity != "tests::native::a_fails" || report.Tests[0].TrapCategory == nil || *report.Tests[0].TrapCategory != "assertion-failed" || report.Tests[1].Identity != "tests::native::z_passes" || report.Tests[1].Outcome != "passed" || report.Tests[0].Source == nil {
			t.Fatalf("trap continuation, order or locations: %+v", report)
		}
		for _, replay := range []bool{false, true} {
			out.Reset()
			diag.Reset()
			args = append([]string{}, prefix...)
			count := 1
			if replay {
				args = append(args, "--dir", "tests/io", "--stdin-file", "input.bin")
				count = 2
			} else {
				args = append(args, "--exact", "tests::io::streams::io_a", "--stdin")
			}
			args = append(args, "--", "--locked", "", "한글λ")
			code = RunWithInput(args, root, strings.NewReader("A\x00"), &out, &diag)
			report = portableNativeReport(t, code, 0, out.String(), diag.String(), profile, count)
			for _, result := range report.Tests {
				if result.Outcome != "passed" || result.StdoutBase64 == nil || *result.StdoutBase64 != "AP8=" || result.StderrBase64 == nil || *result.StderrBase64 != base64.StdEncoding.EncodeToString([]byte("warn\n")) {
					t.Fatalf("binary streams or input replay: %+v", result)
				}
			}
		}
	}
	var out, diag strings.Builder
	code := Run([]string{"test", "--native", "--locked", "--exact", "tests::timeout::hang::waits", "--timeout-seconds", "1"}, root, &out, &diag)
	if code != ExitBuildFailure || out.String() != "" || !strings.Contains(diag.String(), "exceeded 1s") {
		t.Fatalf("timeout: %d %q %q", code, out.String(), diag.String())
	}
	if _, err := os.Stat(filepath.Join(root, "build")); !os.IsNotExist(err) {
		t.Fatalf("native tests retained a build artifact: %v", err)
	}
}

func portableNativeReport(t *testing.T, code, wantCode int, stdout, stderr, profile string, count int) stableTestExecutionReport {
	t.Helper()
	if code != wantCode || stderr != "" {
		t.Fatalf("native CLI: %d %q %q", code, stdout, stderr)
	}
	var report stableTestExecutionReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != stableCapturedTestReportSchema || report.Engine != "native" || report.Target == nil || *report.Target != string(nativeWorkspaceTarget()) || report.Profile == nil || *report.Profile != profile || report.Count != count || len(report.Tests) != count {
		t.Fatalf("native report: %+v", report)
	}
	return report
}
