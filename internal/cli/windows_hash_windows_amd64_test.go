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
	jobOnly := os.Getenv("FAIRYC_HASH_JOB_ONLY") == "1"
	usability := os.Getenv("FAIRYC_HASH_USABILITY")
	if jobOnly && usability != "" {
		t.Fatal("choose FAIRYC_HASH_JOB_ONLY or FAIRYC_HASH_USABILITY, not both")
	}
	guiKind := "actual-win32-hash-dialogs"
	if jobOnly {
		guiKind = "actual-win32-hash-job"
		t.Log("job-only GUI verification: physical input and pixel inspection omitted")
	}
	for _, name := range []string{"fairy.toml", "fairy.windows.json", "src/main.fy", "src/hash_file.fy",
		"src/sha256.fy", "src/sha256_stream.fy", "src/application.fy", "src/ui.fy", "src/progress.fy", "native/window.c", "native/app.rc"} {
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
				{"verify_cli.py", "fairy-file-sha256"}, {"verify_sha.py", "fairy-sha256-vectors"}, {"verify_gui.py", guiKind},
			} {
				if verifier.file == "verify_gui.py" && usability != "" {
					verifyWindowsHashUsability(t, image, usability)
					continue
				}
				ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
				args := []string{"-B", "../../examples/win32_hash/" + verifier.file, "--exe", image}
				if jobOnly && verifier.file == "verify_gui.py" {
					args = append(args, "--job-only")
				}
				out, err := exec.CommandContext(ctx, "python", args...).CombinedOutput()
				cancel()
				var report struct {
					OK, ProcessExited, NativeOpenAndSave, CloseDuringHash, SourceProtected   bool
					TwoLineDigest, ClipboardDigest, ClipboardRestored, ClipboardBusyRecovery bool
					NativeVisualStyles, ResponsiveLayout, MinimizeRestore                    bool
					FilenameResizeRepaint                                                    bool
					BoundedDigestLayout                                                      bool
					NativeTooltips                                                           bool
					ControlBackgrounds                                                       bool
					Verification                                                             string
					Cases                                                                    int
					LargeJob                                                                 struct {
						ProgressObserved, CancelAndReuse bool
					}
				}
				if err != nil || json.Unmarshal(out, &report) != nil || !report.OK || !report.ProcessExited || report.Verification != verifier.kind {
					t.Fatalf("%s: %v %s", verifier.file, err, out)
				}
				if verifier.file == "verify_cli.py" && report.Cases != 19 || verifier.file == "verify_sha.py" && report.Cases != 24 || verifier.file == "verify_gui.py" &&
					(!report.LargeJob.ProgressObserved || !report.LargeJob.CancelAndReuse || !jobOnly && (!report.NativeOpenAndSave || !report.CloseDuringHash || !report.SourceProtected ||
						!report.TwoLineDigest || !report.ClipboardDigest || !report.ClipboardRestored || !report.ClipboardBusyRecovery ||
						!report.NativeVisualStyles || !report.ResponsiveLayout || !report.MinimizeRestore || !report.FilenameResizeRepaint || !report.BoundedDigestLayout || !report.NativeTooltips || !report.ControlBackgrounds)) {
					t.Fatalf("incomplete utility verification: %s", out)
				}
				if jobOnly && verifier.file == "verify_gui.py" {
					t.Logf("native GUI job: %s", out)
				}
			}
		})
	}
}
