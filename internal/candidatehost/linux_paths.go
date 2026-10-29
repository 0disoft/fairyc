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

//go:embed linux_capture.c
var linuxCapture string

//go:embed linux_rooted_file.c
var linuxRootedFile string

//go:embed linux_directory.c
var linuxDirectory string

var linuxSymbols = regexp.MustCompile(`\b(fairyc_private_linux_[A-Za-z0-9_]+)\s*\(`)

// LinuxSource selects the admitted private path/process/filesystem adapters. The Linux
// candidate CLI/build bridge is not connected by these internal host slices.
func LinuxSource(header string) (string, error) {
	matches := linuxSymbols.FindAllStringSubmatch(header, -1)
	paths, process, capture, rooted, directory := false, false, false, false, false
	for _, match := range matches {
		switch match[1] {
		case "fairyc_private_linux_host_path_v1":
			paths = true
		case "fairyc_private_linux_process_start_v1", "fairyc_private_linux_process_wait_v1", "fairyc_private_linux_process_discard_v1", "fairyc_private_linux_process_wait_for_v1":
			process = true
		case "fairyc_private_linux_capture_open_v1", "fairyc_private_linux_capture_drop_v1", "fairyc_private_linux_capture_collect_v1":
			capture = true
			process = true
		case "fairyc_private_linux_directory_open_v1", "fairyc_private_linux_root_directory_open_v1",
			"fairyc_private_linux_root_file_open_v1", "fairyc_private_linux_root_file_info_v1",
			"fairyc_private_linux_root_file_read_v1", "fairyc_private_linux_root_file_close_v1",
			"fairyc_private_linux_root_file_create_v1", "fairyc_private_linux_root_file_write_v1",
			"fairyc_private_linux_root_file_sync_v1", "fairyc_private_linux_root_file_publish_new_v1":
			rooted = true
		case "fairyc_private_linux_directory_scan_open_v1", "fairyc_private_linux_directory_scan_next_v1", "fairyc_private_linux_directory_scan_close_v1":
			rooted, directory = true, true
		default:
			return "", fmt.Errorf("unknown private Linux host symbol %s", match[1])
		}
	}
	if strings.Contains(header, "fairyc_private_windows_") {
		return "", fmt.Errorf("private Windows host symbols cannot enter Linux support")
	}
	var source string
	if process {
		source = linuxProcess
	}
	if capture {
		source += "\n" + linuxCapture
	}
	if paths {
		if source != "" {
			source += "\n"
		}
		source += linuxPaths
	}
	if rooted {
		// GNU feature selection must precede every POSIX header in the bundle.
		if source != "" {
			source = linuxRootedFile + "\n" + source
		} else {
			source = linuxRootedFile
		}
	}
	if directory {
		source += "\n" + linuxDirectory
	}
	return source, nil
}
