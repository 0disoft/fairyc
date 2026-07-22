//go:build windows

package packagecache

import "os"

func renameCacheGeneration(root *os.Root, temporaryPath, finalPath string) error {
	return root.Rename(temporaryPath, finalPath)
}

// Windows has no documented directory-handle flush contract. The temporary file is flushed before
// publication; after the handle-relative rename, flush the published file as the strongest bounded
// operation that preserves the opened-root containment guarantee.
func syncPublishedCacheGeneration(root *os.Root, finalPath, _ string) error {
	file, err := root.OpenFile(finalPath, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func syncCacheDirectory(*os.Root, string) error {
	return nil
}
