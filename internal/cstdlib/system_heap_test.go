package cstdlib

import (
	"strings"
	"testing"
)

func TestSystemHeapSourcesAreClosedAndCanonical(t *testing.T) {
	t.Parallel()
	implementation, err := SystemHeapSource()
	if err != nil {
		t.Fatal(err)
	}
	probe, err := SystemHeapProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"fy_std_system_heap_allocate_v0",
		"fy_std_system_heap_resize_v0",
		"fy_std_system_heap_deallocate_v0",
		"#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 _Alignof(max_align_t)",
		"#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) malloc(size)",
		"#define FY_STD_SYSTEM_HEAP_FREE_V0(data) free(data)",
		"alignment <= FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0",
		"void *data = FY_STD_SYSTEM_HEAP_MALLOC_V0(size)",
		"FY_STD_SYSTEM_HEAP_FREE_V0(block->data)",
		"memcpy(replacement.data, block->data, copy_size)",
	} {
		if !strings.Contains(implementation, required) {
			t.Errorf("SystemHeap implementation is missing %q", required)
		}
	}
	if strings.Contains(implementation, "realloc(") || strings.Contains(implementation, "printf") {
		t.Fatal("SystemHeap implementation contains forbidden raw realloc or output")
	}
	if !strings.Contains(probe, "int main(void)") || !strings.Contains(probe, "FY_STD_ALLOC_UNSUPPORTED_ALIGNMENT_V0") {
		t.Fatal("SystemHeap probe does not exercise the closed contract")
	}
	for name, source := range map[string]string{"implementation": implementation, "probe": probe} {
		repeated := source
		if name == "implementation" {
			repeated, err = SystemHeapSource()
		} else {
			repeated, err = SystemHeapProbeSource()
		}
		if err != nil || repeated != source || strings.Contains(source, "\r") || !strings.HasSuffix(source, "\n") {
			t.Fatalf("%s source is not deterministic canonical C: %v", name, err)
		}
	}
}

func TestBufferSourcesOwnFailureAtomicOverlapSafeStorage(t *testing.T) {
	t.Parallel()
	implementation, err := StorageSource()
	if err != nil {
		t.Fatal(err)
	}
	probe, err := BufferProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	probeImplementation, err := BufferProbeImplementationSource()
	if err != nil {
		t.Fatal(err)
	}
	windowsProbeImplementation, err := WindowsMSVCBufferProbeImplementationSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"fy_std_buffer_reserve_v0",
		"fy_std_buffer_append_v0",
		"count > SIZE_MAX - buffer->length",
		"fy_std_buffer_internal_source_v0",
		"memmove(",
		"fy_std_buffer_resize_zeroed_v0",
	} {
		if !strings.Contains(implementation, required) {
			t.Errorf("Buffer implementation is missing %q", required)
		}
	}
	for _, required := range []string{
		"abcdefbcde",
		"SIZE_MAX",
		"fy_std_system_heap_probe_fail_after_allocations_v0(0)",
		"failed_initialize.block.data != (void *)(uintptr_t)1",
		"fy_std_system_heap_probe_live_allocations_v0() != 1",
		"buffer.block.data != preserved_data",
		"buffer.length != preserved_length",
	} {
		if !strings.Contains(probe, required) {
			t.Errorf("Buffer probe is missing %q", required)
		}
	}
	repeated, err := StorageSource()
	if err != nil || repeated != implementation || strings.Contains(implementation, "\r") || !strings.HasSuffix(implementation, "\n") {
		t.Fatalf("StorageSource() is not deterministic canonical C: %v", err)
	}
	for _, candidate := range []struct {
		name   string
		source string
	}{
		{name: "portable", source: probeImplementation},
		{name: "Windows MSVC", source: windowsProbeImplementation},
	} {
		for _, required := range []string{
			"fy_std_system_heap_probe_fail_after_allocations_v0",
			"fy_std_system_heap_probe_live_allocations_v0",
			"#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) fy_std_system_heap_probe_malloc_v0(size)",
			"fy_std_buffer_initialize_v0",
			"fy_std_buffer_resize_zeroed_v0",
			"fy_std_buffer_drop_v0",
		} {
			if !strings.Contains(candidate.source, required) {
				t.Errorf("%s Buffer probe implementation is missing %q", candidate.name, required)
			}
		}
		if strings.Index(candidate.source, "#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) fy_std_system_heap_probe_malloc_v0(size)") >
			strings.Index(candidate.source, "#ifndef FY_STD_SYSTEM_HEAP_MALLOC_V0") {
			t.Errorf("%s Buffer probe allocator adapter follows the SystemHeap fallback", candidate.name)
		}
	}
	if !strings.Contains(windowsProbeImplementation, "#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)") {
		t.Error("Windows MSVC Buffer probe implementation is missing its target alignment contract")
	}
	if strings.Contains(implementation, "fy_std_system_heap_probe_fail_after_allocations_v0") {
		t.Error("product Buffer implementation leaked the probe-only allocator controls")
	}
	repeatedProbeImplementation, err := BufferProbeImplementationSource()
	if err != nil || repeatedProbeImplementation != probeImplementation {
		t.Fatalf("BufferProbeImplementationSource() is not deterministic canonical C: %v", err)
	}
	repeatedWindowsProbeImplementation, err := WindowsMSVCBufferProbeImplementationSource()
	if err != nil || repeatedWindowsProbeImplementation != windowsProbeImplementation {
		t.Fatalf("WindowsMSVCBufferProbeImplementationSource() is not deterministic canonical C: %v", err)
	}
}

