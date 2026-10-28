package candidatehost

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"
)

//go:embed linux_paths.c
var linuxPaths string

var linuxSymbols = regexp.MustCompile(`\b(fairyc_private_linux_[A-Za-z0-9_]+)\s*\(`)

// LinuxSource selects only the admitted private path adapter. The Linux
// candidate CLI/build bridge is not connected by this initial host slice.
func LinuxSource(header string) (string, error) {
	matches := linuxSymbols.FindAllStringSubmatch(header, -1)
	for _, match := range matches {
		if match[1] != "fairyc_private_linux_host_path_v1" {
			return "", fmt.Errorf("unknown private Linux host symbol %s", match[1])
		}
	}
	if strings.Contains(header, "fairyc_private_windows_") {
		return "", fmt.Errorf("private Windows host symbols cannot enter Linux support")
	}
	if len(matches) == 0 {
		return "", nil
	}
	return linuxPaths, nil
}
