package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Desktop input is opt-in: ordinary CI must not manipulate foreground focus.
func TestWindowsGUIDesktopClose(t *testing.T) {
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
				"--exe", filepath.Join(root, "build/win32-minimal.exe"), "--verify-desktop-close").CombinedOutput()
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
				}
			}
			if err := json.Unmarshal(out, &report); err != nil || !report.OK ||
				report.Verification != "win32-desktop-close" || !report.ProcessExited || len(report.Observations) != 2 {
				t.Fatalf("invalid desktop close receipt: %v %s", err, out)
			}
			for _, row := range report.Observations {
				if !row.Closed || !row.SystemClose {
					t.Fatalf("missing SC_CLOSE or Fairy approval: %s", out)
				}
			}
		})
	}
}
