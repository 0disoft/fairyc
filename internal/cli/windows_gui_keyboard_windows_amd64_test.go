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

func TestWindowsGUIKeyboard(t *testing.T) {
	windowsGUIDesktopProfiles(t, "--verify-keyboard", "win32-keyboard", 5)
}

func TestWindowsGUIKeyboardPolicy(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_DESKTOP_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_DESKTOP_INTEGRATION=1 on an idle interactive Windows desktop")
	}
	python, err := exec.LookPath("python")
	if err != nil {
		t.Fatal("keyboard verification requires Python 3", err)
	}
	for _, mutation := range []struct{ name, from, to, failure string }{
		{"enter", "if (code == 13) { return 1; }", "if (code == 13) { return 0; }", "Enter did not run Fairy"},
		{"modified-enter", "if (flags != 0) { return 0; }", "if (flags != 0) { return 1; }", "modified Enter must leave"},
		{"escape", "if (code == 27) { return 3; }", "if (code == 27) { return 0; }", "Escape did not follow Fairy"},
		{"close-denial", "fairy_win32_resolve_close(window, 1)", "fairy_win32_resolve_close(window, 0)", ""},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			root := windowsGUIExample(t)
			path := filepath.Join(root, "src/main.fy")
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			broken := strings.Replace(string(source), mutation.from, mutation.to, 1)
			if broken == string(source) {
				t.Fatal("missing Fairy keyboard mutation target")
			}
			writeWorkspaceFile(t, root, "src/main.fy", broken)
			if code, out, diag := runForTest([]string{"build", "--release"}, root); code != 0 || diag != "" {
				t.Fatalf("build keyboard policy control: %d %s %s", code, out, diag)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			image := filepath.Join(root, "build/win32-minimal.exe")
			if mutation.failure == "" {
				probe := `import sys
sys.path.insert(0, sys.argv[1])
from control import Session
session = Session(sys.argv[2])
try:
    initial = session.initial
    for command in (b'arm-key\t0\t201', b'arm-key\t13\t999', b'arm-key\t27junk\t201'):
        assert session.wire(command)['ok'] is False
    assert session.request({'command': 'state'}) == initial
    for focus in ('input', 'apply'):
        denied = session.request({'command': 'key', 'key': 'escape', 'focus': focus})
        assert denied['closed'] is False and denied['keyAction'] == 3
        assert session.process.poll() is None
    edited = session.request({'command': 'input', 'text': 'after Escape denial'})
    applied = session.request({'command': 'key', 'key': 'enter'})
    assert applied['clicks'] == 1 and applied['edits'] == edited['edits']
    assert applied['result'] == 'Result: after Escape denial'
    assert session.request({'command': 'state'})['clicks'] == 1
finally:
    session.finish()
`
				out, err := exec.CommandContext(ctx, python, "-B", "-c", probe, "../../examples/win32_minimal", image).CombinedOutput()
				if err != nil || len(out) != 0 {
					t.Fatalf("Escape denial must preserve interactive GUI: %v %s", err, out)
				}
				return
			}
			out, err := exec.CommandContext(ctx, python, "-B", "../../examples/win32_minimal/control.py",
				"--exe", image, "--verify-keyboard").CombinedOutput()
			var report struct {
				OK    bool
				Error string
			}
			exit, exited := err.(*exec.ExitError)
			if !exited || exit.ExitCode() != 1 || json.Unmarshal(out, &report) != nil || report.OK ||
				!strings.Contains(report.Error, mutation.failure) {
				t.Fatalf("broken Fairy keyboard policy must fail verification: %v %s", err, out)
			}
		})
	}
}
