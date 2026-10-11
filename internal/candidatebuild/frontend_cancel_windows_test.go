//go:build windows

package candidatebuild

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func init() {
	mode := os.Getenv("FAIRYC_FRONTEND_CANCEL_TEST")
	if mode == "" {
		return
	}
	if mode == "child" {
		if err := os.WriteFile(os.Getenv("FAIRYC_FRONTEND_CANCEL_PID"), []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
			os.Exit(2)
		}
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	child := exec.Command(os.Args[0])
	child.Env = append(os.Environ(), "FAIRYC_FRONTEND_CANCEL_TEST=child")
	child.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := child.Run(); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestFrontendDeadlineTerminatesChildAndPreservesImage(t *testing.T) {
	root := t.TempDir()
	pidPath := filepath.Join(root, "child.pid")
	t.Setenv("FAIRYC_FRONTEND_CANCEL_TEST", "parent")
	t.Setenv("FAIRYC_FRONTEND_CANCEL_PID", pidPath)
	output := filepath.Join(root, "previous.exe")
	if err := os.WriteFile(output, []byte("previous-image"), 0700); err != nil {
		t.Fatal(err)
	}
	_, err := Build(Options{Candidate: os.Args[0], Manifest: "ignored", Output: output, Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), "candidate frontend timeout") {
		t.Fatalf("frontend deadline: %v", err)
	}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("child did not start: %v", err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatal(err)
	}
	handle, err := syscall.OpenProcess(0x1000, false, uint32(pid))
	if err == nil {
		defer syscall.CloseHandle(handle)
		var code uint32
		if err := syscall.GetExitCodeProcess(handle, &code); err != nil {
			t.Fatal(err)
		}
		if code == 259 {
			if cleanup, err := syscall.OpenProcess(1, false, uint32(pid)); err == nil {
				_ = syscall.TerminateProcess(cleanup, 2)
				syscall.CloseHandle(cleanup)
			}
			t.Fatal("candidate child survived the frontend deadline")
		}
	} else if err != syscall.Errno(87) {
		t.Fatalf("cannot inspect child: %v", err)
	}
	data, err = os.ReadFile(output)
	if err != nil || string(data) != "previous-image" {
		t.Fatalf("frontend timeout changed previous image: %v", err)
	}
}
