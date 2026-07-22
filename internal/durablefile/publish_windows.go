//go:build windows

package durablefile

import "os"

func rename(root *os.Root, temporaryPath, finalPath string) error {
	return root.Rename(temporaryPath, finalPath)
}

// Windows has no documented directory-handle flush contract. Flush the published file after the
// handle-relative rename as the strongest bounded operation that preserves opened-root containment.
func syncPublished(root *os.Root, finalPath, _ string) error {
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

func syncDirectory(*os.Root, string) error {
	return nil
}
