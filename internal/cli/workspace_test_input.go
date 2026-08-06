package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const stableNativeTestInputMaximum = 64 * 1024

func readStableNativeTestInput(sourceRoot, portablePath string) ([]byte, error) {
	if portablePath == "." || !fs.ValidPath(portablePath) || strings.Contains(portablePath, `\`) {
		return nil, fmt.Errorf("native test stdin file %q is not a canonical portable workspace file", portablePath)
	}
	root, err := os.OpenRoot(sourceRoot)
	if err != nil {
		return nil, fmt.Errorf("open native test stdin workspace root")
	}
	defer root.Close()
	file, err := root.Open(filepath.FromSlash(portablePath))
	if err != nil {
		return nil, fmt.Errorf("open native test stdin file %q", portablePath)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("native test stdin file %q is not a regular file", portablePath)
	}
	if info.Size() > stableNativeTestInputMaximum {
		return nil, fmt.Errorf("native test stdin file %q exceeds %d bytes", portablePath, stableNativeTestInputMaximum)
	}
	contents, err := io.ReadAll(io.LimitReader(file, stableNativeTestInputMaximum+1))
	if err != nil {
		return nil, fmt.Errorf("read native test stdin file %q", portablePath)
	}
	if len(contents) > stableNativeTestInputMaximum {
		return nil, fmt.Errorf("native test stdin file %q exceeds %d bytes", portablePath, stableNativeTestInputMaximum)
	}
	return contents, nil
}
