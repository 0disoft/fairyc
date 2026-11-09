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

func TestWindowsGUIFocus(t *testing.T) {
	windowsGUIDesktopProfiles(t, "--verify-focus", "win32-focus", 8)
}

func TestWindowsGUIFocusPolicy(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_DESKTOP_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_DESKTOP_INTEGRATION=1 on an idle interactive Windows desktop")
	}
	python, err := exec.LookPath("python")
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []struct{ name, from, to string }{
		{"forward", "return if (focused == 201) { 202 } else { 201 };", "return 201;"},
		{"reverse", "if (flags == 1) {", "if (flags == 99) {"},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			root := windowsGUIExample(t)
			source, err := os.ReadFile(filepath.Join(root, "src/main.fy"))
			if err != nil {
				t.Fatal(err)
			}
			broken := strings.Replace(string(source), mutation.from, mutation.to, 1)
			if broken == string(source) {
				t.Fatal("missing Fairy focus mutation target")
			}
			writeWorkspaceFile(t, root, "src/main.fy", broken)
			if code, out, diag := runForTest([]string{"build", "--release"}, root); code != 0 || diag != "" {
				t.Fatalf("build focus control: %d %s %s", code, out, diag)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, python, "-B", "../../examples/win32_minimal/control.py",
				"--exe", filepath.Join(root, "build/win32-minimal.exe"), "--verify-focus").CombinedOutput()
			var report struct {
				OK    bool
				Error string
			}
			exit, exited := err.(*exec.ExitError)
			if !exited || exit.ExitCode() != 1 || json.Unmarshal(out, &report) != nil || report.OK ||
				!strings.Contains(report.Error, "Tab focus policy") {
				t.Fatalf("broken Fairy focus order must fail: %v %s", err, out)
			}
		})
	}
}
