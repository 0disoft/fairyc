package nativesupport

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

func TestLinuxCandidateHostSelection(t *testing.T) {
	symbols := []string{"fairyc_private_linux_root_file_read_v1", "fy_std_system_heap_allocate_v0"}
	ordinary, err := Source(target.X8664LinuxGNU, symbols, Options{})
	if err != nil || strings.Contains(ordinary, "fairyc_private_linux_root_file_read_v1") {
		t.Fatalf("implicit private linkage: %v", err)
	}
	linked, err := Source(target.X8664LinuxGNU, symbols, Options{CandidateHost: true})
	if err != nil || !strings.Contains(linked, "fairyc_private_linux_root_file_read_v1") || strings.Contains(linked, "#include <windows.h>") {
		t.Fatalf("Linux candidate support selection: %v", err)
	}
	feature := strings.Index(linked, "#define _GNU_SOURCE")
	if feature < 0 || feature > strings.Index(linked, "#include") {
		t.Fatal("GNU feature selection must precede runtime libc headers")
	}
	for _, symbol := range []string{"fairyc_private_linux_unknown_v1", "fairyc_private_windows_host_path_v1"} {
		if _, err := Source(target.X8664LinuxGNU, []string{symbol}, Options{CandidateHost: true}); err == nil {
			t.Fatalf("unsupported candidate symbol admitted: %s", symbol)
		}
	}
}
