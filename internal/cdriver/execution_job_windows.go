//go:build windows

package cdriver

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
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
	// Let admission finish before cancellation inspects the job. Cmd.Start
	// can launch its context watcher before the suspended child is assigned.
	admissionDone := make(chan struct{})
	admitted := false
	cleanup := sync.OnceValue(func() error { return terminateOwnedWindowsJob(job) })
	command.Cancel = func() error {
		<-admissionDone
		if admitted {
			if err := cleanup(); err == nil {
				return nil
			}
		}
		return command.Process.Kill()
	}
	start := func(cmd *exec.Cmd) error {
		if err := cmd.Start(); err != nil {
			return err
		}
		process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
		if err == nil {
			err = windows.AssignProcessToJobObject(job, process)
			admitted = err == nil
			windows.CloseHandle(process)
		}
		if err == nil {
			err = resumeOwnedWindowsProcess(uint32(cmd.Process.Pid))
		}
		close(admissionDone)
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return fmt.Errorf("admit execution process to job: %w", err)
		}
		return nil
	}
	wait := func(cmd *exec.Cmd) error {
		err := cmd.Wait()
		if cleanupErr := cleanup(); cleanupErr != nil {
			// Do not hide a cleanup failure behind an ExitError's ordinary exit status.
			return &windowsJobCleanupError{wait: err, cleanup: cleanupErr}
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

const windowsJobCleanupTimeout = 5 * time.Second

func terminateOwnedWindowsJob(job windows.Handle) error {
	deadline := time.Now().Add(windowsJobCleanupTimeout)
	// Block new runnable descendants before taking the process-handle snapshot.
	// An existing member already occupies the one available slot.
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS
	limits.BasicLimitInformation.ActiveProcessLimit = 1
	_, limitErr := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)))
	handles, snapshotErr := openWindowsJobProcesses(job)
	defer func() {
		for _, handle := range handles {
			windows.CloseHandle(handle)
		}
	}()
	// Termination can clear job membership before the process objects signal.
	// Keep the handles across that call and wait on each under one deadline.
	if err := windows.TerminateJobObject(job, 1); err != nil {
		return errors.Join(limitErr, snapshotErr, err)
	}
	for _, handle := range handles {
		remaining := max(0, time.Until(deadline))
		event, err := windows.WaitForSingleObject(handle, uint32(remaining.Round(time.Millisecond)/time.Millisecond))
		if err != nil {
			return errors.Join(limitErr, snapshotErr, err)
		}
		if event != windows.WAIT_OBJECT_0 {
			return errors.Join(limitErr, snapshotErr, fmt.Errorf("execution job process did not exit: wait status %d", event))
		}
	}
	return errors.Join(limitErr, snapshotErr)
}

func openWindowsJobProcesses(job windows.Handle) ([]windows.Handle, error) {
	// JOBOBJECT_BASIC_PROCESS_ID_LIST has two DWORDs followed by ULONG_PTRs.
	const headerWords = int(8 / unsafe.Sizeof(uintptr(0)))
	buffer := make([]uintptr, headerWords+16)
	for {
		err := windows.QueryInformationJobObject(job, windows.JobObjectBasicProcessIdList,
			uintptr(unsafe.Pointer(&buffer[0])), uint32(uintptr(len(buffer))*unsafe.Sizeof(buffer[0])), nil)
		counts := (*[2]uint32)(unsafe.Pointer(&buffer[0]))
		if errors.Is(err, windows.ERROR_MORE_DATA) && counts[0] > uint32(len(buffer)-headerWords) {
			buffer = make([]uintptr, headerWords+int(counts[0]))
			continue
		}
		if err != nil {
			return nil, err
		}
		var handles []windows.Handle
		for _, pid := range buffer[headerWords : headerWords+int(counts[1])] {
			handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
			if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
				continue // Exited between the job query and OpenProcess.
			}
			if err != nil {
				return handles, err
			}
			handles = append(handles, handle)
		}
		return handles, nil
	}
}

// A scanner can retain a handle after process exit. Retry only transient
// access or sharing failures.
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