func TestArenaSourcesRejectConcurrentAndStaleRegions(t *testing.T) {
	t.Parallel()
	implementation, err := ArenaSource()
	if err != nil {
		t.Fatal(err)
	}
	probe, err := ArenaProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	probeImplementation, err := ArenaProbeImplementationSource()
	if err != nil {
		t.Fatal(err)
	}
	windowsProbeImplementation, err := WindowsMSVCArenaProbeImplementationSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"fy_std_region_begin_v0",
		"fy_std_region_allocate_zeroed_v0",
		"fy_std_region_copy_bytes_v0",
		"fy_std_region_reset_v0",
		"region->generation == region->arena->generation",
		"arena->active_region != 0",
	} {
		if !strings.Contains(implementation, required) {
			t.Errorf("Arena implementation is missing %q", required)
		}
	}
	for _, required := range []string{
		"fy_std_region_begin_v0(&arena, &second)",
		"fy_std_region_allocate_zeroed_v0(&region, 1, 1, &empty)",
		"fy_std_system_heap_probe_fail_after_allocations_v0(0)",
		"fy_std_system_heap_probe_fail_after_allocations_v0(1)",
		"fy_std_system_heap_probe_live_allocations_v0() != 4",
		"failed_output != (void *)(uintptr_t)1",
	} {
		if !strings.Contains(probe, required) {
			t.Errorf("Arena probe is missing %q", required)
		}
	}
	repeated, err := ArenaSource()
	if err != nil || repeated != implementation || strings.Contains(implementation, "\r") || !strings.HasSuffix(implementation, "\n") {
		t.Fatalf("ArenaSource() is not deterministic canonical C: %v", err)
	}
	for _, candidate := range []struct {
		name   string
		source string
	}{
		{name: "portable", source: probeImplementation},
		{name: "Windows MSVC", source: windowsProbeImplementation},
	} {
		for _, required := range []string{
			"fy_std_system_heap_probe_fail_after_allocations_v0",
			"fy_std_system_heap_probe_live_allocations_v0",
			"#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) fy_std_system_heap_probe_malloc_v0(size)",
			"fy_std_region_allocate_zeroed_v0",
			"fy_std_region_reset_v0",
		} {
			if !strings.Contains(candidate.source, required) {
				t.Errorf("%s Arena probe implementation is missing %q", candidate.name, required)
			}
		}
		if strings.Index(candidate.source, "#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) fy_std_system_heap_probe_malloc_v0(size)") >
			strings.Index(candidate.source, "#ifndef FY_STD_SYSTEM_HEAP_MALLOC_V0") {
			t.Errorf("%s Arena probe allocator adapter follows the SystemHeap fallback", candidate.name)
		}
	}
	if !strings.Contains(windowsProbeImplementation, "#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)") {
		t.Error("Windows MSVC Arena probe implementation is missing its target alignment contract")
	}
	if strings.Contains(implementation, "fy_std_system_heap_probe_fail_after_allocations_v0") {
		t.Error("product Arena implementation leaked the probe-only allocator controls")
	}
	repeatedProbeImplementation, err := ArenaProbeImplementationSource()
	if err != nil || repeatedProbeImplementation != probeImplementation {
		t.Fatalf("ArenaProbeImplementationSource() is not deterministic canonical C: %v", err)
	}
	repeatedWindowsProbeImplementation, err := WindowsMSVCArenaProbeImplementationSource()
	if err != nil || repeatedWindowsProbeImplementation != windowsProbeImplementation {
		t.Fatalf("WindowsMSVCArenaProbeImplementationSource() is not deterministic canonical C: %v", err)
	}
}

