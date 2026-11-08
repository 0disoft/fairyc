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

// Controlled messages verify real native geometry/fonts without changing
// desktop scale settings or claiming a hardware monitor transition.
func TestWindowsGUIDPI(t *testing.T) {
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("DPI CLI verification requires Python 3")
	}
	root := windowsGUIExample(t)
	image := filepath.Join(root, "build/win32-minimal.exe")
	for _, profile := range []struct {
		name  string
		flags []string
	}{{"development", nil}, {"release", []string{"--release"}}} {
		t.Run(profile.name, func(t *testing.T) {
			if code, out, diag := runForTest(append([]string{"build"}, profile.flags...), root); code != 0 || diag != "" {
				t.Fatalf("build DPI example: %d %s %s", code, out, diag)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, python, "-B", "../../examples/win32_minimal/control.py",
				"--exe", image, "--verify-dpi").CombinedOutput()
			if err != nil {
				t.Fatalf("native DPI verification: %v %s", err, out)
			}
			var report struct {
				OK                              bool
				Verification                    string
				ProcessExited                   bool
				PhysicalMonitorTransitionTested bool
				Observations                    []json.RawMessage
			}
			if err := json.Unmarshal(out, &report); err != nil || !report.OK ||
				report.Verification != "injected-win32-dpi" || !report.ProcessExited ||
				report.PhysicalMonitorTransitionTested || len(report.Observations) != 5 {
				t.Fatalf("invalid controlled DPI receipt: %v %s", err, out)
			}
		})
	}
	// The native bridge must not apply the recommended rectangle by itself.
	path := filepath.Join(root, "src/main.fy")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(source),
		"fairy_win32_accept_dpi(window, dpi_changes, scale(400, dpi), scale(240, dpi))", "0", 1)
	if broken == string(source) {
		t.Fatal("missing Fairy DPI acceptance mutation target")
	}
	writeWorkspaceFile(t, root, "src/main.fy", broken)
	if code, out, diag := runForTest([]string{"build", "--release"}, root); code != 0 || diag != "" {
		t.Fatalf("build ignored DPI control: %d %s %s", code, out, diag)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, python, "-B", "../../examples/win32_minimal/control.py",
		"--exe", image, "--verify-dpi").CombinedOutput()
	var report struct {
		OK    bool
		Error string
	}
	exit, exited := err.(*exec.ExitError)
	if !exited || exit.ExitCode() != 1 || json.Unmarshal(out, &report) != nil || report.OK || !strings.Contains(report.Error, "DPI") {
		t.Fatalf("Fairy ignoring DPI must fail verification: %v %s", err, out)
	}
}
