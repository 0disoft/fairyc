package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestWindowsVectorStableCLI(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.stable-vector"
name = "client"
version = "1.0.0"
language = "fairyc-language/16"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", nativeVectorClient)
	code, stdout, stderr := runForTest([]string{"check"}, root)
	if code != ExitSuccess || stderr != "" || !strings.Contains(stdout, "check ok:") {
		t.Fatalf("check: %d %s %s", code, stdout, stderr)
	}
	for _, release := range []bool{false, true} {
		for _, command := range []string{"build", "run"} {
			args := []string{command, "--locked"}
			if release {
				args = append(args, "--release")
			}
			code, stdout, stderr := runForTest(args, root)
			if stderr != "" {
				t.Fatalf("%s: %s", command, stderr)
			}
			if command == "run" {
				if code != 41 || stdout != "41" {
					t.Fatalf("run: %d %q", code, stdout)
				}
			} else {
				artifact, err := os.ReadFile(filepath.Join(root, "build", "client.exe"))
				if code != ExitSuccess || !strings.Contains(stdout, "built: build/client.exe") || err != nil || len(artifact) == 0 {
					t.Fatalf("build: %d %s %v", code, stdout, err)
				}
			}
		}
	}
	testSource := strings.NewReplacer("module main;", "module tests::vector;", "@effects(console)", "@test\n@effects(console)", "fn i32 main()", "fn void works()", "    return 41;", "").Replace(nativeVectorClient)
	writeTestSource(t, root, "tests/vector.fy", testSource)
	code, _, stderr = runForTest([]string{"test", "--locked"}, root)
	if code != ExitInvalidInvocation || !strings.Contains(stderr, "require --native") {
		t.Fatalf("interpreter boundary: %d %s", code, stderr)
	}
	for _, release := range []bool{false, true} {
		args := []string{"test", "--native", "--locked", "--json", "--capture-output"}
		if release {
			args = append(args, "--release")
		}
		code, stdout, stderr = runForTest(args, root)
		var report stableTestExecutionReport
		if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(stdout), &report) != nil || report.Count != 1 || report.Passed != 1 || report.Tests[0].StdoutBase64 == nil || *report.Tests[0].StdoutBase64 != "NDE=" {
			t.Fatalf("native test: %d %s %s", code, stdout, stderr)
		}
	}
	writeTestSource(t, root, "src/main.fy", nativeVectorClient+"\nstruct Local<T> { T value; }")
	code, _, stderr = runForTest([]string{"check"}, root)
	if code != ExitSourceDiagnostics || !strings.Contains(stderr, "E4501") {
		t.Fatalf("source admission: %d %s", code, stderr)
	}
}

func TestWindowsVectorQuickstart(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	for _, file := range []string{"fairy.toml", "src/main.fy", "tests/vector.fy"} {
		contents, err := os.ReadFile(filepath.Join("..", "..", "examples", "std_vec_quickstart", filepath.FromSlash(file)))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, file, string(contents))
	}
	code, stdout, stderr := runForTest([]string{"check"}, root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("quickstart check: %d %s %s", code, stdout, stderr)
	}
	for _, release := range []bool{false, true} {
		args := []string{"run", "--locked"}
		if release {
			args = append(args, "--release")
		}
		code, stdout, stderr = runForTest(args, root)
		if code != ExitSuccess || stdout != "42" || stderr != "" {
			t.Fatalf("quickstart run: %d %s %s", code, stdout, stderr)
		}
		args = []string{"test", "--native", "--locked"}
		if release {
			args = append(args, "--release")
		}
		code, stdout, stderr = runForTest(args, root)
		if code != ExitSuccess || stderr != "" || !strings.Contains(stdout, "1 passed; 0 failed") {
			t.Fatalf("quickstart test: %d %s %s", code, stdout, stderr)
		}
	}
}

