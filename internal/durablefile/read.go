package durablefile

import (
	"fmt"
	"os"
)

// OpenRegular opens a detached, caller-owned regular-file handle below root.
// Check both the path and opened identity; Linux opens nonblocking so replacing
// a checked file with a FIFO cannot block before the second check.
func OpenRegular(root *os.Root, name string) (*os.File, error) {
	return openRegular(root, name, openReadFile)
}

func openRegular(root *os.Root, name string, open func(*os.Root, string) (*os.File, error)) (*os.File, error) {
	if root == nil || !validRelativePath(name) {
		return nil, fmt.Errorf("regular file requires an opened root and canonical path")
	}
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, fmt.Errorf("durable file path is not a regular file")
	}
	file, err := open(root, name)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("durable file changed during open: %w", err)
	}
	if !after.Mode().IsRegular() || !os.SameFile(before, after) {
		file.Close()
		return nil, fmt.Errorf("durable file changed during open")
	}
	return file, nil
}
