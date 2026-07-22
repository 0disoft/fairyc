//go:build !windows

package packagecache

import "os"

func renameCacheGeneration(temporaryPath, finalPath string) error {
	return os.Rename(temporaryPath, finalPath)
}

func syncCacheDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}
