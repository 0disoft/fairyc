package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
)

func TestLinuxGNUCompanionRuntimeAssets(t *testing.T) {
	root := t.TempDir()
	pack := filepath.Join(root, "linux.pack")
	python := "python3"
	if runtime.GOOS == "windows" {
		python = "python"
	}
	command := exec.Command(python, "-B", "../../tools/package-candidate-linux-runtime.py", "--output", pack)
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatalf("package Linux assets: %s %v", data, err)
	}
	cases := [][]string{{}, {"fy_std_stdout_write_all_v0"}, {"fy_std_file_open_v0"},
		{"fy_std_system_heap_allocate_v0"}, {"fy_std_bytes_match_at_v0"}, {"fy_std_buffer_drop_v0"},
		{"fy_std_file_read_append_v0"}, {"fy_std_arena_drop_v0"}, {"fy_std_text_utf8_valid_v0"},
		{"fy_std_process_arg_v0"}, {"fy_std_string_copy_v0"}, {"fy_private_vector_reserve_v0"}}
	hosts := []string{"fairyc_private_linux_root_file_open_v1", "fairyc_private_linux_process_start_v1",
		"fairyc_private_linux_capture_open_v1", "fairyc_private_linux_test_capture_open_v1",
		"fairyc_private_linux_host_path_v1", "fairyc_private_linux_directory_scan_open_v1",
		"fairyc_private_linux_root_file_erase_if_v1", "fairyc_private_linux_workspace_path_kind_v1",
		"fairyc_private_linux_build_scratch_open_v1", "fairyc_private_linux_starter_create_v1", "fairyc_private_linux_gnu_run_v1",
		"fairyc_private_linux_package_directory_create_v1", "fairyc_private_linux_package_directory_edit_v1",
		"fairyc_private_linux_package_directory_publish_v1", "fairyc_private_linux_package_directory_erase_v1",
		"fairyc_private_linux_package_directory_clone_v1", "fairyc_private_linux_package_file_info_v1",
		"fairyc_private_linux_monotonic_ms_v1", "fairyc_private_linux_https_open_v1",
		"fairyc_private_linux_edit_file_prepare_v1", "fairyc_private_linux_edit_file_publish_v1",
		"fairyc_private_linux_https_headers_v1", "fairyc_private_linux_https_read_v1", "fairyc_private_linux_https_close_v1"}
	for _, name := range hosts {
		cases = append(cases, []string{name})
	}
	all := append([]string{}, hosts...)
	for _, names := range cases[:12] {
		all = append(all, names...)
	}
	cases = append(cases, all,
		[]string{"fairyc_private_linux_package_directory_publish_v1", "fairyc_private_linux_package_file_info_v1", "fairyc_private_linux_root_file_erase_if_v1"},
		[]string{"fairyc_private_linux_https_open_v1", "fairyc_private_linux_process_start_v1", "fy_std_stdout_write_all_v0"},
		[]string{"fairyc_private_linux_https_read_v1", "fairyc_private_linux_monotonic_ms_v1", "fairyc_private_linux_https_open_v1"})
	var fixture bytes.Buffer
	put := func(n uint32) {
		t.Helper()
		if err := binary.Write(&fixture, binary.LittleEndian, n); err != nil {
			t.Fatal(err)
		}
	}
	part := func(s string) { put(uint32(len(s))); fixture.WriteString(s) }
	put(uint32(len(cases) * 2))
	for _, names := range cases {
		for _, trap := range []bool{false, true} {
			header := "/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\n"
			for _, name := range names {
				header += "extern void " + name + "(void);\n"
			}
			source, err := nativesupport.Source(target.X8664LinuxGNU, names, nativesupport.Options{SDKStorage: true, CandidateHost: true, TrapReceipt: trap})
			if err != nil {
				t.Fatal(err)
			}
			if trap {
				put(1)
			} else {
				put(0)
			}
			part(header)
			part(source)
		}
	}
	path := filepath.Join(root, "cases.bin")
	if err := os.WriteFile(path, fixture.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("comparing %d complete Linux runtime C outputs and asset tampering", len(cases)*2)
	entry := fmt.Sprintf("@effects(filesystem) fn i32 main(){compare(%q,%q);return 0;}", filepath.ToSlash(pack), filepath.ToSlash(path))
	// The source owner includes no Windows-native build module in the Linux assembly closure.
	if strings.Contains(entry, "\r") {
		t.Fatal("noncanonical fixture path")
	}
	runLinuxCandidateFixture(t, []string{"linux_runtime_assets", "linux_runtime_asset_pin", "linux_runtime", "linux_runtime_contract", "sha256"}, "linux_runtime_assets", entry, "")
}
