package durablefile

import (
	"errors"
	"fmt"
	"os"
)

var (
	// ErrNotRegularFile classifies a path that cannot be admitted as an ordinary file.
	ErrNotRegularFile = errors.New("durable file path is not a regular file")
	// ErrNotDirectory classifies a path that cannot be admitted as a directory.
	ErrNotDirectory = errors.New("durable file path is not a directory")
	// ErrIdentityChanged classifies a path that changed between inspection and open.
	ErrIdentityChanged = errors.New("durable file changed during open")
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
		return nil, ErrNotRegularFile
	}
	file, err := open(root, name)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil {
		return nil, errors.Join(ErrIdentityChanged, file.Close(), err)
	}
	if !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, errors.Join(ErrIdentityChanged, file.Close())
	}
	return file, nil
}

// OpenDirectory opens a detached, caller-owned directory handle below root.
// Check both the path and opened identity; Linux opens nonblocking with
// O_DIRECTORY so replacing a checked directory with a FIFO cannot block.
func OpenDirectory(root *os.Root, name string) (*os.File, error) {
	return openDirectory(root, name, openReadDirectory)
}

func openDirectory(root *os.Root, name string, open func(*os.Root, string) (*os.File, error)) (*os.File, error) {
	if root == nil || !validRelativePath(name) {
		return nil, fmt.Errorf("directory requires an opened root and canonical path")
	}
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.IsDir() {
		return nil, ErrNotDirectory
	}
	directory, err := open(root, name)
	if err != nil {
		return nil, err
	}
	after, err := directory.Stat()
	if err != nil {
		return nil, errors.Join(ErrIdentityChanged, directory.Close(), err)
	}
	if !after.IsDir() || !os.SameFile(before, after) {
		return nil, errors.Join(ErrIdentityChanged, directory.Close())
	}
	return directory, nil
}
