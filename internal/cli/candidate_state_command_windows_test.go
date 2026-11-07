//go:build windows

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func stateCommandProcess(t *testing.T, image, directory string, args ...string) (int, []byte, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, image, args...)
	cmd.Dir, cmd.WaitDelay = directory, 2*time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return code, stdout.Bytes(), stderr.Bytes()
}

func TestCandidateStateCommandWindows(t *testing.T) {
	entry, err := os.ReadFile(filepath.Join("..", "..", "tools", "dev", "fairyc-state", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("actual authored developer entry sha256=%x", sha256.Sum256(entry))
	root := t.TempDir()
	reference := filepath.Join(root, "fairyc-state-go.exe")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-o", reference, "./tools/dev/fairyc-state")
	build.Dir = filepath.Join("..", "..")
	build.WaitDelay = time.Second
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("Go developer command build: %v %s", err, out)
	}
	runtimePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "runtime.pack"))
	framePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-frame.py", filepath.Join(root, "frame.pack"))
	trials := []stateGraphAuthoredTrial{{"read-after-write", stateGraphAuthoredSource("storage_read,storage_write", `storage::write("count",42);let n=storage::read("count");storage::write("count",n);storage::commit();`, ""), ""}}
	for _, trial := range graphTransportTrials() {
		switch trial.name {
		case "empty-commit", "empty-reject", "trap", "helper-transitive", "present-borrow-copy":
			trials = append(trials, trial)
		}
	}
	if len(trials) != 6 {
		t.Fatalf("command fixture count=%d want=6", len(trials))
	}
	candidateAtlasProbeSize(t, "module_state_command_windows", string(entry), 512<<10, func(t *testing.T, image string) {
		profile := "low"
		if strings.Contains(t.Name(), "/optimized") {
			profile = "optimized"
		}
		// Help and invocation refusal must work before distribution assets exist.
		for _, tr := range []struct {
			args []string
			code int
		}{{[]string{"--help"}, 0}, {nil, 2}, {[]string{"execute", "missing", "missing.fy"}, 2}} {
			code, out, stderr := stateCommandProcess(t, image, root, tr.args...)
			if code != tr.code || (code == 0 && (!bytes.Contains(out, []byte("--engine native-graph")) || len(stderr) != 0)) || (code != 0 && len(out) != 0) {
				t.Fatalf("asset-free admission code=%d want=%d stdout=%q stderr=%q", code, tr.code, out, stderr)
			}
		}
		runtimePath := filepath.Join(filepath.Dir(image), "fairyc-state-graph-runtime.pack")
		framePath := filepath.Join(filepath.Dir(image), "fairyc-state-graph-frame.pack")
		if err := os.WriteFile(runtimePath, runtimePack, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(framePath, framePack, 0600); err != nil {
			t.Fatal(err)
		}
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				directory := filepath.Join(t.TempDir(), "원본 source 😀")
				if err := os.Mkdir(directory, 0700); err != nil {
					t.Fatal(err)
				}
				sourcePath := filepath.Join(directory, "ledger.fy")
				if err := os.WriteFile(sourcePath, []byte(trial.text), 0600); err != nil {
					t.Fatal(err)
				}
				fyStore, goStore := filepath.Join(directory, "Fairy store"), filepath.Join(directory, "Go store")
				for _, path := range []string{fyStore, goStore} {
					if err := os.Mkdir(path, 0700); err != nil {
						t.Fatal(err)
					}
				}
				for step, action := range []string{"init", "snapshot", "execute", "snapshot"} {
					fyArgs := []string{action, fyStore, filepath.Base(sourcePath), "--engine", "native-graph", "--profile", profile}
					goArgs := append([]string(nil), fyArgs...)
					goArgs[1] = goStore
					wantCode, want, wantErr := stateCommandProcess(t, reference, directory, goArgs...)
					gotCode, got, gotErr := stateCommandProcess(t, image, directory, fyArgs...)
					if gotCode != wantCode || !bytes.Equal(got, want) || len(gotErr) != 0 || len(wantErr) != 0 {
						t.Fatalf("actual command %s/%d %s: Fairy code=%d out=%s err=%s; Go code=%d out=%s err=%s", trial.name, step, action, gotCode, got, gotErr, wantCode, want, wantErr)
					}
					fyBytes, err := os.ReadFile(filepath.Join(fyStore, "state-v1.bin"))
					if err != nil {
						t.Fatal(err)
					}
					goBytes, err := os.ReadFile(filepath.Join(goStore, "state-v1.bin"))
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(fyBytes, goBytes) {
						t.Fatal("actual durable Go/Fairy bytes differ")
					}
					t.Logf("actual %s code=%d Go JSON exact bytes=%d durable=%x", action, gotCode, len(got), sha256.Sum256(fyBytes))
				}
			})
		}
		t.Run("pre-store-failures", func(t *testing.T) {
			directory := t.TempDir()
			storePath := filepath.Join(directory, "store")
			if err := os.Mkdir(storePath, 0700); err != nil {
				t.Fatal(err)
			}
			sourcePath := filepath.Join(directory, "ledger.fy")
			for _, tr := range []struct {
				name, text string
				code       int
			}{{"missing", "", 4}, {"syntax", "module bad; invalid", 3}, {"module-name", strings.Replace(trials[0].text, "module ledger;", "module different;", 1), 3}, {"private-size", trials[0].text + strings.Repeat(" ", 65537-len(trials[0].text)), 3}, {"bad-pin", trials[0].text, 4}} {
				t.Run(tr.name, func(t *testing.T) {
					if tr.name != "missing" {
						if err := os.WriteFile(sourcePath, []byte(tr.text), 0600); err != nil {
							t.Fatal(err)
						}
					}
					if tr.name == "bad-pin" {
						if err := os.WriteFile(runtimePath, []byte("corrupt"), 0600); err != nil {
							t.Fatal(err)
						}
						defer func() {
							if err := os.WriteFile(runtimePath, runtimePack, 0600); err != nil {
								t.Error(err)
							}
						}()
					}
					code, out, stderr := stateCommandProcess(t, image, directory, "execute", storePath, sourcePath, "--engine", "native-graph", "--profile", profile)
					if code != tr.code || len(out) != 0 || len(stderr) == 0 {
						t.Fatalf("%s code=%d want=%d stdout=%q stderr=%q", tr.name, code, tr.code, out, stderr)
					}
					entries, err := os.ReadDir(storePath)
					if err != nil || len(entries) != 0 {
						t.Fatalf("pre-build failure touched store: %v %v", entries, err)
					}
				})
			}
		})
		t.Logf("profile=%s actual developer commands include fresh file admission, pinned assets, native execution, rejection/trap rollback and Go byte comparison", profile)
	})
}
