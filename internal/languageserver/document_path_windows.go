package languageserver

import (
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

// Match the disk spelling used by the workspace walk while keeping the URI
// supplied by the client. New files and other kinds of aliases keep their path.
func documentDiskPath(path string) string {
	name, err := windows.UTF16FromString(path)
	if err != nil {
		return path
	}
	buffer := make([]uint16, len(name))
	n, err := windows.GetLongPathName(&name[0], &buffer[0], uint32(len(buffer)))
	if err != nil || n >= uint32(len(buffer)) {
		return path
	}
	spelling := windows.UTF16ToString(buffer[:n])
	if spelling == path || !strings.EqualFold(spelling, path) {
		return path
	}
	original, originalErr := os.Lstat(path)
	onDisk, diskErr := os.Lstat(spelling)
	if originalErr != nil || diskErr != nil || !os.SameFile(original, onDisk) {
		return path
	}
	return spelling
}
