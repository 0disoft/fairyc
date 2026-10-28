package candidatehost

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"
)

//go:embed linux_paths.c
var linuxPaths string

//go:embed linux_process.c
var linuxProcess string

var linuxSymbols = regexp.MustCompile(`\b(fairyc_private_linux_[A-Za-z0-9_]+)\s*\(`)

// LinuxSource selects the admitted private path/process adapters. The Linux
// candidate CLI/build bridge is not connected by these internal host slices.
func LinuxSource(header string) (string, error) {
	matches := linuxSymbols.FindAllStringSubmatch(header, -1)
	paths, process := false, false
	for _, match := range matches {
		switch match[1] {
		case "fairyc_private_linux_host_path_v1":
			paths = true
		case "fairyc_private_linux_process_start_v1", "fairyc_private_linux_process_wait_v1", "fairyc_private_linux_process_discard_v1":
			process = true
		default:
			return "", fmt.Errorf("unknown private Linux host symbol %s", match[1])
		}
	}
	if strings.Contains(header, "fairyc_private_windows_") {
		return "", fmt.Errorf("private Windows host symbols cannot enter Linux support")
	}
	if process && paths {
		return linuxProcess + "\n" + linuxPaths, nil
	}
	if process {
		return linuxProcess, nil
	}
	if paths {
		return linuxPaths, nil
	}
	return "", nil
}