func TestTextViewSourcesRejectNonCanonicalUTF8AndInteriorBoundaries(t *testing.T) {
	t.Parallel()
	implementation, err := TextViewSource()
	if err != nil {
		t.Fatal(err)
	}
	probe, err := TextViewProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"uint32_t fy_std_text_utf8_valid_v0(const uint8_t *data, size_t length)",
		"first >= UINT8_C(0xc2)",
		"first == UINT8_C(0xe0)",
		"first == UINT8_C(0xed)",
		"first == UINT8_C(0xf4)",
		"fy_std_text_validate_utf8_v0",
		"fy_std_text_slice_v0",
	} {
		if !strings.Contains(implementation, required) {
			t.Errorf("text view implementation is missing %q", required)
		}
	}
	if strings.Contains(implementation, "static int fy_std_text_utf8_valid_v0") {
		t.Fatal("text view implementation keeps the source-callable validation transport private")
	}
	for _, required := range []string{"overlong_two", "surrogate", "over_maximum", "truncated", "obsolete_five"} {
		if !strings.Contains(probe, required) {
			t.Errorf("text view probe is missing %q", required)
		}
	}
	repeated, err := TextViewSource()
	if err != nil || repeated != implementation || strings.Contains(implementation, "\r") || !strings.HasSuffix(implementation, "\n") {
		t.Fatalf("TextViewSource() is not deterministic canonical C: %v", err)
	}
}

