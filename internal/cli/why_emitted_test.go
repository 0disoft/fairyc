package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/target"
)

func TestWhyEmittedExplicitSourceSet(t *testing.T) {
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main; import helpers as h; fn i32 main() { return h::get(); }")
	writeTestSource(t, root, "helpers.fy", "module helpers; pub fn i32 get() { return 42; }")
	args := []string{"experimental", "why-emitted", "main.fy", "--source", "helpers.fy", "--symbol"}
	code, stdout, stderr := runForTest(append(args, "helpers::get"), root)
	var report cbackend.EmissionExplanation
	if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(stdout), &report) != nil ||
		report.Schema != "fairyc.why-emitted/v1" || !report.Emitted || len(report.Path) != 1 ||
		report.Path[0] != (cbackend.EmissionStep{From: "main::main", To: "helpers::get", Reason: "call"}) {
		t.Fatalf("explain: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if code, stdout, stderr := runForTest(append(args, "missing"), root); code != ExitInvalidInvocation ||
		stdout != "" || !strings.Contains(stderr, "expected one internal function") {
		t.Fatalf("unknown selector: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestWhyEmittedWorkspace(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	if _, _, err := loadWorkspaceWithLock(root, false); err != nil {
		t.Fatal(err)
	}
	before := remoteTree(t, root)
	args := []string{"experimental", "why-emitted", "--workspace", "--locked", "--target"}
	for _, id := range []target.ID{target.X8664LinuxGNU, target.X8664WindowsMSVC} {
		t.Run(string(id), func(t *testing.T) {
			code, out, diagnostics := runForTest(append(args, string(id), "--symbol", "value"), root)
			var report cbackend.EmissionExplanation
			if code != ExitSuccess || diagnostics != "" || json.Unmarshal([]byte(out), &report) != nil ||
				!report.Emitted || !strings.HasSuffix(report.Entry, "::main::main") ||
				!strings.HasSuffix(report.Symbol, "::answer::value") ||
				!slices.Equal(report.Path, []cbackend.EmissionStep{{From: report.Entry, To: report.Symbol, Reason: "call"}}) {
				t.Fatalf("explanation: %d %q %q", code, out, diagnostics)
			}
			code, full, diagnostics := runForTest(append(args, string(id), "--symbol", report.Symbol), root)
			if code != ExitSuccess || diagnostics != "" || full != out {
				t.Fatalf("exact selector: %d %q %q", code, full, diagnostics)
			}
		})
	}
	if after := remoteTree(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("locked explanation mutated workspace")
	}
	writeWorkspaceFile(t, root, "src/main.fy", workspaceSessionRootSource+"\n")
	if code, out, _ := runForTest(append(args, string(target.X8664LinuxGNU), "--symbol", "value"), root); code != ExitPackageFailure || out != "" {
		t.Fatalf("stale lock: %d %q", code, out)
	}
}

func TestWhyEmittedWorkspaceStorageMatchesC(t *testing.T) {
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.why-emitted"
name = "client"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", nativeVectorClient)
	if _, _, err := loadWorkspaceWithLock(root, false); err != nil {
		t.Fatal(err)
	}
	var diagnostic bytes.Buffer
	lowered, code := lowerWorkspaceExecutable(root, true, workspaceCompilePolicy{}, "workspace C transport", &diagnostic)
	if code != ExitSuccess || !lowered.sdkStorage {
		t.Fatalf("lower: %d %s", code, &diagnostic)
	}
	for _, id := range []target.ID{target.X8664LinuxGNU, target.X8664WindowsMSVC} {
		t.Run(string(id), func(t *testing.T) {
			code, ordinary, diagnostics := runForTest([]string{"experimental", "emit-workspace-c", "--locked", "--target", string(id)}, root)
			if code != ExitSuccess || diagnostics != "" {
				t.Fatalf("emit: %d %s", code, diagnostics)
			}
			profile, _ := workspaceEmissionProfile(id)
			traced, report, err := cbackend.EmitStorageTransfersWithExplanation(lowered.program, lowered.entryName, profile, lowered.entryName)
			if err != nil || !report.Emitted || traced.Source != ordinary {
				t.Fatalf("traced C differs from ordinary workspace C: %+v, %v", report, err)
			}
			code, out, diagnostics := runForTest([]string{"experimental", "why-emitted", "--workspace", "--locked", "--target", string(id), "--symbol", lowered.entryName}, root)
			var cliReport cbackend.EmissionExplanation
			if code != ExitSuccess || diagnostics != "" || json.Unmarshal([]byte(out), &cliReport) != nil ||
				!reflect.DeepEqual(cliReport, report) {
				t.Fatalf("SDK workspace explanation: %d %q %q", code, out, diagnostics)
			}
		})
	}
}

func TestWhyEmittedWorkspaceInvalidFlags(t *testing.T) {
	root := t.TempDir()
	base := []string{"experimental", "why-emitted", "--workspace"}
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"--symbol", "main"}, "requires --target"},
		{[]string{"--target", "unsupported", "--symbol", "main"}, "does not know target"},
		{[]string{"--target", string(target.X8664LinuxGNU)}, "requires --symbol"},
		{[]string{"--target", string(target.X8664LinuxGNU), "--symbol", "main", "--source", "other.fy"}, "does not accept flag"},
	} {
		code, out, diagnostics := runForTest(append(base, test.args...), root)
		if code != ExitInvalidInvocation || out != "" || !strings.Contains(diagnostics, test.want) {
			t.Fatalf("%v: %d %q %q", test.args, code, out, diagnostics)
		}
	}
}
