package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

// The only replaced code is the Win32 I/O seam in the production support unit.
// Application validation, mutation, publication, cleanup and response are emitted
// from the exact Fairy example. There is no parallel C implementation of saving.
func TestTaskManagerPreparedSaveFaults(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := filepath.Join("..", "..")
	workspace := t.TempDir()
	example := filepath.Join(root, "examples", "task_manager_cli")
	copyFile := func(from, to string) {
		t.Helper()
		data, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(to), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(to, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	copyFile(filepath.Join(example, "fairy.toml"), filepath.Join(workspace, "fairy.toml"))
	entries, err := os.ReadDir(filepath.Join(example, "src"))
	if err != nil {
		t.Fatal(err)
	}
	sourceHashes := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".fy") {
			t.Fatal("unexpected example source")
		}
		from := filepath.Join(example, "src", e.Name())
		copyFile(from, filepath.Join(workspace, "src", e.Name()))
		data, _ := os.ReadFile(from)
		sourceHashes[e.Name()] = taskSaveDigest(data)
	}
	var diagnostics strings.Builder
	lowered, code := lowerWorkspaceExecutable(workspace, false, workspaceCompilePolicy{stableDiagnostics: true}, "task save fault evidence", &diagnostics)
	if code != ExitSuccess {
		t.Fatalf("lower: %d %s", code, diagnostics.String())
	}
	unit, err := cbackend.Emit(lowered.program, lowered.entryName, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	support, err := windowsTargetSupportSource(unit.ForeignSymbols)
	if err != nil {
		t.Fatal(err)
	}
	hooked := strings.NewReplacer("WriteFile(", "task_save_write(", "FlushFileBuffers(", "task_save_sync(", "CloseHandle(", "task_save_close(", "MoveFileExW(", "task_save_move(").Replace(support)
	if hooked == support {
		t.Fatal("no production Win32 seam replaced")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, ".cache", "task-save-run")
	if err := os.MkdirAll(output, 0700); err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	var images []map[string]any
	defer func() {
		receipt := map[string]any{"sourceSHA256": sourceHashes, "generatedCSourceSHA256": taskSaveDigest([]byte(unit.Source)), "productionSupportSHA256": taskSaveDigest([]byte(support)), "instrumentedSupportSHA256": taskSaveDigest([]byte(taskSaveNativeHooks + hooked)), "images": images, "cases": records, "passed": len(records), "failed": t.Failed(), "evidence": "compiler-emitted Fairy CLI with test-only Win32 faults and real files; no full host drive or physical power cut"}
		data, err := json.MarshalIndent(receipt, "", "  ")
		if err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(filepath.Join(output, "fault-receipt.json"), append(data, '\n'), 0600); err != nil {
			t.Error(err)
		}
	}()
	const old = "FYTASK1\t2\n1\t0\told\n"
	const changed = "FYTASK1\t3\n1\t0\told\n2\t0\tnew\n"
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, unit.Source, taskSaveNativeHooks+hooked)
			if err != nil {
				t.Fatal(err)
			}
			image := filepath.Join(output, "task-manager-fault-"+profile.Name+".exe")
			if err := os.WriteFile(image, artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			images = append(images, map[string]any{"profile": profile.Name, "sha256": taskSaveDigest(artifact.Bytes), "bytes": len(artifact.Bytes), "path": image, "testOnly": true})
			for _, mode := range []string{"ok", "first-write-full", "partial-full", "sync-failure", "close-failure", "replace-full", "during-write", "after-sync", "before", "after"} {
				t.Run(mode, func(t *testing.T) {
					dir := t.TempDir()
					store := filepath.Join(dir, "한글 작업.tsv")
					if err := os.WriteFile(store, []byte(old), 0600); err != nil {
						t.Fatal(err)
					}
					killed := mode == "during-write" || mode == "after-sync" || mode == "before" || mode == "after"
					stdout, stderr, exit, err := taskSaveExecute(image, mode, killed, store, "add", "new")
					if err != nil {
						t.Fatal(err)
					}
					wantLive, wantTemp := old, changed
					if mode == "ok" || mode == "after" {
						wantLive = changed
					}
					if mode == "first-write-full" {
						wantTemp = ""
					}
					if mode == "partial-full" || mode == "during-write" {
						wantTemp = changed[:2]
					}
					practicalFileEquals(t, store, wantLive)
					tempExists := mode != "ok" && mode != "after"
					if tempExists {
						practicalFileEquals(t, store+".tmp", wantTemp)
					} else {
						taskSaveAbsent(t, store+".tmp")
					}
					taskSaveAbsent(t, store+".bak")
					if killed {
						practicalFileEquals(t, store+".lock", "")
					} else {
						taskSaveAbsent(t, store+".lock")
					}
					if mode == "ok" {
						if exit != 0 || stdout != "added id=2\n" || stderr != "" {
							t.Fatalf("success: %d %q %q", exit, stdout, stderr)
						}
					} else if killed {
						if exit == 0 || stdout != "R" || stderr != "" {
							t.Fatalf("kill: %d %q %q", exit, stdout, stderr)
						}
					} else {
						operation, osCode := "write failed; temporary file preserved", 112
						if mode == "sync-failure" {
							operation, osCode = "sync failed; temporary file preserved", 29
						}
						if mode == "close-failure" {
							operation, osCode = "close failed", 29
						}
						wantError := fmt.Sprintf("%s.tmp: error: %s (status=1 os_code=%d)\n%s: error: save not installed; temporary file preserved\n", store, operation, osCode, store)
						if mode == "replace-full" {
							wantError = fmt.Sprintf("%s: error: replace failed; live store unchanged; temporary file preserved (status=1 os_code=112)\n", store)
						}
						if exit != 1 || stdout != "" || stderr != wantError {
							t.Fatalf("failure: %d %q %q, want %q", exit, stdout, stderr, wantError)
						}
					}
					// Restart twice. Refusal must neither promote nor discard the orphan.
					var restarts []map[string]any
					for i := 0; i < 2; i++ {
						out, diag, result, err := taskSaveExecute(image, "", false, store, "list")
						if err != nil {
							t.Fatal(err)
						}
						if mode == "ok" {
							if result != 0 || out != "1\topen\told\n2\topen\tnew\n" || diag != "" {
								t.Fatalf("restart: %d %q %q", result, out, diag)
							}
						} else {
							wanted := store + ".tmp: error: recovery sidecar exists; inspect .tmp/.bak before retry\n"
							if killed {
								wanted = store + ".lock: error: create lock failed; another run or stale lock (status=1 os_code=80)\n"
							}
							if result != 1 || out != "" || diag != wanted {
								t.Fatalf("orphan restart: %d %q %q", result, out, diag)
							}
						}
						practicalFileEquals(t, store, wantLive)
						if tempExists {
							practicalFileEquals(t, store+".tmp", wantTemp)
						}
						restarts = append(restarts, map[string]any{"exitCode": result, "stdout": out, "stderr": diag})
					}
					// Explicit operator cleanup in a test-owned directory after the child exited.
					if killed {
						if err := os.Remove(store + ".lock"); err != nil {
							t.Fatal(err)
						}
					}
					if tempExists {
						if err := os.Rename(store+".tmp", store+".orphan"); err != nil {
							t.Fatal(err)
						}
					}
					out, diag, result, err := taskSaveExecute(image, "", false, store, "list")
					wantList := "1\topen\told\n"
					if wantLive == changed {
						wantList += "2\topen\tnew\n"
					}
					if err != nil || result != 0 || out != wantList || diag != "" {
						t.Fatalf("operator recovery: %d %q %q %v", result, out, diag, err)
					}
					records = append(records, map[string]any{"profile": profile.Name, "mode": mode, "exitCode": exit, "stdout": stdout, "stderr": stderr, "forcedTermination": killed, "liveBytes": wantLive, "liveSHA256": taskSaveDigest([]byte(wantLive)), "temporaryExistsAtCheckpoint": tempExists, "temporaryBytesAtCheckpoint": wantTemp, "lockExistsAtCheckpoint": killed, "backupAbsent": true, "restarts": restarts, "manualRecoveryList": out})
				})
			}
		})
	}
}

func taskSaveDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func taskSaveAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unexpected path %s: %v", path, err)
	}
}

func taskSaveExecute(image, mode string, kill bool, args ...string) (string, string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, image, args...)
	cmd.WaitDelay = time.Second
	for _, e := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(e), "FAIRYC_TASK_SAVE_FAULT=") {
			cmd.Env = append(cmd.Env, e)
		}
	}
	cmd.Env = append(cmd.Env, "FAIRYC_TASK_SAVE_FAULT="+mode)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if kill {
		pipe, err := cmd.StdoutPipe()
		if err != nil {
			return "", "", -1, err
		}
		if err := cmd.Start(); err != nil {
			return "", "", -1, err
		}
		var marker [1]byte
		_, err = io.ReadFull(pipe, marker[:])
		if err != nil || marker[0] != 'R' {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return "", stderr.String(), -1, fmt.Errorf("checkpoint: %q %v", marker, err)
		}
		if err := cmd.Process.Kill(); err != nil {
			_ = cmd.Wait()
			return "", "", -1, err
		}
		err = cmd.Wait()
		if err == nil || ctx.Err() != nil {
			return "", "", -1, fmt.Errorf("child did not reach controlled termination: %v", err)
		}
		return "R", stderr.String(), cmd.ProcessState.ExitCode(), nil
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if ctx.Err() != nil {
		return "", "", -1, ctx.Err()
	}
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return "", "", -1, err
		}
	}
	return stdout.String(), stderr.String(), cmd.ProcessState.ExitCode(), nil
}
