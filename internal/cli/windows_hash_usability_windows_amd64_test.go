package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Explicit desktop opt-in only; high contrast additionally requires its own opt-in.
func verifyWindowsHashUsability(t *testing.T, image, selection string) {
	t.Helper()
	highContrast := os.Getenv("FAIRYC_HASH_HIGH_CONTRAST") == "1"
	modes := strings.Split(selection, ",")
	displaySelected := false
	for _, mode := range modes {
		switch mode {
		case "accessibility", "keyboard":
		case "display", "all":
			displaySelected = true
		default:
			t.Fatalf("invalid FAIRYC_HASH_USABILITY mode: %q", mode)
		}
	}
	if highContrast && !displaySelected {
		t.Fatal("FAIRYC_HASH_HIGH_CONTRAST requires display or all usability checks")
	}
	for _, mode := range modes {
		ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
		args := []string{"-B", "../../examples/win32_hash/verify_gui.py", "--exe", image, "--usability", mode}
		if highContrast && (mode == "display" || mode == "all") {
			args = append(args, "--high-contrast")
		}
		out, err := exec.CommandContext(ctx, "python", args...).CombinedOutput()
		cancel()
		var report struct {
			OK, ProcessExited bool
			Verification      string
			Accessibility     struct {
				OK              bool
				NativeTabOrders map[string][]int
			}
			Keyboard struct{ ActualSendInput bool }
			Display  struct {
				PerMonitorV2, PhysicalDifferentDpiTransition bool
				TransitionSkipReason                         string
				Observations                                 []struct{ DPI, DigestHeight, FontHeight int }
				HighContrast                                 struct{ OSEnabled, OriginalFlagsAndColorsRestored bool }
			}
		}
		if err != nil || json.Unmarshal(out, &report) != nil || !report.OK || !report.ProcessExited || report.Verification != "actual-win32-hash-usability" {
			t.Fatalf("native usability %s: %v %s", mode, err, out)
		}
		if (mode == "accessibility" || mode == "all") && (!report.Accessibility.OK || len(report.Accessibility.NativeTabOrders) != 6) {
			t.Fatalf("incomplete accessibility verification: %s", out)
		}
		if (mode == "keyboard" || mode == "all") && !report.Keyboard.ActualSendInput {
			t.Fatalf("missing actual keyboard input: %s", out)
		}
		if mode == "display" || mode == "all" {
			if !report.Display.PerMonitorV2 || len(report.Display.Observations) == 0 ||
				highContrast && (!report.Display.HighContrast.OSEnabled || !report.Display.HighContrast.OriginalFlagsAndColorsRestored) {
				t.Fatalf("incomplete display verification: %s", out)
			}
			if !report.Display.PhysicalDifferentDpiTransition {
				if report.Display.TransitionSkipReason == "" {
					t.Fatalf("missing physical transition skip reason: %s", out)
				}
				t.Logf("physical DPI transition omitted: %s", report.Display.TransitionSkipReason)
			}
			t.Logf("actual monitor observations: %+v", report.Display.Observations)
		}
		t.Logf("native usability %s passed", mode)
	}
}