func TestOwnedTextSourcesPreserveOwnershipAndNULTermination(t *testing.T) {
	t.Parallel()
	implementation, err := OwnedTextSource()
	if err != nil {
		t.Fatal(err)
	}
	probe, err := OwnedTextProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	probeImplementation, err := OwnedTextProbeImplementationSource()
	if err != nil {
		t.Fatal(err)
	}
	windowsProbeImplementation, err := WindowsMSVCOwnedTextProbeImplementationSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"fy_std_string_from_buffer_v0",
		"fy_std_string_into_buffer_v0",
		"input == NULL || !fy_std_buffer_valid_v0(&input->bytes) || output == NULL",
		"FY_STD_OWNED_TEXT_INVALID_UTF8_V0",
		"fy_std_string_append_v0",
		"fy_std_cstr_validate_v0",
		"FY_STD_OWNED_TEXT_INTERIOR_NUL_V0",
		"fy_std_cstring_from_text_v0",
		"fy_std_string_copy_storage_v0",
		"fy_std_string_append_storage_v0",
		"fy_std_string_drop_storage_v0",
		"fy_std_string_from_buffer_storage_v0",
		"fy_std_string_into_buffer_storage_v0",
		"input_storage == output_storage",
		"memcpy(input_storage, &input, sizeof input)",
		"fy_std_cstring_from_text_storage_v0",
		"fy_std_cstring_drop_storage_v0",
		"memcpy(output_storage, &output, sizeof output)",
		"memcpy(value_storage, &value, sizeof value)",
		"if (result.status == FY_STD_OWNED_TEXT_OK_V0)",
	} {
		if !strings.Contains(implementation, required) {
			t.Errorf("owned text implementation is missing %q", required)
		}
	}
	for _, required := range []string{
		"invalid_buffer.block.data != invalid_data",
		"invalid_cstr",
		"cview.data[cview.length] != 0",
		"fy_std_string_copy_storage_v0",
		"fy_std_string_append_storage_v0",
		"fy_std_string_drop_storage_v0",
		"fy_std_string_from_buffer_storage_v0",
		"fy_std_string_into_buffer_storage_v0",
		"fy_std_cstring_from_text_storage_v0",
		"fy_std_cstring_drop_storage_v0",
		"preserved_string_bytes",
		"preserved_nonempty_string_output",
		"_Static_assert(sizeof(struct fy_std_buffer_storage_v0) == sizeof(struct fy_std_buffer_v0)",
		"preserved_nonempty_from_buffer_input",
		"preserved_nonempty_from_buffer_output",
		"from_buffer_output.data != transfer_data",
		"buffer_storage.data != transfer_data",
		"empty_buffer_storage",
		"invalid_buffer_data",
		"storage_self_prefix",
		"malformed_string",
		"overflow_text",
		"preserved_nonempty_cstring_output",
		"malformed_cstring",
		"allocation_failed",
		"fy_std_system_heap_probe_fail_after_allocations_v0",
		"fy_std_system_heap_probe_live_allocations_v0() != 1",
	} {
		if !strings.Contains(probe, required) {
			t.Errorf("owned text probe is missing %q", required)
		}
	}
	repeated, err := OwnedTextSource()
	if err != nil || repeated != implementation || strings.Contains(implementation, "\r") || !strings.HasSuffix(implementation, "\n") {
		t.Fatalf("OwnedTextSource() is not deterministic canonical C: %v", err)
	}
	repeatedProbe, err := OwnedTextProbeSource()
	if err != nil || repeatedProbe != probe || strings.Contains(probe, "\r") || !strings.HasSuffix(probe, "\n") {
		t.Fatalf("OwnedTextProbeSource() is not deterministic canonical C: %v", err)
	}
	for _, candidate := range []struct {
		name   string
		source string
	}{
		{name: "portable", source: probeImplementation},
		{name: "Windows MSVC", source: windowsProbeImplementation},
	} {
		for _, required := range []string{
			"fy_std_system_heap_probe_fail_after_allocations_v0",
			"fy_std_system_heap_probe_live_allocations_v0",
			"#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) fy_std_system_heap_probe_malloc_v0(size)",
			"#define FY_STD_SYSTEM_HEAP_FREE_V0(data) fy_std_system_heap_probe_free_v0(data)",
			"fy_std_string_copy_storage_v0",
			"fy_std_string_from_buffer_storage_v0",
			"fy_std_string_into_buffer_storage_v0",
			"fy_std_cstring_from_text_storage_v0",
		} {
			if !strings.Contains(candidate.source, required) {
				t.Errorf("%s owned-text probe implementation is missing %q", candidate.name, required)
			}
		}
		if strings.Index(candidate.source, "#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) fy_std_system_heap_probe_malloc_v0(size)") >
			strings.Index(candidate.source, "#ifndef FY_STD_SYSTEM_HEAP_MALLOC_V0") {
			t.Errorf("%s owned-text probe allocator adapter follows the SystemHeap fallback", candidate.name)
		}
		if strings.Contains(candidate.source, "\r") || !strings.HasSuffix(candidate.source, "\n") {
			t.Errorf("%s owned-text probe implementation is not canonical C", candidate.name)
		}
	}
	if !strings.Contains(windowsProbeImplementation, "#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)") {
		t.Error("Windows MSVC owned-text probe implementation is missing its target alignment contract")
	}
	if strings.Contains(implementation, "fy_std_system_heap_probe_fail_after_allocations_v0") {
		t.Error("product owned-text implementation leaked the probe-only allocator controls")
	}
	repeatedProbeImplementation, err := OwnedTextProbeImplementationSource()
	if err != nil || repeatedProbeImplementation != probeImplementation {
		t.Fatalf("OwnedTextProbeImplementationSource() is not deterministic canonical C: %v", err)
	}
	repeatedWindowsProbeImplementation, err := WindowsMSVCOwnedTextProbeImplementationSource()
	if err != nil || repeatedWindowsProbeImplementation != windowsProbeImplementation {
		t.Fatalf("WindowsMSVCOwnedTextProbeImplementationSource() is not deterministic canonical C: %v", err)
	}
}

