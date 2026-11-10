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

func TestWindowsGUIIME(t *testing.T) {
	windowsGUIDesktopProfiles(t, "--verify-ime", "actual-korean-ime", 4)
}

func TestWindowsGUIIMEPolicy(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_DESKTOP_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_DESKTOP_INTEGRATION=1 with a Korean IME on an idle Windows desktop")
	}
	for _, mutation := range []struct{ name, file, from, to string }{
		{"application-key", "src/main.fy", "if (flags >= 32) { return 0; }", "if (flags >= 32) { return 1; }"},
		{"focus-key", "src/main.fy", "fn i32 on_tab(i32 focused, i32 flags) {\n    if (flags >= 32) { return 0; }", "fn i32 on_tab(i32 focused, i32 flags) {\n    if (flags >= 32) { return 202; }"},
		{"native-boundary", "native/ime.c", "owns_key = state.composing || boundary_pending;", "owns_key = state.composing;"},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			root := windowsGUIExample(t)
			source, err := os.ReadFile(filepath.Join(root, mutation.file))
			if err != nil {
				t.Fatal(err)
			}
			// Git checkouts may use CRLF; mutation is independent of checkout EOL.
			text := strings.ReplaceAll(string(source), "\r\n", "\n")
			broken := strings.Replace(text, mutation.from, mutation.to, 1)
			if broken == text {
				t.Fatal("missing IME ownership mutation target")
			}
			writeWorkspaceFile(t, root, mutation.file, broken)
			if code, out, diag := runForTest([]string{"build", "--release"}, root); code != 0 || diag != "" {
				t.Fatalf("build IME control: %d %s %s", code, out, diag)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 35*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, "python", "-B", "../../examples/win32_minimal/control.py",
				"--exe", filepath.Join(root, "build/win32-minimal.exe"), "--verify-ime").CombinedOutput()
			var report struct {
				OK    bool
				Error string
			}
			exit, exited := err.(*exec.ExitError)
			if !exited || exit.ExitCode() != 1 || json.Unmarshal(out, &report) != nil || report.OK ||
				!strings.Contains(report.Error, "IME shortcut executed an application action") {
				t.Fatalf("broken IME priority must expose the application collision: %v %s", err, out)
			}
		})
	}
}
