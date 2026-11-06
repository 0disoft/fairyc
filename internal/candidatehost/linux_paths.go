package candidatehost

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"
)

//go:embed linux_paths.c
var linuxPaths string

//go:embed linux_clock.c
var linuxClock string

//go:embed linux_process.c
var linuxProcess string

//go:embed linux_capture.c
var linuxCapture string

//go:embed linux_test_input.c
var linuxTestInput string

//go:embed linux_rooted_file.c
var linuxRootedFile string

//go:embed linux_directory.c
var linuxDirectory string

//go:embed linux_package_directory.c
var linuxPackageDirectory string

//go:embed linux_package_file.c
var linuxPackageFile string

//go:embed linux_retention.c
var linuxRetention string

//go:embed linux_workspace.c
var linuxWorkspace string

//go:embed linux_starter.c
var linuxStarter string

//go:embed linux_build_scratch.c
var linuxBuildScratch string

//go:embed linux_gnu.c
var linuxGNU string

var linuxSymbols = regexp.MustCompile(`\b(fairyc_private_linux_[A-Za-z0-9_]+)\s*\(`)

// LinuxSource selects the admitted private path/process/filesystem adapters.
// The native support linker opts into these for Linux file and workspace check
// commands; workspace build/run still require their separate host integration.
func LinuxSource(header string) (string, error) {
	matches := linuxSymbols.FindAllStringSubmatch(header, -1)
	paths, process, capture, rooted, directory, retention := false, false, false, false, false, false
	workspace := false
	buildScratch := false
	testInput := false
	starter := false
	gnu := false
	packageDirectory := false
	packageFile := false
	clock := false
	for _, match := range matches {
		switch match[1] {
		case "fairyc_private_linux_monotonic_ms_v1":
			clock = true
		case "fairyc_private_linux_package_file_info_v1":
			rooted, packageFile = true, true
		case "fairyc_private_linux_package_directory_create_v1", "fairyc_private_linux_package_directory_edit_v1",
			"fairyc_private_linux_package_directory_publish_v1", "fairyc_private_linux_package_directory_erase_v1",
			"fairyc_private_linux_package_directory_clone_v1":
			rooted, packageDirectory = true, true
		case "fairyc_private_linux_gnu_stage_v1", "fairyc_private_linux_gnu_write_v1", "fairyc_private_linux_gnu_read_v1",
			"fairyc_private_linux_gnu_find_v1", "fairyc_private_linux_gnu_run_v1", "fairyc_private_linux_gnu_publish_v1":
			rooted, process, capture, gnu = true, true, true, true
		case "fairyc_private_linux_starter_create_v1":
			rooted, starter = true, true
		case "fairyc_private_linux_test_capture_open_v1":
			testInput, capture, process = true, true, true
		case "fairyc_private_linux_build_scratch_open_v1", "fairyc_private_linux_build_scratch_drop_v1":
			rooted, buildScratch = true, true
		case "fairyc_private_linux_workspace_path_kind_v1":
			rooted, workspace = true, true
		case "fairyc_private_linux_host_path_v1":
			paths = true
		case "fairyc_private_linux_process_start_v1", "fairyc_private_linux_process_wait_v1", "fairyc_private_linux_process_discard_v1", "fairyc_private_linux_process_wait_for_v1":
			process = true
		case "fairyc_private_linux_run_capture_open_v1", "fairyc_private_linux_run_capture_receipt_v1", "fairyc_private_linux_capture_open_v1", "fairyc_private_linux_capture_drop_v1", "fairyc_private_linux_capture_collect_v1":
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
		case "fairyc_private_linux_root_file_erase_if_v1":
			rooted, retention = true, true
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
	if testInput {
		source += "\n" + linuxTestInput
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
	if packageDirectory {
		source += "\n" + linuxPackageDirectory
	}
	if packageFile {
		source += "\n" + linuxPackageFile
	}
	if retention {
		source += "\n" + linuxRetention
	}
	if workspace {
		source += "\n" + linuxWorkspace
	}
	if buildScratch {
		source += "\n" + linuxBuildScratch
	}
	if starter {
		source += "\n" + linuxStarter
	}
	if gnu {
		source += "\n" + linuxGNU
	}
	if clock {
		if source != "" {
			source += "\n"
		}
		source += linuxClock
	}
	return source, nil
}
