package languageserver

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/sys/windows"
)

// Match the disk spelling used by the workspace walk while keeping the URI
// supplied by the client. New files and other kinds of aliases keep their path.
func documentDiskPath(path string) string {
	// Query each name separately so expanding an 8.3 ancestor cannot discard
	// the on-disk casing of its descendants.
	var components []string
	prefix := path
	for parent := filepath.Dir(prefix); parent != prefix; parent = filepath.Dir(prefix) {
		component := filepath.Base(prefix)
		if name, err := windows.UTF16PtrFromString(prefix); err == nil {
			var data windows.Win32finddata
			if handle, err := windows.FindFirstFile(name, &data); err == nil {
				windows.FindClose(handle)
				if diskName := windows.UTF16ToString(data.FileName[:]); strings.EqualFold(diskName, component) {
					component = diskName
				}
			}
		}
		components = append(components, component)
		prefix = parent
	}
	slices.Reverse(components)
	spelling := filepath.Join(append([]string{prefix}, components...)...)
	if spelling == path {
		return path
	}
	original, originalErr := os.Lstat(path)
	onDisk, diskErr := os.Lstat(spelling)
	if originalErr != nil || diskErr != nil || !os.SameFile(original, onDisk) {
		return path
	}
	return spelling
}
