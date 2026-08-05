package cstdlibcontract

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

func TestComponentsForSymbolsClosesDependenciesInDeterministicOrder(t *testing.T) {
	t.Parallel()
	components, err := ComponentsForSymbols([]string{
		"fairyc_external_helper",
		"fy_std_string_append_v0",
		"fy_std_linux_file_read_v0",
		"fy_std_string_append_v0",
		"fy_std_arena_initialize_v0",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []Component{
		ComponentLinuxFile,
		ComponentSystemHeap,
		ComponentBuffer,
		ComponentArena,
		ComponentTextView,
		ComponentOwnedText,
	}
	if !slices.Equal(components, want) {
		t.Fatalf("components = %v, want %v", components, want)
	}
}

func TestComponentsForSymbolsIgnoresNonStandardForeignSymbols(t *testing.T) {
	t.Parallel()
	components, err := ComponentsForSymbols([]string{"fairyc_external_helper"})
	if err != nil || len(components) != 0 {
		t.Fatalf("ComponentsForSymbols() = %v, %v", components, err)
	}
}

func TestComponentsForSymbolsRejectsUnknownStandardLibraryNamespaceMember(t *testing.T) {
	t.Parallel()
	if _, err := ComponentsForSymbols([]string{"fy_std_missing_v0"}); err == nil {
		t.Fatal("ComponentsForSymbols() accepted an unknown fy_std_ symbol")
	}
}

func TestAdmittedKeepsUnexercisedLinuxFileSymbolsClosed(t *testing.T) {
	t.Parallel()
	if !Admitted("fy_std_linux_file_close_v0") {
		t.Fatal("close symbol is not admitted")
	}
	for _, symbol := range []string{"fy_std_linux_file_write_v0", "fy_std_linux_file_drop_v0"} {
		if Admitted(symbol) {
			t.Fatalf("%s unexpectedly admitted", symbol)
		}
	}
}

func TestAdmittedForTargetKeepsWindowsConsoleTargetBound(t *testing.T) {
	t.Parallel()
	const symbol = "fy_std_windows_stdout_write_all_v0"
	if !Admitted(symbol) || !AdmittedForTarget(symbol, target.X8664WindowsMSVC) {
		t.Fatal("Windows stdout symbol is not admitted for its target")
	}
	if AdmittedForTarget(symbol, target.X8664LinuxGNU) {
		t.Fatal("Windows stdout symbol leaked into the Linux target contract")
	}
	if _, err := ComponentsForSymbols([]string{symbol}); err == nil || !strings.Contains(err.Error(), "no x86_64-linux-gnu source contract") {
		t.Fatalf("ComponentsForSymbols() error = %v", err)
	}
}

func TestTargetNeutralConsoleIOSelectsOneTargetOwnedComponent(t *testing.T) {
	t.Parallel()
	symbols := []string{
		"fy_std_stdout_write_all_v0",
		"fy_std_stdout_write_i32_v0",
		"fy_std_stdout_write_u32_v0",
		"fy_std_stdout_write_usize_v0",
		"fy_std_stderr_write_all_v0",
		"fy_std_stdin_read_v0",
	}
	for _, symbol := range symbols {
		if !Admitted(symbol) || !AdmittedForTarget(symbol, target.X8664LinuxGNU) ||
			!AdmittedForTarget(symbol, target.X8664WindowsMSVC) {
			t.Errorf("target-neutral console symbol %q is not admitted for both implemented targets", symbol)
		}
	}
	linux, err := ComponentsForSymbols(symbols)
	if err != nil || !slices.Equal(linux, []Component{ComponentLinuxConsole}) {
		t.Fatalf("Linux components = %v, %v", linux, err)
	}
	windows, err := WindowsMSVCComponentsForSymbols(symbols)
	if err != nil || !slices.Equal(windows, []Component{ComponentWindowsConsole}) {
		t.Fatalf("Windows components = %v, %v", windows, err)
	}
}

func TestScalarOutputExcludesBoolRawSymbol(t *testing.T) {
	t.Parallel()
	if Admitted("fy_std_stdout_write_bool_v0") {
		t.Fatal("bool raw output symbol is admitted; write_bool must reuse stdout_write_all")
	}
}

func TestFromBufferStorageAdmitsOnlyThePublicStorageSymbol(t *testing.T) {
	t.Parallel()
	const symbol = "fy_std_string_from_buffer_storage_v0"
	if !Admitted(symbol) || !AdmittedForTarget(symbol, target.X8664LinuxGNU) ||
		!AdmittedForTarget(symbol, target.X8664WindowsMSVC) {
		t.Fatal("from-buffer storage symbol is not admitted for both implemented targets")
	}
	if Admitted("fy_std_string_from_buffer_v0") {
		t.Fatal("probe-only fy_std_string_from_buffer_v0 leaked into the public symbol contract")
	}
	components, err := ComponentsForSymbols([]string{symbol})
	if err != nil || !slices.Equal(components, []Component{ComponentSystemHeap, ComponentBuffer, ComponentTextView, ComponentOwnedText}) {
		t.Fatalf("ComponentsForSymbols(from-buffer storage) = %v, %v", components, err)
	}
}

func TestAdmittedForTargetKeepsWindowsFileTargetBound(t *testing.T) {
	t.Parallel()
	for _, symbol := range []string{
		"fy_std_windows_file_open_v0",
		"fy_std_windows_file_remove_v0",
		"fy_std_windows_directory_create_v0",
		"fy_std_windows_directory_remove_v0",
		"fy_std_windows_file_rename_v0",
		"fy_std_windows_file_read_v0",
		"fy_std_windows_file_read_exact_v0",
		"fy_std_windows_file_read_append_v0",
		"fy_std_windows_file_position_v0",
		"fy_std_windows_file_size_v0",
		"fy_std_windows_file_kind_v0",
		"fy_std_windows_file_id_v0",
		"fy_std_windows_file_set_len_v0",
		"fy_std_windows_file_seek_start_v0",
		"fy_std_windows_file_write_v0",
		"fy_std_windows_file_write_all_v0",
		"fy_std_windows_file_sync_v0",
		"fy_std_windows_file_close_v0",
		"fy_std_windows_file_drop_v0",
	} {
		if !Admitted(symbol) || !AdmittedForTarget(symbol, target.X8664WindowsMSVC) {
			t.Errorf("%s is not admitted for Windows", symbol)
		}
		if AdmittedForTarget(symbol, target.X8664LinuxGNU) {
			t.Errorf("%s leaked into the Linux target contract", symbol)
		}
	}
}

func TestAdmittedForTargetKeepsProcessArgsTargetBound(t *testing.T) {
	t.Parallel()
	for _, symbol := range []string{"fy_std_process_arg_count_v0", "fy_std_process_arg_v0"} {
		if !Admitted(symbol) || !AdmittedForTarget(symbol, target.X8664WindowsMSVC) {
			t.Fatalf("%s is not admitted for Windows", symbol)
		}
		if AdmittedForTarget(symbol, target.X8664LinuxGNU) {
			t.Fatalf("%s leaked into the Linux target contract", symbol)
		}
		if _, err := ComponentsForSymbols([]string{symbol}); err == nil || !strings.Contains(err.Error(), "no x86_64-linux-gnu source contract") {
			t.Fatalf("ComponentsForSymbols(%q) error = %v", symbol, err)
		}
	}
	components, err := WindowsMSVCComponentsForSymbols([]string{"fy_std_process_arg_count_v0", "fy_std_process_arg_v0"})
	if err != nil || !slices.Equal(components, []Component{ComponentWindowsProcessArgs}) {
		t.Fatalf("WindowsMSVCComponentsForSymbols(process args) = %v, %v", components, err)
	}
}

func TestWindowsMSVCComponentsForSymbolsAdmitsOnlyProvenRuntimeSlice(t *testing.T) {
	t.Parallel()
	components, err := WindowsMSVCComponentsForSymbols([]string{
		"fy_std_stdout_write_all_v0",
		"fy_std_stdout_write_i32_v0",
		"fy_std_stdout_write_u32_v0",
		"fy_std_stdout_write_usize_v0",
		"fy_std_stderr_write_all_v0",
		"fy_std_stdin_read_v0",
		"fy_std_windows_stdout_write_all_v0",
		"fy_std_windows_file_open_v0",
		"fy_std_windows_file_remove_v0",
		"fy_std_windows_directory_create_v0",
		"fy_std_windows_directory_remove_v0",
		"fy_std_windows_file_rename_v0",
		"fy_std_windows_file_read_v0",
		"fy_std_windows_file_read_append_v0",
		"fy_std_windows_file_set_len_v0",
		"fy_std_windows_file_write_v0",
		"fy_std_windows_file_write_all_v0",
		"fy_std_windows_file_sync_v0",
		"fy_std_windows_file_close_v0",
		"fy_std_windows_file_drop_v0",
		"fy_std_process_arg_count_v0",
		"fy_std_process_arg_v0",
		"fy_std_text_utf8_valid_v0",
		"fy_std_buffer_reserve_v0",
		"fy_std_buffer_append_v0",
		"fy_std_buffer_resize_zeroed_v0",
		"fy_std_buffer_initialize_v0",
		"fy_std_buffer_truncate_v0",
		"fy_std_buffer_clear_v0",
		"fy_std_buffer_drop_v0",
		"fy_std_region_allocate_zeroed_v0",
		"fy_std_region_copy_bytes_v0",
		"fy_std_arena_initialize_v0",
		"fy_std_region_begin_v0",
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
		"fy_std_string_from_buffer_storage_v0",
		"fy_std_cstring_from_text_storage_v0",
		"fy_std_cstring_drop_storage_v0",
		"fy_std_system_heap_deallocate_v0",
		"fy_std_system_heap_allocate_v0",
		"fy_std_text_utf8_valid_v0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []Component{ComponentWindowsConsole, ComponentWindowsFile, ComponentWindowsProcessArgs, ComponentSystemHeap, ComponentBuffer, ComponentWindowsFileBuffer, ComponentArena, ComponentTextView, ComponentOwnedText}; !slices.Equal(components, want) {
		t.Fatalf("components = %v, want %v", components, want)
	}
	for _, test := range []struct {
		symbol   string
		contains string
	}{
		{symbol: "fairyc_external_helper", contains: "no closed x86_64-windows-msvc companion contract"},
		{symbol: "fy_std_linux_file_open_v0", contains: "no x86_64-windows-msvc source contract"},
		{symbol: "fy_std_missing_v0", contains: "unrecognized Stage 0 standard-library symbol"},
	} {
		if _, err := WindowsMSVCComponentsForSymbols([]string{test.symbol}); err == nil || !strings.Contains(err.Error(), test.contains) {
			t.Errorf("WindowsMSVCComponentsForSymbols(%q) error = %v; want containing %q", test.symbol, err, test.contains)
		}
	}
}
