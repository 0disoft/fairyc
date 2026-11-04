//go:build windows

package cdriver

import (
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

	"golang.org/x/sys/windows"
)

func TestWindowsOwnedJobHelper(t *testing.T) {
	mode := os.Getenv("FAIRYC_OWNED_JOB_MODE")
	if mode == "" {
		return
	}
	receipt := os.Getenv("FAIRYC_OWNED_JOB_RECEIPT")
	if mode == "child" {
		if err := os.WriteFile(receipt, []byte(fmt.Sprintf("%d\n%s", os.Getpid(), filepath.Dir(os.Args[0]))), 0600); err != nil {
			os.Exit(2)
		}
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestWindowsOwnedJobHelper$")
	child.Env = replaceWindowsEnvironment(os.Environ(), "FAIRYC_OWNED_JOB_MODE", "child")
	child.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := child.Start(); err != nil {
		os.Exit(2)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(receipt); err == nil {
			break
		}
		if time.Now().After(deadline) {
			os.Exit(2)
		}
		time.Sleep(time.Millisecond)
	}
	if mode == "timeout" {
		time.Sleep(30 * time.Second)
	}
	os.Exit(0)
}

func TestWindowsOwnedJobReapsLockedImage(t *testing.T) {
	image, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, trap := range []bool{false, true} {
		for _, mode := range []string{"success", "timeout"} {
			t.Run(fmt.Sprintf("trap=%t/%s", trap, mode), func(t *testing.T) {
				receipt := filepath.Join(t.TempDir(), "child.txt")
				t.Setenv("FAIRYC_OWNED_JOB_MODE", mode)
				t.Setenv("FAIRYC_OWNED_JOB_RECEIPT", receipt)
				_, err := executeWindowsMSVCArtifactWithInputArgsAndTimeout(WindowsHostArtifact{Bytes: image, TrapControl: trap},
					"owned job helper", nil, []string{"-test.run=^TestWindowsOwnedJobHelper$"}, 3*time.Second)
				data, readErr := os.ReadFile(receipt)
				if readErr != nil {
					t.Fatalf("child did not report: %v (execution %v)", readErr, err)
				}
				parts := strings.SplitN(string(data), "\n", 2)
				if len(parts) != 2 {
					t.Fatal("invalid child receipt")
				}
				pid, parseErr := strconv.Atoi(parts[0])
				if parseErr != nil {
					t.Fatal(parseErr)
				}
				process, openErr := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_TERMINATE, false, uint32(pid))
				if openErr == nil {
					defer windows.CloseHandle(process)
					defer windows.TerminateProcess(process, 1)
					var status uint32
					if err := windows.GetExitCodeProcess(process, &status); err != nil || status == 259 {
						t.Fatalf("descendant still active: status %d, %v", status, err)
					}
				} else if !errors.Is(openErr, windows.ERROR_INVALID_PARAMETER) {
					t.Fatal(openErr)
				}
				if mode == "success" && err != nil {
					t.Fatal(err)
				}
				if mode == "timeout" && (err == nil || !strings.Contains(err.Error(), "exceeded 3s")) {
					t.Fatalf("timeout lost: %v", err)
				}
				if _, err := os.Stat(parts[1]); !os.IsNotExist(err) {
					t.Fatalf("owned execution directory retained: %v", err)
				}
			})
		}
	}
}
