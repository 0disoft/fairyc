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

func TestWindowsGUIDistribution(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_DESKTOP_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_DESKTOP_INTEGRATION=1 with Korean two-set IME on an idle Windows desktop")
	}
	root := windowsGUIExample(t)
	config, err := os.ReadFile(filepath.Join(root, "fairy.windows.json"))
	if err != nil {
		t.Fatal(err)
	}
	slim := strings.ReplaceAll(strings.ReplaceAll(string(config), "native/window.c", "native/window_distribution.c"), "native/ime.c", "native/ime_distribution.c")
	for _, profile := range []struct {
		name  string
		flags []string
	}{{"development", nil}, {"release", []string{"--release"}}} {
		t.Run(profile.name, func(t *testing.T) {
			build := func(settings string) []byte {
				t.Helper()
				writeWorkspaceFile(t, root, "fairy.windows.json", settings)
				if code, out, diag := runForTest(append([]string{"build"}, profile.flags...), root); code != 0 || diag != "" {
					t.Fatalf("build: %d %s %s", code, out, diag)
				}
				data, err := os.ReadFile(filepath.Join(root, "build/win32-minimal.exe"))
				if err != nil {
					t.Fatal(err)
				}
				return data
			}
			diagnostic := build(string(config))
			distribution := build(slim)
			if len(distribution) >= len(diagnostic) {
				t.Fatalf("distribution must remove diagnostic bytes: %d >= %d", len(distribution), len(diagnostic))
			}
			t.Logf("diagnostic %d bytes; distribution %d bytes; saved %d", len(diagnostic), len(distribution), len(diagnostic)-len(distribution))
			for _, flag := range []string{"", "smoke", "automation-stdio"} {
				args := []string{"-B", "../../examples/win32_minimal/verify_distribution.py", "--exe", filepath.Join(root, "build/win32-minimal.exe")}
				if flag != "" {
					args = append(args, "--launch-flag", flag)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				out, err := exec.CommandContext(ctx, "python", args...).CombinedOutput()
				cancel()
				var report struct {
					OK, ProcessExited, PrivateProbeAbsent, DiagnosticImportsAbsent bool
					Verification                                                   string
					FileBytes                                                      int
				}
				if err != nil || json.Unmarshal(out, &report) != nil || !report.OK || !report.ProcessExited ||
					!report.PrivateProbeAbsent || !report.DiagnosticImportsAbsent || report.FileBytes != len(distribution) ||
					report.Verification != "external-win32-distribution" {
					t.Fatalf("distribution external verification %q: %v %s", flag, err, out)
				}
			}
		})
	}
}
