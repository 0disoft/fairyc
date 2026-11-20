package cdriver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestWindowsProjectCompileSerialParallelPE(t *testing.T) {
	tc, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	header := "#include <stdint.h>\nextern const int32_t input;\nint32_t left(void);\nint32_t right(void);\n"
	literals := "#include \"fy_project.h\"\nconst int32_t input=7;\n"
	units := []string{
		"#include \"fy_project.h\"\nint main(void){return (int)(left()+right());}\n",
		"#include \"fy_project.h\"\nint32_t left(void){return input*3;}\n",
		"#include \"fy_project.h\"\nint32_t right(void){return input+14;}\n",
	}
	support := "int compiler_runtime=0;\n"
	for _, profile := range []Profile{LowProfile, OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			serial, err := BuildWindowsMSVCTargetPartsImageWithStackReserve(tc, profile, header, literals, units, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			parallel, err := BuildWindowsMSVCCompilerHostPartsWithStackReserve(tc, profile, header, literals, units, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			if len(serial.Objects) != 0 || len(parallel.Objects) != 0 || !bytes.Equal(serial.Bytes, parallel.Bytes) {
				t.Fatal("serial/parallel image bytes or image-only ownership differ")
			}
			out, err := ExecuteWindowsMSVCTarget(parallel)
			if err != nil || out.ExitCode != 42 || out.Stdout != "" || out.Stderr != "" {
				t.Fatalf("parallel PE result: %+v %v", out, err)
			}
			t.Logf("serial=parallel PE bytes=%d SHA256=%x; native exit=42", len(parallel.Bytes), sha256.Sum256(parallel.Bytes))
		})
	}
	broken := []string{"#error first_source_failure\n", "#error later_source_failure\n"}
	artifact, err := BuildWindowsMSVCCompilerHostPartsWithStackReserve(tc, LowProfile, header, literals, broken, support, time.Minute, 16<<20)
	if err == nil || !strings.Contains(err.Error(), "compile project source fy_unit_0.c") || len(artifact.Bytes) != 0 || len(artifact.Objects) != 0 {
		t.Fatalf("failed project published output or lost first source: %v", err)
	}
}

func TestWindowsProjectCompileDeadlinePreservesCleanupFailure(t *testing.T) {
	root := t.TempDir()
	cause := errors.New("could not verify process termination")
	cleanup := &windowsJobCleanupError{
		wait:    &exec.ExitError{},
		cleanup: fmt.Errorf("%s: %w", root, cause),
	}
	output := boundedBuffer{limit: 16, truncated: true}
	err := windowsProjectSourceError{
		"fy_unit_0.c", root, windowsProjectCommandResult(context.DeadlineExceeded, cleanup, output, time.Second),
	}
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, cause) {
		t.Fatalf("deadline and cleanup did not both survive compilation: %v", err)
	}
	if !strings.Contains(err.Error(), "exceeded 1s") ||
		!strings.Contains(err.Error(), "clean up execution job: <temporary>") || strings.Contains(err.Error(), root) {
		t.Fatalf("unexpected diagnostic: %v", err)
	}
	var exit *exec.ExitError
	if errors.As(cleanup, &exit) {
		t.Fatal("cleanup failure was classified as a normal tool exit")
	}
	ordinary := windowsProjectCommandResult(context.DeadlineExceeded, errors.New("tool killed"), output, time.Second)
	if ordinary.Error() != "MSVC host command exceeded 1s: context deadline exceeded" {
		t.Fatalf("ordinary deadline changed: %v", ordinary)
	}
}

// The helper is only ever launched by this test through its own context. It
// creates one owned descendant; no process-name enumeration or global kill.
func TestWindowsProjectCompileProcessHelper(t *testing.T) {
	mode := os.Getenv("FAIRYC_PROJECT_COMPILE_HELPER")
	if mode == "" {
		return
	}
	root := os.Getenv("FAIRYC_PROJECT_COMPILE_SCRATCH")
	if err := os.WriteFile(filepath.Join(root, mode+".pid"), []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
		os.Exit(2)
	}
	if mode == "child" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestWindowsProjectCompileProcessHelper$")
	child.Env = replaceWindowsEnvironment(os.Environ(), "FAIRYC_PROJECT_COMPILE_HELPER", "child")
	child.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	if err := child.Run(); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestWindowsProjectCompileFailureAndCancelReapOwnedChildren(t *testing.T) {
	for _, fail := range []bool{true, false} {
		t.Run(fmt.Sprintf("failure-%t", fail), func(t *testing.T) {
			root := t.TempDir()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			env := replaceWindowsEnvironment(os.Environ(), "FAIRYC_PROJECT_COMPILE_HELPER", "parent")
			env = replaceWindowsEnvironment(env, "FAIRYC_PROJECT_COMPILE_SCRATCH", root)
			var handles []syscall.Handle
			defer func() {
				for _, handle := range handles {
					// Cleanup on a failed assertion is restricted to the exact
					// handles opened for this helper's reported parent/child.
					var code uint32
					if syscall.GetExitCodeProcess(handle, &code) == nil && code == 259 {
						_ = syscall.TerminateProcess(handle, 2)
					}
					syscall.CloseHandle(handle)
				}
			}()
			failure := errors.New("source zero failed")
			err := compileWindowsProjectOrdered(ctx, 2, 4, func(commandContext context.Context, index int) error {
				if index == 1 {
					return runWindowsProjectCompileCommand(commandContext, os.Args[0], env, root, 1024, time.Minute,
						"-test.run=^TestWindowsProjectCompileProcessHelper$")
				}
				if index != 0 {
					return fmt.Errorf("started source %d after failure", index)
				}
				for _, mode := range []string{"parent", "child"} {
					var data []byte
					for {
						var readErr error
						data, readErr = os.ReadFile(filepath.Join(root, mode+".pid"))
						if readErr == nil && len(data) != 0 {
							break
						}
						select {
						case <-commandContext.Done():
							return commandContext.Err()
						case <-time.After(10 * time.Millisecond):
						}
					}
					pid, parseErr := strconv.ParseUint(string(data), 10, 32)
					if parseErr != nil {
						return parseErr
					}
					handle, openErr := syscall.OpenProcess(0x100000|0x1000|1, false, uint32(pid))
					if openErr != nil {
						return openErr
					}
					handles = append(handles, handle)
				}
				if fail {
					return failure
				}
				cancel()
				return commandContext.Err()
			})
			want := error(context.Canceled)
			if fail {
				want = failure
			}
			if !errors.Is(err, want) || len(handles) != 2 {
				t.Fatalf("failure/cancel: %v handles=%d", err, len(handles))
			}
			for _, handle := range handles {
				state, err := syscall.WaitForSingleObject(handle, 0)
				if err != nil || state != 0 {
					t.Fatalf("owned process survived return: state=%d %v", state, err)
				}
			}
			// Removing the owned directory is safe only after all child commands
			// and descendants have exited and their Wait has completed.
			if err := os.RemoveAll(root); err != nil {
				t.Fatal("cleanup after joined commands:", err)
			}
		})
	}
}
