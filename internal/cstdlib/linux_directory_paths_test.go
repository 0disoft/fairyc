package cstdlib

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestLinuxDirectoryPathsModel(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC POSIX model")
	}
	fixture, err := os.ReadFile("testdata/linux_directory_paths_model.c")
	if err != nil {
		t.Fatal(err)
	}
	var production strings.Builder
	production.WriteString(stage0TextView)
	for _, source := range []string{stage0LinuxDirectory, stage0LinuxDirectoryPaths} {
		for _, line := range strings.Split(source, "\n") {
			if !strings.HasPrefix(line, "#include") && !strings.HasPrefix(line, "//go:build") {
				production.WriteString(line + "\n")
			}
		}
	}
	runWindowsFilesystemModel(t, strings.Replace(string(fixture), "/* PRODUCTION */", production.String(), 1))
}
