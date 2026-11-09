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

func TestWindowsGUIMouseTarget(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_DESKTOP_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_DESKTOP_INTEGRATION=1 on an idle interactive Windows desktop")
	}
	python, err := exec.LookPath("python")
	if err != nil {
		t.Fatal("mouse target verification requires Python 3", err)
	}
	root := windowsGUIExample(t)
	if code, out, diag := runForTest([]string{"build", "--release"}, root); code != 0 || diag != "" {
		t.Fatalf("build mouse target: %d %s %s", code, out, diag)
	}
	for _, scenario := range []struct{ name, from, to, failure string }{
		{"targeted", "", "", ""},
		{"untargeted-press", "pointer_input(x, y, 0xC003, token)", "pointer_input(x, y, 2, token)", "native press hit"},
		{"untargeted-release", "pointer_input(x, y, 0xC005, token)", "pointer_input(x, y, 4, token)", "GUI reply timed out"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			controller := t.TempDir()
			for _, name := range []string{"control.py", "desktop_input.py"} {
				source, err := os.ReadFile(filepath.Join("../../examples/win32_minimal", name))
				if err != nil {
					t.Fatal(err)
				}
				if name == "desktop_input.py" && scenario.from != "" {
					broken := strings.Replace(string(source), scenario.from, scenario.to, 1)
					if broken == string(source) {
						t.Fatal("missing targeted mouse input mutation")
					}
					source = []byte(broken)
				}
				if err := os.WriteFile(filepath.Join(controller, name), source, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, python, "-B", "testdata/windows_gui_mouse_target.py",
				controller, filepath.Join(root, "build/win32-minimal.exe")).CombinedOutput()
			var report struct {
				OK            bool
				Verification  string
				Perturbations []string
				Error         string
			}
			if decode := json.Unmarshal(out, &report); decode != nil {
				t.Fatalf("invalid mouse observation: %v %s", decode, out)
			}
			if scenario.failure == "" {
				if err != nil || !report.OK || report.Verification != "win32-mouse-target" || len(report.Perturbations) != 2 {
					t.Fatalf("atomic target must survive cursor perturbations: %v %s", err, out)
				}
			} else if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 || report.OK || !strings.Contains(report.Error, scenario.failure) {
				t.Fatalf("untargeted input must expose cursor race: %v %s", err, out)
			}
		})
	}
}
