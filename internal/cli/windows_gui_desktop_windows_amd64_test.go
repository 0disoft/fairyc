package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Desktop input is opt-in: ordinary CI must not manipulate foreground focus.
func TestWindowsGUIDesktopClose(t *testing.T) {
	windowsGUIDesktopProfiles(t, "--verify-desktop-close", "win32-desktop-close", 2)
}

func windowsGUIDesktopProfiles(t *testing.T, flag, verification string, count int) string {
	t.Helper()
	if os.Getenv("FAIRYC_RUN_DESKTOP_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_DESKTOP_INTEGRATION=1 on an idle interactive Windows desktop")
	}
	python, err := exec.LookPath("python")
	if err != nil {
		t.Fatal("desktop close verification requires Python 3", err)
	}
	root := windowsGUIExample(t)
	for _, profile := range []struct {
		name  string
		flags []string
	}{{"development", nil}, {"release", []string{"--release"}}} {
		t.Run(profile.name, func(t *testing.T) {
			if code, out, diag := runForTest(append([]string{"build"}, profile.flags...), root); code != 0 || diag != "" {
				t.Fatalf("build: %d %s %s", code, out, diag)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, python, "-B", "../../examples/win32_minimal/control.py",
				"--exe", filepath.Join(root, "build/win32-minimal.exe"), flag).CombinedOutput()
			if err != nil {
				t.Fatalf("desktop close: %v %s", err, out)
			}
			var report struct {
				OK            bool
				Verification  string
				ProcessExited bool
				Observations  []struct {
					Closed      bool
					SystemClose bool
					Sizing      bool
					MouseDown   bool
					LiveLayouts int
				}
			}
			if err := json.Unmarshal(out, &report); err != nil || !report.OK ||
				report.Verification != verification || !report.ProcessExited || len(report.Observations) != count {
				t.Fatalf("invalid desktop close receipt: %v %s", err, out)
			}
			for _, row := range report.Observations {
				if flag == "--verify-desktop-close" && (!row.Closed || !row.SystemClose) {
					t.Fatalf("missing SC_CLOSE or Fairy approval: %s", out)
				}
				if flag == "--verify-drag" && (!row.Sizing || !row.MouseDown || row.LiveLayouts == 0) {
					t.Fatalf("layout must be observed before mouse release: %s", out)
				}
			}
		})
	}
	return root
}

func TestWindowsGUILiveDrag(t *testing.T) {
	root := windowsGUIDesktopProfiles(t, "--verify-drag", "win32-live-drag", 4)
	// Removing just the modal handoff must expose stale rectangles before
	// release, even though the ordinary event pump still updates after release.
	path := filepath.Join(root, "native/window.c")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(source), "SwitchToFiber(state->app_fiber); /* live-size handoff */", "/* delayed until modal loop returns */", 1)
	if broken == string(source) {
		t.Fatal("missing modal handoff mutation target")
	}
	writeWorkspaceFile(t, root, "native/window.c", broken)
	if code, out, diag := runForTest([]string{"build", "--release"}, root); code != 0 || diag != "" {
		t.Fatalf("build delayed-layout control: %d %s %s", code, out, diag)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "python", "-B", "../../examples/win32_minimal/control.py",
		"--exe", filepath.Join(root, "build/win32-minimal.exe"), "--verify-drag").CombinedOutput()
	var report struct {
		OK    bool
		Error string
	}
	exit, exited := err.(*exec.ExitError)
	if !exited || exit.ExitCode() != 1 || json.Unmarshal(out, &report) != nil || report.OK || !strings.Contains(report.Error, "Fairy layout") {
		t.Fatalf("delayed layout must fail before mouse release: %v %s", err, out)
	}
}
