//go:build !windows

package languageserver

func documentDiskPath(path string) string { return path }
