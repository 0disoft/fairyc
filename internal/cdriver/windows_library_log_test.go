//go:build windows

package cdriver

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/pecoff"
)

func TestWindowsLibraryLogCapture(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	const source = "#include <stdio.h>\nint main(void) { printf(\"ok\"); return 0; }\n"
	release, err := BuildWindowsMSVCProbeWithMap(toolchain, OptimizedProfile, source)
	if err != nil {
		t.Fatal(err)
	}
	verbose, err := BuildWindowsMSVCProbeWithLibraryLog(toolchain, OptimizedProfile, source)
	if err != nil {
		t.Fatal(err)
	}
	if len(release.LinkerVerbose) != 0 || len(verbose.LinkerVerbose) == 0 || len(verbose.LinkerVerbose) > 8*1024*1024 || len(verbose.LinkMap) == 0 {
		t.Fatal("bounded library log or map is missing")
	}
	log := strings.ToLower(string(verbose.LinkerVerbose))
	var members []string
	for _, field := range strings.Fields(string(verbose.LinkerVerbose)) {
		if strings.Contains(strings.ToLower(field), ".lib") || strings.Contains(strings.ToLower(field), ".obj") {
			members = append(members, filepath.Base(strings.ReplaceAll(field, "\\", "/")))
			if len(members) == 12 {
				break
			}
		}
	}
	var pathFreeLines []string
	for _, line := range strings.Split(string(verbose.LinkerVerbose), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && len(line) < 120 && !strings.ContainsAny(line, "\\/:") {
			pathFreeLines = append(pathFreeLines, line)
			if len(pathFreeLines) == 15 {
				break
			}
		}
	}
	t.Logf("library log bytes=%d lib=%t obj=%t searching=%d found=%d loaded=%d referenced=%d tokens=%v",
		len(verbose.LinkerVerbose), strings.Contains(log, ".lib"), strings.Contains(log, ".obj"),
		strings.Count(log, "searching"), strings.Count(log, "found"), strings.Count(log, "loaded"),
		strings.Count(log, "referenced"), members)
	t.Logf("path-free linker lines=%v", pathFreeLines)
	if !strings.Contains(log, ".lib") || !strings.Contains(log, ".obj") {
		t.Fatal("MSVC library log omitted archive/member evidence")
	}
	for _, artifact := range []WindowsHostArtifact{release, verbose} {
		outcome, runErr := ExecuteWindowsHost(artifact)
		if runErr != nil || outcome.ExitCode != 0 || outcome.Stdout != "ok" || outcome.Stderr != "" {
			t.Fatalf("library-log build changed execution: %+v %v", outcome, runErr)
		}
	}
	releasePE, err := pecoff.InspectExecutable(release.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	verbosePE, err := pecoff.InspectExecutable(verbose.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(release.Bytes) != len(verbose.Bytes) || len(releasePE.Sections) != len(verbosePE.Sections) {
		t.Fatal("library logging changed executable size or section count")
	}
	t.Logf("EXE bytes=%d bounded library log bytes=%d", len(verbose.Bytes), len(verbose.LinkerVerbose))
}
