// Package candidatehost embeds private native ABI adapters for Fairy host code.
// No frontend, command policy, process sequence or deadline policy lives here.
package candidatehost

import (
	_ "embed"
	"fmt"
	"regexp"
)

//go:embed windows_process.c
var windowsProcess string

var symbols = regexp.MustCompile(`\bfairyc_private_windows_[A-Za-z0-9_]+\s*\(`)
var admitted = map[string]bool{
	"fairyc_private_windows_create_env_v1": true, "fairyc_private_windows_pipe_v1": true,
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
}

func WindowsSource(header string) (string, error) {
	matches := symbols.FindAllString(header, -1)
	if len(matches) == 0 {
		return "", nil
	}
	for _, match := range matches {
		name := match[:len(match)-1]
		for len(name) > 0 && (name[len(name)-1] == ' ' || name[len(name)-1] == '\t' || name[len(name)-1] == '\n' || name[len(name)-1] == '\r') {
			name = name[:len(name)-1]
		}
		if !admitted[name] {
			return "", fmt.Errorf("unknown private Windows host symbol %s", name)
		}
	}
	return windowsProcess, nil
}