func TestLinuxFileSourcesConsumeCloseAndExposePartialIOPositionSizeAndSeek(t *testing.T) {
	t.Parallel()
	implementation, err := LinuxFileSource()
	if err != nil {
		t.Fatal(err)
	}
	probe, err := LinuxFileProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"fy_std_linux_file_read_v0", "fy_std_linux_file_write_v0",
		"fy_std_linux_file_write_all_v0", "file->descriptor = -1", "(void)close(descriptor)",
		"fy_std_file_open_v0", "fy_std_file_read_v0", "fy_std_file_write_v0",
		"fy_std_file_position_v0", "fy_std_file_size_v0", "fy_std_file_seek_start_v0",
		"fy_std_file_seek_current_v0", "fy_std_file_seek_end_v0", "SEEK_CUR", "SEEK_END",
		"fy_std_file_write_all_v0", "fy_std_file_sync_v0", "fy_std_file_close_v0",
		"fy_std_file_drop_v0", "O_EXCL", "O_APPEND", "(mode_t)0666",
		"#include <sys/stat.h>",
		"memcpy(file, storage, sizeof *file)", "lseek(file.descriptor", "fstat(file.descriptor",
		"INT64_MAX", "EOVERFLOW", "sizeof(off_t) == sizeof(int64_t)",
		"position result layout drift", "size result layout drift",
	} {
		if !strings.Contains(implementation, required) {
			t.Errorf("Linux File implementation is missing %q", required)
		}
	}
	for _, required := range []string{"mkstemp", "result.error_number != EBADF", "fy_std_linux_file_read_v0(&file, received + total, 3)"} {
		if !strings.Contains(probe, required) {
			t.Errorf("Linux File probe is missing %q", required)
		}
	}
}

func TestLinuxConsoleSourcesPreserveWriteProgressAndReadEOF(t *testing.T) {
	t.Parallel()
	implementation, err := LinuxConsoleSource()
	if err != nil {
		t.Fatal(err)
	}
	probe, err := LinuxConsoleProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"fy_std_stdout_write_all_v0", "fy_std_stdout_write_i32_v0", "fy_std_stderr_write_all_v0", "fy_std_stderr_write_i32_v0", "fy_std_stderr_write_u32_v0", "fy_std_stderr_write_usize_v0", "STDOUT_FILENO", "STDERR_FILENO",
		"fy_std_stdin_read_v0", "read(STDIN_FILENO, data, requested)",
		"SSIZE_MAX", "errno == EINTR", "errno == EBADF",
		"result.written += (size_t)completed",
		"UINT32_C(0) - (uint32_t)value", "uint8_t buffer[11]",
	} {
		if !strings.Contains(implementation, required) {
			t.Errorf("Linux console implementation is missing %q", required)
		}
	}
	for _, required := range []string{
		"#include <errno.h>",
		"open(\"/dev/null\", O_WRONLY)", "dup2(sink, descriptor)",
		"close(descriptor)", "STDOUT_FILENO", "STDERR_FILENO",
		"pipe(pipe_descriptors)", "memcmp(buffer, payload, sizeof(payload))",
		"end.read != (size_t)0", "unavailable.status != UINT32_C(2)",
		"fy_std_stdout_write_i32_v0(INT32_MIN)", "fy_std_stderr_write_i32_v0(INT32_MIN)", "fy_std_stderr_write_usize_v0((uintptr_t)UINT64_MAX)",
		"fy_std_probe_stderr_expected_i32_min_v0[] = \"-2147483648\"",
		"fy_std_probe_stderr_expected_u32_max_v0[] = \"4294967295\"",
		"fy_std_probe_stderr_expected_usize_max_v0[] = \"18446744073709551615\"",
		"results[index].written != fy_std_probe_stderr_expected_v0[index].length",
		"fy_std_probe_read_expected_v0(", "completed != (ssize_t)0",
	} {
		if !strings.Contains(probe, required) {
			t.Errorf("Linux console probe is missing %q", required)
		}
	}
	if strings.Contains(probe, "0-42-214748364821474836470142949672950118446744073709551615") {
		t.Error("Linux console probe retains the malformed concatenated stderr oracle")
	}
}

