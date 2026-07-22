//go:build !windows

package durablefile

import "os"

func rename(root *os.Root, temporaryPath, finalPath string) error {
	return root.Rename(temporaryPath, finalPath)
}

func syncPublished(root *os.Root, _, directory string) error {
	return syncDirectory(root, directory)
}

func syncDirectory(root *os.Root, path string) error {
	directory, err := root.Open(path)
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}
