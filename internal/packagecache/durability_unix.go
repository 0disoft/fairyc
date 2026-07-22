//go:build !windows

package packagecache

import "os"

func renameCacheGeneration(root *os.Root, temporaryPath, finalPath string) error {
	return root.Rename(temporaryPath, finalPath)
}

func syncPublishedCacheGeneration(root *os.Root, _, directory string) error {
	return syncCacheDirectory(root, directory)
}

func syncCacheDirectory(root *os.Root, path string) error {
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