func TestWindowsMSVCSourceForSymbolsSelectsOnlyProcessArgs(t *testing.T) {
	t.Parallel()
	source, err := WindowsMSVCSourceForSymbols([]string{"fy_std_process_arg_count_v0"})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"fy_std_process_arg_count_v0", "fy_std_process_arg_v0",
		"__p___argc", "__p___wargv", "WideCharToMultiByte",
		"CP_UTF8", "WC_ERR_INVALID_CHARS", "ERROR_NO_UNICODE_TRANSLATION",
		"_configure_wide_argv(_crt_argv_unexpanded_arguments)",
		"written != needed",
		"FY_STD_PROCESS_ARGS_OK_V0", "FY_STD_PROCESS_ARGS_INDEX_OUT_OF_RANGE_V0",
		"FY_STD_PROCESS_ARGS_INVALID_ENCODING_V0", "FY_STD_PROCESS_ARGS_BUFFER_TOO_SMALL_V0",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Windows process args source is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"fy_std_windows_stdout_write_all_v0", "GetStdHandle", "WriteFile",
		"fy_std_system_heap_allocate_v0", "shell32", "_POSIX_C_SOURCE",
	} {
		if strings.Contains(source, forbidden) {
			t.Errorf("Windows process args source unexpectedly contains %q", forbidden)
		}
	}
}

func TestSourceForSymbolsSelectsMinimalDependencyClosure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		symbols   []string
		prefix    string
		required  []string
		forbidden []string
	}{
		{
			name:    "ordinary foreign only",
			symbols: []string{"fairyc_external_helper"},
		},
		{
			name:     "Linux console only",
			symbols:  []string{"fy_std_stdout_write_all_v0", "fy_std_stdout_write_i32_v0", "fy_std_stderr_write_all_v0", "fy_std_stderr_write_i32_v0", "fy_std_stderr_write_u32_v0", "fy_std_stderr_write_usize_v0", "fy_std_stdin_read_v0"},
			prefix:   "#define _POSIX_C_SOURCE 200809L\n",
			required: []string{"fy_std_stdout_write_all_v0", "fy_std_stdout_write_i32_v0", "fy_std_stderr_write_all_v0", "fy_std_stderr_write_i32_v0", "fy_std_stderr_write_u32_v0", "fy_std_stderr_write_usize_v0", "fy_std_stdin_read_v0", "write(descriptor", "read(STDIN_FILENO"},
			forbidden: []string{
				"fy_std_linux_file_open_v0", "fy_std_system_heap_allocate_v0",
			},
		},
		{
			name:     "text view only",
			symbols:  []string{"fy_std_text_utf8_valid_v0"},
			required: []string{"fy_std_text_utf8_valid_v0"},
			forbidden: []string{
				"fy_std_system_heap_allocate_v0", "fy_std_string_copy_v0", "fy_std_linux_file_open_v0",
			},
		},
		{
			name:    "owned text closure",
			symbols: []string{"fy_std_string_append_v0"},
			required: []string{
				"fy_std_system_heap_allocate_v0", "fy_std_buffer_initialize_v0",
				"fy_std_text_utf8_valid_v0", "fy_std_string_append_v0",
			},
			forbidden: []string{"fy_std_arena_initialize_v0", "fy_std_linux_file_open_v0"},
		},
		{
			name:    "Linux File and Arena",
			symbols: []string{"fy_std_linux_file_read_v0", "fy_std_region_begin_v0"},
			prefix:  "#define _POSIX_C_SOURCE 200809L\n",
			required: []string{
				"#define _POSIX_C_SOURCE 200809L", "fy_std_linux_file_read_v0",
				"fy_std_system_heap_allocate_v0", "fy_std_region_begin_v0",
			},
			forbidden: []string{"fy_std_buffer_initialize_v0", "fy_std_text_utf8_valid_v0"},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			source, err := SourceForSymbols(test.symbols)
			if err != nil {
				t.Fatal(err)
			}
			if test.prefix != "" && !strings.HasPrefix(source, test.prefix) {
				t.Errorf("source does not start with %q", test.prefix)
			}
			for _, required := range test.required {
				if !strings.Contains(source, required) {
					t.Errorf("source is missing %q", required)
				}
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(source, forbidden) {
					t.Errorf("source unexpectedly contains %q", forbidden)
				}
			}
			if len(test.required) == 0 && source != "" {
				t.Fatalf("source = %q, want empty", source)
			}
		})
	}
}