func TestWindowsVectorCandidateNativeTests(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.vector-tests"
name = "client"
version = "1.0.0"
language = "fairyc-language/16"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main() { return 0; }")
	client := strings.NewReplacer("module main;", "module tests::vector;", "@effects(console)", "@test\n@effects(console)", "fn i32 main()", "fn void b_passes()", "    return 41;", "").Replace(nativeVectorClient)
	client += `
@test
fn void a_traps() {
    var v::Vec<bytes::Buffer> items = v::empty<bytes::Buffer>();
    assert(bytes::len(v::at<bytes::Buffer>(ref items, 0)) == 0);
}
`
	writeTestSource(t, root, "tests/vector.fy", client)
	for _, release := range []bool{false, true} {
		workspace, err := workspaceloader.LoadVectorCandidate(root)
		if err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer
		options := stableTestOptions{native: true, release: release, json: true, locations: true, captureOutput: true, timeoutSeconds: 10}
		code := runLoadedStableTest(workspace, "verified", options, root, strings.NewReader(""), &stdout, &stderr)
		if code != ExitProgramTrap || stderr.Len() != 0 {
			t.Fatalf("native tests: %d %s %s", code, &stdout, &stderr)
		}
		var report stableTestExecutionReport
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		if report.Count != 2 || report.Passed != 1 || report.Failed != 1 || report.Engine != "native" || report.Lock != "verified" {
			t.Fatalf("report: %+v", report)
		}
		failed, passed := report.Tests[0], report.Tests[1]
		if failed.TrapCategory == nil || *failed.TrapCategory != "assertion-failed" || failed.Outcome != "failed" || passed.Outcome != "passed" || passed.StdoutBase64 == nil || *passed.StdoutBase64 != "NDE=" {
			t.Fatalf("outcomes: %+v", report.Tests)
		}
		if passed.Source == nil || passed.Source.Path != "tests/vector.fy" || failed.StdoutBase64 == nil || *failed.StdoutBase64 != "" {
			t.Fatalf("source/streams: %+v", report.Tests)
		}
	}
}

const nativeVectorClient = `module main;
import std::vec as v;
import std::bytes as bytes;
import std::io as io;
@effects(console)
fn i32 main() {
    var v::Vec<bytes::Buffer> items = v::empty<bytes::Buffer>();
    assert(switch (v::push<bytes::Buffer>(ref mut items, bytes::empty())) { case .ok(n) : { n == 1 } case .err(_) : { false } });
    assert(bytes::len(v::at<bytes::Buffer>(ref items, 0)) == 0);
    assert(switch (io::write_i32(41)) { case .ok(n) : { n == 2 } case .err(_) : { false } });
    return 41;
}
`

func TestWindowsVectorSupportSource(t *testing.T) {
	for _, receipt := range []bool{false, true} {
		got, err := windowsVectorSupportSource(nil, receipt)
		var want string
		if receipt {
			want, _ = windowsTargetRunSupportSource(nil)
		} else {
			want, _ = windowsTargetSupportSource(nil)
		}
		if err != nil || got != want {
			t.Fatalf("ordinary support changed: %v", err)
		}
		if _, err := windowsVectorSupportSource([]string{"fy_private_vector_unknown_v0"}, receipt); err == nil {
			t.Fatal("unknown runtime accepted")
		}
	}
}

func TestWindowsVectorCandidateNativeBuild(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	for _, trap := range []bool{false, true} {
		root := t.TempDir()
		writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.vector"
name = "client"
version = "1.0.0"
language = "fairyc-language/16"
source = "src"
`)
		client := nativeVectorClient
		if trap {
			client = strings.Replace(client, "ref items, 0", "ref items, 1", 1)
		}
		writeTestSource(t, root, "src/main.fy", client)
		workspace, err := workspaceloader.LoadVectorCandidate(root)
		if err != nil {
			t.Fatal(err)
		}
		var stderr bytes.Buffer
		lowered, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "test", &stderr)
		if code != ExitSuccess || !lowered.sdkStorage {
			t.Fatalf("lower: %d %s", code, &stderr)
		}
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			artifact, code := compileWindowsProgramPolicy(lowered.program, lowered.entryName, target.X8664WindowsMSVC, profile, true, lowered.sdkStorage, &stderr)
			if code != ExitSuccess {
				t.Fatalf("build %s: %d %s", profile, code, &stderr)
			}
			outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
			if err != nil {
				t.Fatal(err)
			}
			programStderr, category, err := classifyStableRunOutcome(outcome)
			if err != nil || programStderr != "" {
				t.Fatalf("run: %+v %v", outcome, err)
			}
			if trap {
				if category != "assertion-failed" || outcome.Stdout != "" {
					t.Fatalf("trap: %+v %s", outcome, category)
				}
			} else if category != "" || outcome.ExitCode != 41 || outcome.Stdout != "41" {
				t.Fatalf("run: %+v %s", outcome, category)
			}
		}
		if !trap {
			writeTestSource(t, root, "src/main.fy", nativeVectorClient+`
@link_name("fy_private_vector_release_empty_v0")
unsafe extern C fn u32 forged(ptr mut void value);
`)
			forged, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			stderr.Reset()
			_, code := lowerLoadedWorkspaceExecutable(forged, "missing", workspaceCompilePolicy{}, "test", &stderr)
			if code == ExitSuccess || !strings.Contains(stderr.String(), "reserved for the embedded SDK") {
				t.Fatalf("consumer runtime declaration: %d %s", code, &stderr)
			}
		}
	}
}
