// Package candidatehost embeds private native ABI adapters for Fairy host code.
// No frontend, command policy, process sequence or deadline policy lives here.
package candidatehost

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"
)

//go:embed windows_process.c
var windowsProcess string

//go:embed windows_rooted_file.c
var windowsRootedFile string

var symbols = regexp.MustCompile(`\bfairyc_private_windows_[A-Za-z0-9_]+\s*\(`)
var admitted = map[string]bool{
	"fairyc_private_windows_input_file_v1":   true,
	"fairyc_private_windows_current_path_v1": true,
	"fairyc_private_windows_temp_path_v1":    true,
	"fairyc_private_windows_image_path_v1":   true,
	"fairyc_private_windows_ticks_v1":        true,
	"fairyc_private_windows_create_env_v1":   true, "fairyc_private_windows_pipe_v1": true,
	"fairyc_private_windows_inherit_v1": true, "fairyc_private_windows_peek_v1": true,
	"fairyc_private_windows_read_v1": true, "fairyc_private_windows_handle_value_v1": true,
	"fairyc_private_windows_environment_v1": true, "fairyc_private_windows_free_environment_v1": true,
	"fairyc_private_windows_environment_word_v1": true, "fairyc_private_windows_compare_name_v1": true,
	"fairyc_private_windows_open_v1": true, "fairyc_private_windows_handle_count_v1": true,
	"fairyc_private_windows_job_v1": true, "fairyc_private_windows_job_info_v1": true,
	"fairyc_private_windows_create_v1": true, "fairyc_private_windows_assign_v1": true,
	"fairyc_private_windows_resume_v1": true, "fairyc_private_windows_wait_v1": true,
	"fairyc_private_windows_exit_v1": true, "fairyc_private_windows_kill_job_v1": true,
	"fairyc_private_windows_kill_process_v1": true, "fairyc_private_windows_close_v1": true,
	"fairyc_private_windows_std_handle_v1": true, "fairyc_private_windows_error_v1": true,
	"fairyc_private_windows_startup_size_v1": true, "fairyc_private_windows_job_size_v1": true,
	"fairyc_private_windows_directory_open_v1": true, "fairyc_private_windows_root_file_open_v1": true,
	"fairyc_private_windows_root_file_size_v1": true, "fairyc_private_windows_root_file_write_v1": true,
	"fairyc_private_windows_root_file_read_v1":  true,
	"fairyc_private_windows_root_file_close_v1": true,
	"fairyc_private_windows_root_file_sync_v1":  true, "fairyc_private_windows_root_file_delete_v1": true,
	"fairyc_private_windows_root_file_rename_v1":      true,
	"fairyc_private_windows_root_file_publish_new_v1": true,
	"fairyc_private_windows_root_file_time_v1":        true,
	"fairyc_private_windows_root_file_info_v1":        true, "fairyc_private_windows_root_directory_query_v1": true,
}

func WindowsSource(header string) (string, error) {
	matches := symbols.FindAllString(header, -1)
	if len(matches) == 0 {
		return "", nil
	}
	rooted := false
	for _, match := range matches {
		name := match[:len(match)-1]
		for len(name) > 0 && (name[len(name)-1] == ' ' || name[len(name)-1] == '\t' || name[len(name)-1] == '\n' || name[len(name)-1] == '\r') {
			name = name[:len(name)-1]
		}
		if !admitted[name] {
			return "", fmt.Errorf("unknown private Windows host symbol %s", name)
		}
		if name == "fairyc_private_windows_directory_open_v1" || strings.HasPrefix(name, "fairyc_private_windows_root_file_") || strings.HasPrefix(name, "fairyc_private_windows_root_directory_") {
			rooted = true
		}
	}
	if rooted {
		return windowsProcess + "\n" + windowsRootedFile, nil
	}
	return windowsProcess, nil
}
