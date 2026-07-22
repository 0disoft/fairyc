//go:build windows

package packagecache

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const (
	moveFileReplaceExisting = 0x1
	moveFileWriteThrough    = 0x8
)

var moveFileExW = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

func renameCacheGeneration(temporaryPath, finalPath string) error {
	from, err := windowsExtendedPath(temporaryPath)
	if err != nil {
		return fmt.Errorf("resolve package cache temporary path: %w", err)
	}
	to, err := windowsExtendedPath(finalPath)
	if err != nil {
		return fmt.Errorf("resolve package cache generation path: %w", err)
	}
	fromUTF16, err := syscall.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	toUTF16, err := syscall.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	result, _, callErr := moveFileExW.Call(
		uintptr(unsafe.Pointer(fromUTF16)),
		uintptr(unsafe.Pointer(toUTF16)),
		uintptr(moveFileReplaceExisting|moveFileWriteThrough),
	)
	if result == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return syscall.EINVAL
	}
	return nil
}

// MoveFileExW with MOVEFILE_WRITE_THROUGH owns publication durability on Windows. Opening and
// flushing a directory is not a documented Win32 directory-handle operation.
func syncCacheDirectory(string) error {
	return nil
}

func windowsExtendedPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)
	if strings.HasPrefix(absolute, `\\?\`) {
		return absolute, nil
	}
	if strings.HasPrefix(absolute, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(absolute, `\\`), nil
	}
	return `\\?\` + absolute, nil
}
