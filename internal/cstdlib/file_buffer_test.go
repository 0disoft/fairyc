package cstdlib

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

// Observe real reads while retaining the production adapters and reserve/commit algorithm.
func fileBufferProbe(t *testing.T, windows bool) string {
	t.Helper()
	symbols := []string{"fy_std_file_read_append_v0"}
	implementation, err := SourceForSymbols(symbols)
	prefix := "#define _POSIX_C_SOURCE 200809L\n#include <unistd.h>\nstatic ssize_t fy_probe_read(int, void *, size_t);\n"
	hook := `
static size_t fy_probe_reads = 0;
static int fy_probe_interrupt = 0;
static ssize_t fy_probe_read(int fd, void *data, size_t length) {
    fy_probe_reads++;
    if (fy_probe_interrupt) { fy_probe_interrupt = 0; errno = EINTR; return -1; }
    return read(fd, data, length);
}
`
	if windows {
		implementation, err = WindowsMSVCSourceForSymbols(symbols)
		prefix = "#include <windows.h>\nstatic BOOL WINAPI fy_probe_read_file(HANDLE, LPVOID, DWORD, LPDWORD, LPOVERLAPPED);\n"
		hook = `
static size_t fy_probe_reads = 0;
static BOOL WINAPI fy_probe_read_file(HANDLE file, LPVOID data, DWORD size, LPDWORD done, LPOVERLAPPED overlapped) {
    fy_probe_reads++;
    return ReadFile(file, data, size, done, overlapped);
}
`
		implementation = strings.ReplaceAll(implementation, "ReadFile(", "fy_probe_read_file(")
	} else {
		const read = "count = read(file->descriptor, buffer, length);"
		if strings.Count(implementation, read) != 1 {
			t.Fatal("Linux file read seam changed")
		}
		implementation = strings.Replace(implementation, read, "count = fy_probe_read(file->descriptor, buffer, length);", 1)
	}
	if err != nil {
		t.Fatal(err)
	}
	implementation, err = systemHeapProbeImplementationSource("File Buffer", implementation)
	if err != nil {
		t.Fatal(err)
	}
	probe, err := os.ReadFile("stage0_file_buffer_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	return prefix + implementation + hook + string(probe)
}

func TestFileBufferNative(t *testing.T) {
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "linux" && runtime.GOOS != "windows") {
		t.Skip("requires Linux GNU or Windows MSVC on amd64")
	}
	compilers := []string{"gcc", "clang"}
	if runtime.GOOS == "windows" {
		compilers = []string{"msvc"}
	}
	for _, cc := range compilers {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(cc+"/"+profile.Name, func(t *testing.T) {
				source := fileBufferProbe(t, cc == "msvc")
				if cc == "msvc" {
					tc, err := cdriver.ResolveWindowsHostMSVC()
					if err != nil {
						t.Fatal(err)
					}
					artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, source)
					if err != nil {
						t.Fatal(err)
					}
					out, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
					if err != nil || out.ExitCode != 0 || out.Stdout != "" || out.Stderr != "" || out.TrapControl != "" {
						t.Fatalf("File Buffer: %+v %v", out, err)
					}
				} else {
					tc, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
					if err != nil {
						t.Fatal(err)
					}
					artifact, err := cdriver.Build(tc, profile, source, "typedef int fy_file_buffer_runtime;\n")
					if err != nil {
						t.Fatal(err)
					}
					out, err := cdriver.ExecuteLinuxTarget(artifact)
					if err != nil || out.ExitCode != 0 || out.Stdout != "" || out.Stderr != "" || out.TrapCategory != 0 {
						t.Fatalf("File Buffer: %+v %v", out, err)
					}
				}
			})
		}
	}
}
