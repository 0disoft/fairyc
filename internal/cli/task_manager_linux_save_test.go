package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

func TestTaskManagerLinuxPreparedSaveFaults(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux GNU amd64, GCC and Clang")
	}
	unit, support := taskManagerUnit(t, target.X8664LinuxGNU)
	hooked := strings.NewReplacer("open(", "task_save_open(", "write(", "task_save_write(", "fsync(", "task_save_sync(", "close(", "task_save_close(", "rename(", "task_save_rename(").Replace(support)
	if hooked == support {
		t.Fatal("production POSIX seam not intercepted")
	}
	sdk, err := sdkstdlib.Load()
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	if keep := os.Getenv("FAIRYC_TEST_EVIDENCE_DIR"); keep != "" {
		if err := os.MkdirAll(keep, 0700); err != nil {
			t.Fatal(err)
		}
		output, err = os.MkdirTemp(keep, "linux-save-")
		if err != nil {
			t.Fatal(err)
		}
	}
	var records []map[string]any
	var images []map[string]any
	defer func() {
		receipt := map[string]any{"schema": "fairyc.linux-task-save/v1", "sdkVersion": sdkstdlib.Version, "sdkSourceDigest": sdk.Digest.String(), "generatedCSourceSHA256": taskSaveDigest([]byte(unit.Source)), "productionSupportSHA256": taskSaveDigest([]byte(support)), "instrumentedSupportSHA256": taskSaveDigest([]byte(taskSaveLinuxHooks + hooked)), "images": images, "cases": records, "passed": len(records), "failed": t.Failed(), "faultKind": "test-only POSIX call injection and bounded process kills; no power-loss claim"}
		data, err := json.MarshalIndent(receipt, "", "  ")
		if err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(filepath.Join(output, "receipt.json"), append(data, '\n'), 0600); err != nil {
			t.Error(err)
		}
		t.Logf("Linux save cases=%d evidence=%s", len(records), output)
	}()
	const old = "FYTASK1\t2\n1\t0\told\n"
	const changed = "FYTASK1\t3\n1\t0\told\n2\t0\tnew\n"
	for _, cc := range []string{"gcc", "clang"} {
		tc, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
		if err != nil {
			t.Fatal(err)
		}
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(cc+"/"+profile.Name, func(t *testing.T) {
				artifact, err := cdriver.Build(tc, profile, unit.Source, taskSaveLinuxHooks+hooked)
				if err != nil {
					t.Fatal(err)
				}
				image := filepath.Join(output, "task-save-"+cc+"-"+profile.Name)
				if err := os.WriteFile(image, artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				images = append(images, map[string]any{"cc": cc, "profile": profile.Name, "sha256": taskSaveDigest(artifact.Bytes), "compiler": tc.Version, "testOnly": true})
				for _, mode := range []string{"ok", "first-write-full", "partial-full", "sync-failure", "close-failure", "replace-full", "during-write", "after-sync", "before", "after"} {
					t.Run(mode, func(t *testing.T) {
						dir := t.TempDir()
						store := filepath.Join(dir, "한글 작업.tsv")
						if err := os.WriteFile(store, []byte(old), 0600); err != nil {
							t.Fatal(err)
						}
						killed := mode == "during-write" || mode == "after-sync" || mode == "before" || mode == "after"
						out, diag, exit, err := taskSaveExecute(image, mode, killed, store, "add", "new")
						if err != nil {
							t.Fatal(err)
						}
						live, temp := old, changed
						if mode == "ok" || mode == "after" {
							live = changed
						}
						if mode == "first-write-full" {
							temp = ""
						}
						if mode == "partial-full" || mode == "during-write" {
							temp = changed[:2]
						}
						temporary := mode != "ok" && mode != "after"
						practicalFileEquals(t, store, live)
						if temporary {
							practicalFileEquals(t, store+".tmp", temp)
						} else {
							taskSaveAbsent(t, store+".tmp")
						}
						if killed {
							practicalFileEquals(t, store+".lock", "")
						} else {
							taskSaveAbsent(t, store+".lock")
						}
						taskSaveAbsent(t, store+".bak")
						wantCode, wantOut, wantError := 1, "", ""
						switch mode {
						case "ok":
							wantCode, wantOut = 0, "added id=2\n"
						case "during-write", "after-sync", "before", "after":
							if exit != -1 || out != "R" || diag != "" {
								t.Fatalf("owned kill: %d %q %q", exit, out, diag)
							}
							wantCode, wantOut = exit, "R"
						case "replace-full":
							wantError = fmt.Sprintf("%s: error: replace failed; live store unchanged; temporary file preserved (status=1 os_code=28)\n", store)
						default:
							operation, number := "write failed; temporary file preserved", 28
							if mode == "sync-failure" {
								operation, number = "sync failed; temporary file preserved", 5
							}
							if mode == "close-failure" {
								operation, number = "close failed", 5
							}
							wantError = fmt.Sprintf("%s.tmp: error: %s (status=1 os_code=%d)\n%s: error: save not installed; temporary file preserved\n", store, operation, number, store)
						}
						if exit != wantCode || out != wantOut || diag != wantError {
							t.Fatalf("exit=%d out=%q error=%q want=(%d,%q,%q)", exit, out, diag, wantCode, wantOut, wantError)
						}
						// A restarted process must not promote or delete any surviving sidecar.
						restartedOut, restartedError, restartedCode, err := taskSaveExecute(image, "", false, store, "list")
						if err != nil {
							t.Fatal(err)
						}
						if mode == "ok" {
							if restartedCode != 0 || restartedOut != "1\topen\told\n2\topen\tnew\n" || restartedError != "" {
								t.Fatal("successful restart changed state")
							}
						} else {
							message := store + ".tmp: error: recovery sidecar exists; inspect .tmp/.bak before retry\n"
							if killed {
								message = store + ".lock: error: create lock failed; another run or stale lock (status=1 os_code=17)\n"
							}
							if restartedCode != 1 || restartedOut != "" || restartedError != message {
								t.Fatalf("restart: %d %q %q", restartedCode, restartedOut, restartedError)
							}
						}
						practicalFileEquals(t, store, live)
						if temporary {
							practicalFileEquals(t, store+".tmp", temp)
						}
						if killed {
							practicalFileEquals(t, store+".lock", "")
						}
						// Explicit operator recovery after the owned child has exited.
						if killed {
							if err := os.Remove(store + ".lock"); err != nil {
								t.Fatal(err)
							}
						}
						if temporary {
							if err := os.Rename(store+".tmp", store+".orphan"); err != nil {
								t.Fatal(err)
							}
						}
						recovered, recoveryError, recoveryCode, err := taskSaveExecute(image, "", false, store, "list")
						wantList := "1\topen\told\n"
						if live == changed {
							wantList += "2\topen\tnew\n"
						}
						if err != nil || recoveryCode != 0 || recoveryError != "" || recovered != wantList {
							t.Fatalf("operator recovery: %d %q %q %v", recoveryCode, recovered, recoveryError, err)
						}
						practicalFileEquals(t, store, live)
						if temporary {
							practicalFileEquals(t, store+".orphan", temp)
						}
						records = append(records, map[string]any{"cc": cc, "profile": profile.Name, "mode": mode, "cwd": cwd, "argv": []string{image, store, "add", "new"}, "stdout": out, "stderr": diag, "exitCode": exit, "forcedTermination": killed, "liveBytes": live, "temporaryExists": temporary, "temporaryBytes": temp, "lockExists": killed, "restartExitCode": restartedCode, "restartStdout": restartedOut, "restartStderr": restartedError, "manualRecoveryList": recovered})
					})
				}
			})
		}
	}
}
