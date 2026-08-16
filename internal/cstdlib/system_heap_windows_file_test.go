package cstdlib

import (
	"os"
	"strings"
	"testing"
)

func TestWindowsFileSourcesConsumeCloseAndExposePartialIO(t *testing.T) {
	t.Parallel()
	implementation, err := WindowsFileSource()
	if err != nil {
		t.Fatal(err)
	}
	probe, err := WindowsFileProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"fy_std_file_open_v0", "fy_std_file_read_v0", "fy_std_file_write_v0",
		"fy_std_file_position_v0", "fy_std_file_size_v0", "fy_std_file_seek_start_v0",
		"fy_std_file_seek_current_v0", "fy_std_file_seek_end_v0",
		"fy_std_file_write_all_v0", "fy_std_file_sync_v0", "fy_std_file_close_v0",
		"fy_std_file_drop_v0",
		"CreateFileW", "ReadFile", "WriteFile", "GetFileSizeEx", "GetFileInformationByHandleEx", "FileBasicInfo", "FileStandardInfo", "FileIdInfo", "SetFileInformationByHandle", "FileEndOfFileInfo", "SetFilePointerEx", "FILE_BEGIN", "FILE_CURRENT", "FILE_END", "FlushFileBuffers", "CloseHandle",
		"fy_std_windows_file_seek_current_v0", "fy_std_windows_file_seek_end_v0",
		"fy_std_windows_file_read_only_v0", "fy_std_windows_file_set_readonly_v0", "FILE_ATTRIBUTE_READONLY",
		"fy_std_windows_file_link_count_v0", "NumberOfLinks",
		"fy_std_windows_file_alloc_size_v0", "AllocationSize",
		"fy_std_windows_file_hard_link_v0", "CreateHardLinkW",
		"fy_std_windows_dir_remove_all_v0", "FindFirstFileW", "FindNextFileW", "FindClose", "DeleteFileW",
		"fy_std_windows_file_path_kind_v0", "GetFileAttributesW", "FILE_ATTRIBUTE_REPARSE_POINT", "FY_STD_WINDOWS_PATH_KIND_OTHER_V0",
		"FY_STD_WINDOWS_FILE_PATH_ENCODING_V0", "FY_STD_WINDOWS_FILE_PATH_TOO_LONG_V0",
		"struct fy_std_windows_file_result_v0 fy_std_windows_file_write_v0(",
		"memcpy(file, storage, sizeof *file)", "memcpy(storage, file, sizeof *file)",
		"file.handle = NULL", "file.open = UINT8_C(0)",
		"fy_std_windows_file_store_v0(file_storage, &file);\n    if (!CloseHandle(handle))",
	} {
		if !strings.Contains(implementation, required) {
			t.Errorf("Windows File implementation is missing %q", required)
		}
	}
	for _, required := range []string{
		"result.os_code != (uint32_t)ERROR_INVALID_HANDLE",
		"fy_std_windows_file_seek_start_v0(&file, UINT64_MAX)",
		"fy_std_windows_file_seek_start_v0(&file, UINT64_MAX).status !=",
		"fy_std_windows_file_seek_current_v0(&file, INT32_C(-4))",
		"fy_std_windows_file_seek_end_v0(&file, INT32_C(-9))",
		"fy_std_windows_file_position_v0(&file)",
		"fy_std_windows_file_size_v0(&file)",
		"fy_std_windows_file_kind_v0(&file)",
		"fy_std_windows_file_id_v0(&file, &original_id)",
		"fy_std_windows_file_set_len_v0(&file, UINT64_MAX)",
		"size.os_code != (uint32_t)ERROR_ACCESS_DENIED",
		"fy_std_windows_file_read_v0(&file, received + total, requested)",
		"fy_std_windows_file_read_exact_v0(&file, exact, sizeof exact)",
		"fy_std_windows_file_drop_v0(&file)",
	} {
		if !strings.Contains(probe, required) {
			t.Errorf("Windows File probe is missing %q", required)
		}
	}
}
func TestWindowsFileCopyProbeHasDedicatedSourceOwner(t *testing.T) {
	t.Parallel()

	mainProbe, err := os.ReadFile("stage0_windows_file_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	copyProbe, err := os.ReadFile("stage0_windows_file_copy_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	const maximumProbeBytes = 40 * 1024
	if len(mainProbe) > maximumProbeBytes {
		t.Fatalf("main Windows File probe is %d bytes, want at most %d", len(mainProbe), maximumProbeBytes)
	}
	mainText := string(mainProbe)
	copyText := string(copyProbe)
	if strings.Contains(mainText, "fy_std_windows_file_copy_bytes_v0") ||
		strings.Contains(mainText, "copy_payload") {
		t.Fatal("main Windows File probe retained byte-copy proof ownership")
	}
	if strings.Count(mainText, "int fy_std_windows_file_copy_probe_v0(void);") != 1 ||
		strings.Count(mainText, "return fy_std_windows_file_copy_probe_v0();") != 1 {
		t.Fatal("main Windows File probe must declare and invoke the copy proof exactly once")
	}
	if strings.Count(copyText, "int fy_std_windows_file_copy_probe_v0(void)") != 1 ||
		!strings.Contains(copyText, "fy_std_windows_file_copy_bytes_v0") ||
		!strings.Contains(copyText, "copy_payload") {
		t.Fatal("dedicated Windows File copy probe lost its definition or proof body")
	}
	combined, err := WindowsFileProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(combined, "int fy_std_windows_file_copy_probe_v0(void)") != 2 {
		t.Fatal("combined Windows File probe must contain one declaration and one definition")
	}
}
func TestWindowsFileDirectoryProbeHasDedicatedSourceOwner(t *testing.T) {
	t.Parallel()

	mainProbe, err := os.ReadFile("stage0_windows_file_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	directoryProbe, err := os.ReadFile("stage0_windows_file_directory_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	mainText := string(mainProbe)
	directoryText := string(directoryProbe)
	for _, residual := range []string{
		"directory_path[]", "directory_rename_source[]", "directory_all_leaf[]",
	} {
		if strings.Contains(mainText, residual) {
			t.Errorf("main Windows File probe retained directory proof marker %q", residual)
		}
	}
	if strings.Count(mainText, "int fy_std_windows_file_directory_probe_v0(void);") != 1 ||
		strings.Count(mainText, "fy_std_windows_file_directory_probe_v0();") != 1 {
		t.Fatal("main Windows File probe must declare and invoke the directory proof exactly once")
	}
	if strings.Count(directoryText, "int fy_std_windows_file_directory_probe_v0(void)") != 1 ||
		!strings.Contains(directoryText, "fy_std_windows_directory_create_v0") ||
		!strings.Contains(directoryText, "fy_std_windows_directory_rename_v0") ||
		!strings.Contains(directoryText, "fy_std_windows_directory_ensure_v0") ||
		!strings.Contains(directoryText, "directory_all_partial") {
		t.Fatal("dedicated Windows File directory probe lost its definition or proof body")
	}
	combined, err := WindowsFileProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(combined, "int fy_std_windows_file_directory_probe_v0(void)") != 2 {
		t.Fatal("combined Windows File probe must contain one directory declaration and one definition")
	}
}
func TestWindowsDirectoryRemoveTreeHasDedicatedSourceOwners(t *testing.T) {
	t.Parallel()

	mainSource, err := os.ReadFile("stage0_windows_file.c")
	if err != nil {
		t.Fatal(err)
	}
	removeSource, err := os.ReadFile("stage0_windows_file_remove_tree.c")
	if err != nil {
		t.Fatal(err)
	}
	mainProbe, err := os.ReadFile("stage0_windows_file_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	removeProbe, err := os.ReadFile("stage0_windows_file_remove_tree_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	mainText := string(mainSource)
	removeText := string(removeSource)
	probeText := string(mainProbe)
	removeProbeText := string(removeProbe)
	if strings.Contains(mainText, "fy_std_windows_dir_remove_all_v0") ||
		strings.Contains(mainText, "FindFirstFileW") {
		t.Fatal("main Windows File source retained recursive-removal ownership")
	}
	for _, required := range []string{
		"fy_std_windows_dir_remove_all_v0", "FindFirstFileW", "FindNextFileW", "FindClose",
		"FILE_ATTRIBUTE_REPARSE_POINT", "ERROR_NO_MORE_FILES", "ERROR_ACCESS_DENIED",
		"fy_std_windows_directory_root_length_v0", "fy_std_windows_directory_components_valid_v0",
	} {
		if !strings.Contains(removeText, required) {
			t.Errorf("recursive-removal source is missing %q", required)
		}
	}
	if strings.Count(probeText, "int fy_std_windows_file_remove_tree_probe_v0(void);") != 1 ||
		strings.Count(probeText, "fy_std_windows_file_remove_tree_probe_v0();") != 1 {
		t.Fatal("main Windows File probe must declare and invoke recursive removal exactly once")
	}
	for _, required := range []string{
		"int fy_std_windows_file_remove_tree_probe_v0(void)",
		"ERROR_SHARING_VIOLATION", "result.count != 1", "FILE_ATTRIBUTE_READONLY",
		"FSCTL_SET_REPARSE_POINT", "IO_REPARSE_TAG_MOUNT_POINT",
		"fy-remove-tree-target\\\\marker.tmp", "junction_attributes & FILE_ATTRIBUTE_REPARSE_POINT",
	} {
		if !strings.Contains(removeProbeText, required) {
			t.Errorf("recursive-removal probe is missing %q", required)
		}
	}
	combined, err := WindowsFileSource()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(combined, "fy_std_windows_dir_remove_all_v0(") != 1 ||
		!strings.Contains(combined, "static uint32_t fy_std_windows_directory_root_length_v0(") {
		t.Fatal("combined Windows File source lost recursive removal or shared validation")
	}
	combinedProbe, err := WindowsFileProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(combinedProbe, "int fy_std_windows_file_remove_tree_probe_v0(void)") != 2 {
		t.Fatal("combined Windows File probe must contain one declaration and one definition")
	}
}
func TestWindowsFileLinkProbeHasDedicatedSourceOwner(t *testing.T) {
	t.Parallel()

	mainProbe, err := os.ReadFile("stage0_windows_file_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	linkProbe, err := os.ReadFile("stage0_windows_file_link_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	mainText := string(mainProbe)
	linkText := string(linkProbe)
	if strings.Contains(mainText, "fairyc-windows-file-link-source-probe.tmp") {
		t.Fatal("main Windows File probe retained hard-link proof ownership")
	}
	if strings.Count(mainText, "int fy_std_windows_file_link_probe_v0(void);") != 1 ||
		strings.Count(mainText, "fy_std_windows_file_link_probe_v0();") != 1 {
		t.Fatal("main Windows File probe must declare and invoke the hard-link proof exactly once")
	}
	if strings.Count(linkText, "int fy_std_windows_file_link_probe_v0(void)") != 1 ||
		!strings.Contains(linkText, "fy_std_windows_file_hard_link_v0") ||
		!strings.Contains(linkText, "ERROR_ALREADY_EXISTS") ||
		!strings.Contains(linkText, "fy_std_windows_file_ids_equal_v0") ||
		!strings.Contains(linkText, "memcmp(received, payload, sizeof payload)") {
		t.Fatal("dedicated Windows File hard-link probe lost its definition or proof body")
	}
	combined, err := WindowsFileProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(combined, "int fy_std_windows_file_link_probe_v0(void)") != 2 {
		t.Fatal("combined Windows File probe must contain one hard-link declaration and one definition")
	}
}
func TestWindowsFilePathKindProbeHasDedicatedSourceOwner(t *testing.T) {
	t.Parallel()

	mainProbe, err := os.ReadFile("stage0_windows_file_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	pathKindProbe, err := os.ReadFile("stage0_windows_file_path_kind_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	mainText := string(mainProbe)
	pathKindText := string(pathKindProbe)
	if strings.Contains(mainText, "fy_std_windows_file_path_kind_v0") ||
		strings.Contains(mainText, "path_kind_directory") {
		t.Fatal("main Windows File probe retained path-kind proof ownership")
	}
	if strings.Count(mainText, "int fy_std_windows_file_path_kind_probe_v0(void);") != 1 ||
		strings.Count(mainText, "fy_std_windows_file_path_kind_probe_v0();") != 1 {
		t.Fatal("main Windows File probe must declare and invoke the path-kind proof exactly once")
	}
	if strings.Count(pathKindText, "int fy_std_windows_file_path_kind_probe_v0(void)") != 1 ||
		!strings.Contains(pathKindText, "fy_std_windows_file_path_kind_v0") ||
		!strings.Contains(pathKindText, "path_kind_directory") {
		t.Fatal("dedicated Windows File path-kind probe lost its definition or proof body")
	}
	combined, err := WindowsFileProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(combined, "int fy_std_windows_file_path_kind_probe_v0(void)") != 2 {
		t.Fatal("combined Windows File probe must contain one path-kind declaration and one definition")
	}
}
func TestWindowsFileSeekProbeHasDedicatedSourceOwner(t *testing.T) {
	t.Parallel()

	mainProbe, err := os.ReadFile("stage0_windows_file_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	seekProbe, err := os.ReadFile("stage0_windows_file_seek_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	mainText := string(mainProbe)
	seekText := string(seekProbe)
	if strings.Contains(mainText, "fy_std_windows_file_seek_current_v0") ||
		strings.Contains(mainText, "fy_std_windows_file_seek_end_v0") {
		t.Fatal("main Windows File probe retained relative seek proof ownership")
	}
	if strings.Count(mainText, "int fy_std_windows_file_seek_probe_v0(void);") != 1 ||
		strings.Count(mainText, "fy_std_windows_file_seek_probe_v0();") != 1 {
		t.Fatal("main Windows File probe must declare and invoke the seek proof exactly once")
	}
	if strings.Count(seekText, "int fy_std_windows_file_seek_probe_v0(void)") != 1 ||
		!strings.Contains(seekText, "fy_std_windows_file_seek_current_v0") ||
		!strings.Contains(seekText, "fy_std_windows_file_seek_end_v0") {
		t.Fatal("dedicated Windows File seek probe lost its definition or proof body")
	}
	combined, err := WindowsFileProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(combined, "int fy_std_windows_file_seek_probe_v0(void)") != 2 {
		t.Fatal("combined Windows File probe must contain one seek declaration and one definition")
	}
}
func TestWindowsFileMetadataProbeHasDedicatedSourceOwner(t *testing.T) {
	t.Parallel()

	mainProbe, err := os.ReadFile("stage0_windows_file_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	metadataProbe, err := os.ReadFile("stage0_windows_file_metadata_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	mainText := string(mainProbe)
	metadataText := string(metadataProbe)
	if strings.Contains(mainText, "fy_std_windows_file_read_only_v0") ||
		strings.Contains(mainText, "fy_std_windows_file_set_readonly_v0") ||
		strings.Contains(mainText, "fy_std_windows_file_link_count_v0") ||
		strings.Contains(mainText, "fy_std_windows_file_alloc_size_v0") ||
		strings.Contains(mainText, "metadata_read_only") {
		t.Fatal("main Windows File probe retained metadata proof ownership")
	}
	if strings.Count(mainText, "int fy_std_windows_file_metadata_probe_v0(void);") != 1 ||
		strings.Count(mainText, "fy_std_windows_file_metadata_probe_v0();") != 1 {
		t.Fatal("main Windows File probe must declare and invoke the metadata proof exactly once")
	}
	if strings.Count(metadataText, "int fy_std_windows_file_metadata_probe_v0(void)") != 1 ||
		!strings.Contains(metadataText, "fy_std_windows_file_read_only_v0") ||
		!strings.Contains(metadataText, "fy_std_windows_file_set_readonly_v0") ||
		!strings.Contains(metadataText, "ERROR_ACCESS_DENIED") ||
		!strings.Contains(metadataText, "fy_std_windows_file_link_count_v0") ||
		!strings.Contains(metadataText, "fy_std_windows_file_alloc_size_v0") ||
		!strings.Contains(metadataText, "allocated.size < sizeof payload") ||
		!strings.Contains(metadataText, "CreateHardLinkW") {
		t.Fatal("dedicated Windows File metadata probe lost its definition or proof body")
	}
	combined, err := WindowsFileProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(combined, "int fy_std_windows_file_metadata_probe_v0(void)") != 2 {
		t.Fatal("combined Windows File probe must contain one metadata declaration and one definition")
	}
}
func TestWindowsFileMetadataImplementationHasDedicatedSourceOwner(t *testing.T) {
	t.Parallel()

	mainSource, err := os.ReadFile("stage0_windows_file.c")
	if err != nil {
		t.Fatal(err)
	}
	metadataSource, err := os.ReadFile("stage0_windows_file_metadata.c")
	if err != nil {
		t.Fatal(err)
	}
	mainText := string(mainSource)
	metadataText := string(metadataSource)
	if strings.Contains(mainText, "fy_std_windows_file_attribute_result_v0") ||
		strings.Contains(mainText, "fy_std_windows_file_read_only_v0") ||
		strings.Contains(mainText, "fy_std_windows_file_set_readonly_v0") ||
		strings.Contains(mainText, "fy_std_windows_file_link_count_v0") ||
		strings.Contains(mainText, "fy_std_windows_file_alloc_size_v0") {
		t.Fatal("main Windows File source retained metadata implementation ownership")
	}
	if strings.Count(metadataText, "struct fy_std_windows_file_attribute_result_v0") != 5 ||
		strings.Count(metadataText, "fy_std_windows_file_read_only_v0(") != 1 ||
		strings.Count(metadataText, "fy_std_windows_file_set_readonly_v0(") != 1 ||
		!strings.Contains(metadataText, "SetFileInformationByHandle") ||
		!strings.Contains(metadataText, "FILE_ATTRIBUTE_NORMAL") ||
		!strings.Contains(metadataText, "FileBasicInfo") ||
		!strings.Contains(metadataText, "FILE_ATTRIBUTE_READONLY") ||
		strings.Count(metadataText, "fy_std_windows_file_link_count_v0(") != 1 ||
		!strings.Contains(metadataText, "FileStandardInfo") ||
		!strings.Contains(metadataText, "NumberOfLinks") ||
		strings.Count(metadataText, "fy_std_windows_file_alloc_size_v0(") != 1 ||
		!strings.Contains(metadataText, "AllocationSize") {
		t.Fatal("dedicated Windows File metadata source lost its result or query implementation")
	}
	combined, err := WindowsFileSource()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(combined, "fy_std_windows_file_read_only_v0(") != 1 ||
		strings.Count(combined, "fy_std_windows_file_set_readonly_v0(") != 1 ||
		strings.Count(combined, "fy_std_windows_file_link_count_v0(") != 1 ||
		strings.Count(combined, "fy_std_windows_file_alloc_size_v0(") != 1 ||
		!strings.Contains(combined, "static int fy_std_windows_file_load_v0(") {
		t.Fatal("combined Windows File source lost exact metadata ownership or shared handle loading")
	}
}
func TestWindowsFileHardLinkImplementationHasDedicatedSourceOwner(t *testing.T) {
	t.Parallel()

	mainSource, err := os.ReadFile("stage0_windows_file.c")
	if err != nil {
		t.Fatal(err)
	}
	linkSource, err := os.ReadFile("stage0_windows_file_link.c")
	if err != nil {
		t.Fatal(err)
	}
	mainText := string(mainSource)
	linkText := string(linkSource)
	if strings.Contains(mainText, "fy_std_windows_file_hard_link_v0") ||
		strings.Contains(mainText, "CreateHardLinkW") {
		t.Fatal("main Windows File source retained hard-link implementation ownership")
	}
	if strings.Count(linkText, "fy_std_windows_file_hard_link_v0(") != 1 ||
		!strings.Contains(linkText, "CreateHardLinkW(wide_destination, wide_source, NULL)") ||
		strings.Count(linkText, "fy_std_windows_path_v0(") != 2 {
		t.Fatal("dedicated Windows File hard-link source lost validation or no-replace creation")
	}
	combined, err := WindowsFileSource()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(combined, "fy_std_windows_file_hard_link_v0(") != 1 ||
		!strings.Contains(combined, "static uint32_t fy_std_windows_path_v0(") {
		t.Fatal("combined Windows File source lost exact hard-link ownership or shared path validation")
	}
}
func TestWindowsMSVCSourceForSymbolsSelectsOnlyWindowsFile(t *testing.T) {
	t.Parallel()
	source, err := WindowsMSVCSourceForSymbols([]string{"fy_std_windows_file_open_v0"})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"fy_std_windows_file_open_v0", "fy_std_windows_file_read_v0", "fy_std_windows_file_read_exact_v0",
		"fy_std_windows_file_position_v0", "fy_std_windows_file_seek_current_v0", "fy_std_windows_file_seek_end_v0", "fy_std_windows_file_seek_start_v0",
		"fy_std_windows_file_size_v0", "fy_std_windows_file_kind_v0", "fy_std_windows_file_read_only_v0", "fy_std_windows_file_set_readonly_v0", "fy_std_windows_file_link_count_v0", "fy_std_windows_file_alloc_size_v0", "fy_std_windows_file_hard_link_v0", "fy_std_windows_dir_remove_all_v0", "fy_std_windows_file_path_kind_v0", "fy_std_windows_file_set_len_v0",
		"fy_std_windows_file_id_v0",
		"fy_std_windows_file_write_v0",
		"fy_std_windows_file_write_all_v0", "fy_std_windows_file_sync_v0",
		"fy_std_windows_file_close_v0", "fy_std_windows_file_drop_v0",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Windows File closure is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"fy_std_windows_stdout_write_all_v0", "GetStdHandle",
		"fy_std_system_heap_allocate_v0", "fy_std_buffer_initialize_v0",
		"fy_std_linux_file_open_v0", "_POSIX_C_SOURCE",
	} {
		if strings.Contains(source, forbidden) {
			t.Errorf("Windows File closure unexpectedly contains %q", forbidden)
		}
	}
}
func TestWindowsMSVCSourceForSymbolsClosesOwnedBufferReadDependencies(t *testing.T) {
	t.Parallel()
	source, err := WindowsMSVCSourceForSymbols([]string{"fy_std_windows_file_read_append_v0"})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"fy_std_windows_file_read_v0", "fy_std_system_heap_allocate_v0",
		"fy_std_buffer_reserve_v0", "fy_std_windows_file_read_append_v0",
		"buffer->length = old_length + file_result.count",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("owned-buffer read closure is missing %q", required)
		}
	}
	if strings.Index(source, "fy_std_windows_file_read_v0") > strings.Index(source, "fy_std_windows_file_read_append_v0") ||
		strings.Index(source, "fy_std_buffer_reserve_v0") > strings.Index(source, "fy_std_windows_file_read_append_v0") {
		t.Fatal("owned-buffer adapter precedes one of its C dependencies")
	}
	for _, forbidden := range []string{
		"fy_std_windows_stdout_write_all_v0", "fy_std_process_arg_count_v0",
		"fy_std_arena_initialize_v0", "fy_std_string_copy_storage_v0",
	} {
		if strings.Contains(source, forbidden) {
			t.Errorf("owned-buffer read closure unexpectedly contains %q", forbidden)
		}
	}
}
