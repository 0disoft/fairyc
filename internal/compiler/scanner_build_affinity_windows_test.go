package compiler_test

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Process affinity is inherited by cl.exe and the linker it starts.
func pinScannerBuildLowestAllowedCPU() (string, func() error, error) {
	kernel := syscall.NewLazyDLL("kernel32.dll")
	current := kernel.NewProc("GetCurrentProcess")
	getMask := kernel.NewProc("GetProcessAffinityMask")
	setMask := kernel.NewProc("SetProcessAffinityMask")
	handle, _, _ := current.Call()
	var original, system uintptr
	if ok, _, callErr := getMask.Call(handle, uintptr(unsafe.Pointer(&original)), uintptr(unsafe.Pointer(&system))); ok == 0 {
		return "", nil, fmt.Errorf("read process affinity: %w", callErr)
	}
	if original == 0 || original&system != original {
		return "", nil, fmt.Errorf("invalid process affinity mask %#x within %#x", original, system)
	}
	lowest := original & -original
	if ok, _, callErr := setMask.Call(handle, lowest); ok == 0 {
		return "", nil, fmt.Errorf("pin process affinity: %w", callErr)
	}
	restore := func() error {
		if ok, _, callErr := setMask.Call(handle, original); ok == 0 {
			return fmt.Errorf("restore process affinity: %w", callErr)
		}
		return nil
	}
	return fmt.Sprintf("%#x", lowest), restore, nil
}
