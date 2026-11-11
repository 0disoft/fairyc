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

func TestWindowsHashUtility(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_DESKTOP_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_DESKTOP_INTEGRATION=1 on an unlocked Windows desktop for native file dialogs")
	}
	root := t.TempDir()
	for _, name := range []string{"fairy.toml", "fairy.windows.json", "src/main.fy", "src/hash_file.fy",
		"src/sha256.fy", "src/sha256_stream.fy", "src/application.fy", "src/ui.fy", "native/window.c", "native/app.rc"} {
		data, err := os.ReadFile(filepath.Join("../../examples/win32_hash", name))
		if err != nil {
			t.Fatal(err)
		}
		writeWorkspaceFile(t, root, name, string(data))
	}
	for _, variant := range []struct {
		name  string
		flags []string
	}{{"development", nil}, {"release", []string{"--release"}}} {
		t.Run(variant.name, func(t *testing.T) {
			if code, out, diag := runForTest(append([]string{"build"}, variant.flags...), root); code != 0 || diag != "" {
				t.Fatalf("build: %d %s %s", code, out, diag)
			}
			image := filepath.Join(root, "build/win32-hash.exe")
			data, err := os.ReadFile(image)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("hash utility: %d bytes", len(data))
			for _, verifier := range []struct{ file, kind string }{
				{"verify_cli.py", "fairy-file-sha256"}, {"verify_gui.py", "actual-win32-hash-dialogs"},
			} {
				ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
				out, err := exec.CommandContext(ctx, "python", "-B", "../../examples/win32_hash/"+verifier.file, "--exe", image).CombinedOutput()
				cancel()
				var report struct {
					OK, ProcessExited, NativeOpenAndSave, CloseDuringHash, SourceProtected   bool
					TwoLineDigest, ClipboardDigest, ClipboardRestored, ClipboardBusyRecovery bool
					NativeVisualStyles, ResponsiveLayout, MinimizeRestore                    bool
					FilenameResizeRepaint                                                    bool
					BoundedDigestLayout                                                      bool
					NativeTooltips                                                           bool
					Verification                                                             string
					Cases                                                                    int
				}
				if err != nil || json.Unmarshal(out, &report) != nil || !report.OK || !report.ProcessExited || report.Verification != verifier.kind {
					t.Fatalf("%s: %v %s", verifier.file, err, out)
				}
				if verifier.file == "verify_cli.py" && report.Cases != 19 || verifier.file == "verify_gui.py" &&
					(!report.NativeOpenAndSave || !report.CloseDuringHash || !report.SourceProtected ||
						!report.TwoLineDigest || !report.ClipboardDigest || !report.ClipboardRestored || !report.ClipboardBusyRecovery ||
						!report.NativeVisualStyles || !report.ResponsiveLayout || !report.MinimizeRestore || !report.FilenameResizeRepaint || !report.BoundedDigestLayout || !report.NativeTooltips) {
					t.Fatalf("incomplete utility verification: %s", out)
				}
			}
		})
	}
}
