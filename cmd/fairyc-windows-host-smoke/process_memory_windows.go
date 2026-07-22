//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const processMemoryAccess = 0x0400 | 0x0010 // PROCESS_QUERY_INFORMATION | PROCESS_VM_READ

var getProcessMemoryInfo = syscall.NewLazyDLL("psapi.dll").NewProc("GetProcessMemoryInfo")

type processMemoryCounters struct {
	Size                     uint32
	PageFaultCount           uint32
	PeakWorkingSetSize       uintptr
	WorkingSetSize           uintptr
	QuotaPeakPagedPoolUsage  uintptr
	QuotaPagedPoolUsage      uintptr
	QuotaPeakNonPagedPoolUse uintptr
	QuotaNonPagedPoolUse     uintptr
	PagefileUsage            uintptr
	PeakPagefileUsage        uintptr
}

type processMemoryProbe struct {
	handle syscall.Handle
}

func openProcessMemoryProbe(pid int) (processMemoryProbe, error) {
	handle, err := syscall.OpenProcess(processMemoryAccess, false, uint32(pid))
	if err != nil {
		return processMemoryProbe{}, fmt.Errorf("open process %d for memory counters: %w", pid, err)
	}
	return processMemoryProbe{handle: handle}, nil
}

func (probe processMemoryProbe) snapshot() (uint64, uint64, error) {
	if probe.handle == 0 {
		return 0, 0, fmt.Errorf("process memory probe is closed")
	}
	counters := processMemoryCounters{Size: uint32(unsafe.Sizeof(processMemoryCounters{}))}
	result, _, callErr := getProcessMemoryInfo.Call(
		uintptr(probe.handle),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(counters.Size),
	)
	if result == 0 {
		return 0, 0, fmt.Errorf("query process memory counters: %w", callErr)
	}
	if counters.PeakWorkingSetSize == 0 || counters.PeakPagefileUsage == 0 {
		return 0, 0, fmt.Errorf("process memory counters returned an empty peak")
	}
	return uint64(counters.PeakWorkingSetSize), uint64(counters.PeakPagefileUsage), nil
}

func (probe processMemoryProbe) close() error {
	if probe.handle == 0 {
		return nil
	}
	return syscall.CloseHandle(probe.handle)
}
