package cli

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

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
language = "fairyc-language/14"
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
