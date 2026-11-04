package workspaceloader

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/durablefile"
)

func openWorkspaceRoot(name string) (*os.Root, error) {
	before, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("workspace root is not a real directory")
	}
	root, err := os.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	opened, err := root.Stat(".")
	after, pathErr := os.Lstat(name)
	if err != nil || pathErr != nil || after.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, opened) || !os.SameFile(before, after) {
		root.Close()
		return nil, fmt.Errorf("workspace root changed while opening")
	}
	return root, nil
}

func relativeToRoot(root *os.Root, absolute string) (string, error) {
	if root == nil {
		return "", fmt.Errorf("workspace filesystem boundary is not initialized")
	}
	relative, err := filepath.Rel(root.Name(), absolute)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("workspace path escapes opened root")
	}
	return relative, nil
}

func openSourceBoundary(budget *limits, absolute string) (*os.Root, error) {
	if budget.root == nil {
		return openWorkspaceRoot(absolute)
	}
	return openContainedDirectory(budget.root, absolute)
}

func openContainedDirectory(root *os.Root, absolute string) (*os.Root, error) {
	name, err := relativeToRoot(root, absolute)
	if err != nil {
		return nil, err
	}
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("workspace directory is not a real directory")
	}
	child, err := root.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	opened, err := child.Stat(".")
	after, pathErr := root.Lstat(name)
	if err != nil || pathErr != nil || after.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, opened) || !os.SameFile(before, after) {
		child.Close()
		return nil, fmt.Errorf("workspace directory changed while opening")
	}
	return child, nil
}

func readBoundedRootFile(root *os.Root, name string, maximum int) ([]byte, error) {
	file, err := durablefile.OpenRegular(root, name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > int64(maximum) {
		return nil, fmt.Errorf("file exceeds %d-byte implementation guard", maximum)
	}
	contents, err := io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil {
		return nil, err
	}
	if len(contents) > maximum {
		return nil, fmt.Errorf("file exceeds %d-byte implementation guard", maximum)
	}
	return contents, nil
}
