package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/durablefile"
)

const stableNativeTestInputMaximum = 64 * 1024

func readStableNativeTestInput(sourceRoot, portablePath string) (contents []byte, err error) {
	if portablePath == "." || !fs.ValidPath(portablePath) || strings.Contains(portablePath, `\`) {
		return nil, fmt.Errorf("native test stdin file %q is not a canonical portable workspace file", portablePath)
	}
	root, err := os.OpenRoot(sourceRoot)
	if err != nil {
		return nil, fmt.Errorf("open native test stdin workspace root")
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close native test stdin workspace root: %w", closeErr))
		}
	}()
	file, err := durablefile.OpenRegular(root, filepath.FromSlash(portablePath))
	if err != nil {
		if errors.Is(err, durablefile.ErrNotRegularFile) || errors.Is(err, durablefile.ErrIdentityChanged) {
			return nil, fmt.Errorf("native test stdin file %q is not a regular file", portablePath)
		}
		return nil, fmt.Errorf("open native test stdin file %q", portablePath)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close native test stdin file %q: %w", portablePath, closeErr))
		}
	}()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("native test stdin file %q is not a regular file", portablePath)
	}
	if info.Size() > stableNativeTestInputMaximum {
		return nil, fmt.Errorf("native test stdin file %q exceeds %d bytes", portablePath, stableNativeTestInputMaximum)
	}
	contents, err = io.ReadAll(io.LimitReader(file, stableNativeTestInputMaximum+1))
	if err != nil {
		return nil, fmt.Errorf("read native test stdin file %q", portablePath)
	}
	if len(contents) > stableNativeTestInputMaximum {
		return nil, fmt.Errorf("native test stdin file %q exceeds %d bytes", portablePath, stableNativeTestInputMaximum)
	}
	return contents, nil
}
