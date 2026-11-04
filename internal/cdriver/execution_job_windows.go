//go:build windows

package cdriver

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Start suspended: assigning an already-running parent to a Job lets its first
// child escape. Cmd owns standard handles and its context; this scope owns the Job.
func runOwnedWindowsProcess(command *exec.Cmd, trapEnvironment string) (string, bool, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return "", false, fmt.Errorf("create execution job: %w", err)
	}
	defer windows.CloseHandle(job)
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return "", false, fmt.Errorf("configure execution job: %w", err)
	}
	var attributes syscall.SysProcAttr
	if command.SysProcAttr != nil {
		attributes = *command.SysProcAttr
	}
	attributes.CreationFlags |= windows.CREATE_SUSPENDED
	attributes.HideWindow = true
	command.SysProcAttr = &attributes
	command.Cancel = func() error {
		err := windows.TerminateJobObject(job, 1)
		// Cancellation may race the suspended process's assignment to the Job.
		killErr := command.Process.Kill()
		if errors.Is(killErr, os.ErrProcessDone) {
			killErr = nil
		}
		return errors.Join(err, killErr)
	}
	start := func(cmd *exec.Cmd) error {
		if err := cmd.Start(); err != nil {
			return err
		}
		process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
		if err == nil {
			err = windows.AssignProcessToJobObject(job, process)
			windows.CloseHandle(process)
		}
		if err == nil {
			err = resumeOwnedWindowsProcess(uint32(cmd.Process.Pid))
		}
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return fmt.Errorf("admit execution process to job: %w", err)
		}
		return nil
	}
	wait := func(cmd *exec.Cmd) error {
		err := cmd.Wait()
		if cleanup := terminateOwnedWindowsJob(job); cleanup != nil {
			// Do not hide a cleanup failure behind an ExitError's ordinary exit status.
			return fmt.Errorf("execution result %v; clean up execution job: %w", err, cleanup)
		}
		return err
	}
	if trapEnvironment != "" {
		return runWindowsProcessWithTrapControl(command, trapEnvironment, windowsTrapControlOptions{Start: start, Wait: wait})
	}
	if err := start(command); err != nil {
		return "", false, err
	}
	return "", false, wait(command)
}

func resumeOwnedWindowsProcess(pid uint32) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	for err = windows.Thread32First(snapshot, &entry); err == nil; err = windows.Thread32Next(snapshot, &entry) {
		if entry.OwnerProcessID != pid {
			continue
		}
		thread, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
		if err != nil {
			return err
		}
		_, err = windows.ResumeThread(thread)
		windows.CloseHandle(thread)
		return err
	}
	if err != nil && !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return err
	}
	return fmt.Errorf("suspended process %d has no primary thread", pid)
}

type executionJobAccounting struct {
	TotalUserTime, TotalKernelTime, PeriodUserTime, PeriodKernelTime     int64
	PageFaultCount, TotalProcesses, ActiveProcesses, TerminatedProcesses uint32
}

func terminateOwnedWindowsJob(job windows.Handle) error {
	if err := windows.TerminateJobObject(job, 1); err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		var accounting executionJobAccounting
		if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation,
			uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil); err != nil {
			return err
		}
		if accounting.ActiveProcesses == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("execution job descendants did not exit")
		}
		time.Sleep(time.Millisecond)
	}
}

// ActiveProcesses can reach zero just before Windows releases the last image
// section (or its scanner handle). Retry only transient access/sharing failures.
func removeOwnedWindowsDirectory(path string) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := os.RemoveAll(path)
		if err == nil || (!errors.Is(err, os.ErrPermission) && !errors.Is(err, windows.ERROR_SHARING_VIOLATION)) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
}