func TestSourceForSymbolsRejectsUnknownStandardLibrarySymbol(t *testing.T) {
	t.Parallel()
	if _, err := SourceForSymbols([]string{"fy_std_missing_v0"}); err == nil {
		t.Fatal("SourceForSymbols() accepted an unknown fy_std_ symbol")
	}
}

func TestWindowsMSVCSourceForSymbolsKeepsClosedRuntimeSlice(t *testing.T) {
	t.Parallel()
	source, err := WindowsMSVCSourceForSymbols([]string{
		"fy_std_stdout_write_all_v0",
		"fy_std_stdout_write_i32_v0",
		"fy_std_stderr_write_all_v0",
		"fy_std_stdin_read_v0",
		"fy_std_windows_stdout_write_all_v0",
		"fy_std_windows_file_open_v0",
		"fy_std_windows_file_read_v0",
		"fy_std_windows_file_read_append_v0",
		"fy_std_windows_file_write_v0",
		"fy_std_windows_file_write_all_v0",
		"fy_std_windows_file_sync_v0",
		"fy_std_windows_file_close_v0",
		"fy_std_windows_file_drop_v0",
		"fy_std_process_arg_count_v0",
		"fy_std_process_arg_v0",
		"fy_std_text_utf8_valid_v0",
		"fy_std_buffer_initialize_v0",
		"fy_std_buffer_resize_zeroed_v0",
		"fy_std_buffer_drop_v0",
		"fy_std_arena_initialize_v0",
		"fy_std_region_begin_v0",
		"fy_std_region_allocate_zeroed_v0",
		"fy_std_region_copy_bytes_v0",
		"fy_std_region_reset_v0",
		"fy_std_arena_drop_v0",
		"fy_std_string_copy_v0",
		"fy_std_string_as_view_v0",
		"fy_std_string_append_v0",
		"fy_std_string_drop_v0",
		"fy_std_cstr_validate_v0",
		"fy_std_cstring_from_text_v0",
		"fy_std_cstring_as_view_v0",
		"fy_std_cstring_drop_v0",
		"fy_std_string_copy_storage_v0",
		"fy_std_string_append_storage_v0",
		"fy_std_string_drop_storage_v0",
		"fy_std_cstring_from_text_storage_v0",
		"fy_std_cstring_drop_storage_v0",
		"fy_std_system_heap_allocate_v0",
		"fy_std_system_heap_deallocate_v0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(source, "#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)\n") {
		t.Fatal("Windows SystemHeap source is missing its x64 malloc alignment contract")
	}
	for _, required := range []string{
		"fy_std_stdout_write_all_v0", "fy_std_stdout_write_i32_v0", "fy_std_stderr_write_all_v0", "fy_std_stdin_read_v0", "fy_std_windows_stdout_write_all_v0",
		"STD_INPUT_HANDLE", "STD_OUTPUT_HANDLE", "STD_ERROR_HANDLE", "ReadFile", "WriteFile", "GetStdHandle",
		"UINT32_MAX", "ERROR_BROKEN_PIPE", "ERROR_HANDLE_EOF", "ERROR_INVALID_HANDLE",
		"fy_std_windows_file_open_v0", "fy_std_windows_file_read_v0",
		"fy_std_windows_file_read_append_v0",
		"fy_std_windows_file_write_v0",
		"fy_std_windows_file_write_all_v0", "fy_std_windows_file_sync_v0",
		"fy_std_windows_file_close_v0", "fy_std_windows_file_drop_v0", "CreateFileW",
		"fy_std_process_arg_count_v0", "fy_std_process_arg_v0", "WideCharToMultiByte",
		"fy_std_system_heap_allocate_v0", "fy_std_system_heap_deallocate_v0",
		"fy_std_buffer_initialize_v0", "fy_std_buffer_resize_zeroed_v0", "fy_std_buffer_drop_v0",
		"fy_std_arena_initialize_v0", "fy_std_region_begin_v0",
		"fy_std_region_allocate_zeroed_v0", "fy_std_region_copy_bytes_v0",
		"fy_std_region_reset_v0", "fy_std_arena_drop_v0",
		"fy_std_text_utf8_valid_v0", "fy_std_text_validate_utf8_v0",
		"fy_std_cstr_validate_v0", "fy_std_string_copy_v0", "fy_std_string_as_view_v0",
		"fy_std_string_append_v0", "fy_std_string_drop_v0", "fy_std_cstring_from_text_v0",
		"fy_std_cstring_as_view_v0", "fy_std_cstring_drop_v0",
		"fy_std_string_copy_storage_v0", "fy_std_string_append_storage_v0",
		"fy_std_string_drop_storage_v0", "fy_std_cstring_from_text_storage_v0",
		"fy_std_cstring_drop_storage_v0",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Windows source is missing %q", required)
		}
	}
	for _, forbidden := range []string{"fy_std_linux_file_open_v0", "_POSIX_C_SOURCE"} {
		if strings.Contains(source, forbidden) {
			t.Errorf("Windows source unexpectedly contains %q", forbidden)
		}
	}
	for _, symbol := range []string{"fairyc_external_helper", "fy_std_linux_file_open_v0"} {
		if _, err := WindowsMSVCSourceForSymbols([]string{symbol}); err == nil {
			t.Errorf("WindowsMSVCSourceForSymbols(%q) unexpectedly succeeded", symbol)
		}
	}
}

func TestLinuxSourceRejectsWindowsOnlySymbol(t *testing.T) {
	t.Parallel()
	for _, symbol := range []string{"fy_std_windows_stdout_write_all_v0", "fy_std_windows_file_open_v0"} {
		if _, err := SourceForSymbols([]string{symbol}); err == nil || !strings.Contains(err.Error(), "no x86_64-linux-gnu source contract") {
			t.Fatalf("SourceForSymbols(%q) error = %v", symbol, err)
		}
	}
}

func TestSourceForSymbolsEmitsEachDependencyOnce(t *testing.T) {
	t.Parallel()
	source, err := SourceForSymbols([]string{
		"fy_std_string_append_v0",
		"fy_std_buffer_drop_v0",
		"fy_std_string_append_v0",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range []string{
		"struct fy_std_system_heap_block_v0 {",
		"struct fy_std_buffer_v0 {",
		"struct fy_std_text_view_v0 {",
		"struct fy_std_string_v0 {",
	} {
		if count := strings.Count(source, definition); count != 1 {
			t.Errorf("%q definition count = %d, want 1", definition, count)
		}
	}
}
