// Package cstdlibcontract owns the closed Stage 0 C standard-library symbol and dependency contract.
package cstdlibcontract

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/target"
)

// Component identifies one source-owned standard-library implementation slice.
type Component string

const (
	ComponentLinuxConsole       Component = "linux-console"
	ComponentLinuxFile          Component = "linux-file"
	ComponentWindowsConsole     Component = "windows-console"
	ComponentWindowsFile        Component = "windows-file"
	ComponentWindowsFileBuffer  Component = "windows-file-buffer"
	ComponentWindowsProcessArgs Component = "windows-process-args"
	ComponentSystemHeap         Component = "system-heap"
	ComponentBuffer             Component = "buffer"
	ComponentArena              Component = "arena"
	ComponentTextView           Component = "text-view"
	ComponentOwnedText          Component = "owned-text"
)

var symbolComponents = map[string]Component{
	"fy_std_stdout_write_all_v0":           ComponentLinuxConsole,
	"fy_std_stdout_write_i32_v0":           ComponentLinuxConsole,
	"fy_std_stdout_write_u32_v0":           ComponentLinuxConsole,
	"fy_std_stdout_write_usize_v0":         ComponentLinuxConsole,
	"fy_std_stderr_write_all_v0":           ComponentLinuxConsole,
	"fy_std_stderr_write_i32_v0":           ComponentLinuxConsole,
	"fy_std_stderr_write_u32_v0":           ComponentLinuxConsole,
	"fy_std_stderr_write_usize_v0":         ComponentLinuxConsole,
	"fy_std_stdin_read_v0":                 ComponentLinuxConsole,
	"fy_std_system_heap_allocate_v0":       ComponentSystemHeap,
	"fy_std_system_heap_deallocate_v0":     ComponentSystemHeap,
	"fy_std_buffer_initialize_v0":          ComponentBuffer,
	"fy_std_buffer_reserve_v0":             ComponentBuffer,
	"fy_std_buffer_append_v0":              ComponentBuffer,
	"fy_std_buffer_resize_zeroed_v0":       ComponentBuffer,
	"fy_std_buffer_truncate_v0":            ComponentBuffer,
	"fy_std_buffer_clear_v0":               ComponentBuffer,
	"fy_std_buffer_drop_v0":                ComponentBuffer,
	"fy_std_arena_initialize_v0":           ComponentArena,
	"fy_std_region_begin_v0":               ComponentArena,
	"fy_std_region_allocate_zeroed_v0":     ComponentArena,
	"fy_std_region_copy_bytes_v0":          ComponentArena,
	"fy_std_region_reset_v0":               ComponentArena,
	"fy_std_arena_drop_v0":                 ComponentArena,
	"fy_std_text_utf8_valid_v0":            ComponentTextView,
	"fy_std_text_validate_utf8_v0":         ComponentTextView,
	"fy_std_cstr_validate_v0":              ComponentOwnedText,
	"fy_std_string_copy_v0":                ComponentOwnedText,
	"fy_std_string_as_view_v0":             ComponentOwnedText,
	"fy_std_string_append_v0":              ComponentOwnedText,
	"fy_std_string_drop_v0":                ComponentOwnedText,
	"fy_std_cstring_from_text_v0":          ComponentOwnedText,
	"fy_std_cstring_as_view_v0":            ComponentOwnedText,
	"fy_std_cstring_drop_v0":               ComponentOwnedText,
	"fy_std_string_copy_storage_v0":        ComponentOwnedText,
	"fy_std_string_append_storage_v0":      ComponentOwnedText,
	"fy_std_string_drop_storage_v0":        ComponentOwnedText,
	"fy_std_string_from_buffer_storage_v0": ComponentOwnedText,
	"fy_std_cstring_from_text_storage_v0":  ComponentOwnedText,
	"fy_std_cstring_drop_storage_v0":       ComponentOwnedText,
	"fy_std_linux_file_open_v0":            ComponentLinuxFile,
	"fy_std_linux_file_read_v0":            ComponentLinuxFile,
	"fy_std_linux_file_write_all_v0":       ComponentLinuxFile,
	"fy_std_linux_file_sync_v0":            ComponentLinuxFile,
	"fy_std_linux_file_close_v0":           ComponentLinuxFile,
}

var windowsOnlySymbolComponents = map[string]Component{
	"fy_std_windows_stdout_write_all_v0":  ComponentWindowsConsole,
	"fy_std_windows_file_open_v0":         ComponentWindowsFile,
	"fy_std_windows_file_remove_v0":       ComponentWindowsFile,
	"fy_std_windows_file_copy_bytes_v0":   ComponentWindowsFile,
	"fy_std_windows_directory_create_v0":  ComponentWindowsFile,
	"fy_std_windows_directory_ensure_v0":  ComponentWindowsFile,
	"fy_std_windows_directory_remove_v0":  ComponentWindowsFile,
	"fy_std_windows_dir_remove_all_v0":    ComponentWindowsFile,
	"fy_std_windows_directory_rename_v0":  ComponentWindowsFile,
	"fy_std_windows_file_rename_v0":       ComponentWindowsFile,
	"fy_std_windows_file_hard_link_v0":    ComponentWindowsFile,
	"fy_std_windows_file_read_v0":         ComponentWindowsFile,
	"fy_std_windows_file_read_exact_v0":   ComponentWindowsFile,
	"fy_std_windows_file_read_append_v0":  ComponentWindowsFileBuffer,
	"fy_std_windows_file_position_v0":     ComponentWindowsFile,
	"fy_std_windows_file_size_v0":         ComponentWindowsFile,
	"fy_std_windows_file_kind_v0":         ComponentWindowsFile,
	"fy_std_windows_file_read_only_v0":    ComponentWindowsFile,
	"fy_std_windows_file_set_readonly_v0": ComponentWindowsFile,
	"fy_std_windows_file_link_count_v0":   ComponentWindowsFile,
	"fy_std_windows_file_alloc_size_v0":   ComponentWindowsFile,
	"fy_std_windows_file_path_kind_v0":    ComponentWindowsFile,
	"fy_std_windows_file_id_v0":           ComponentWindowsFile,
	"fy_std_windows_file_set_len_v0":      ComponentWindowsFile,
	"fy_std_windows_file_seek_current_v0": ComponentWindowsFile,
	"fy_std_windows_file_seek_end_v0":     ComponentWindowsFile,
	"fy_std_windows_file_seek_start_v0":   ComponentWindowsFile,
	"fy_std_windows_file_write_v0":        ComponentWindowsFile,
	"fy_std_windows_file_write_all_v0":    ComponentWindowsFile,
	"fy_std_windows_file_sync_v0":         ComponentWindowsFile,
	"fy_std_windows_file_close_v0":        ComponentWindowsFile,
	"fy_std_windows_file_drop_v0":         ComponentWindowsFile,
	"fy_std_process_arg_count_v0":         ComponentWindowsProcessArgs,
	"fy_std_process_arg_v0":               ComponentWindowsProcessArgs,
}

var windowsMSVCSymbolComponents = map[string]Component{
	"fy_std_stdout_write_all_v0":           ComponentWindowsConsole,
	"fy_std_stdout_write_i32_v0":           ComponentWindowsConsole,
	"fy_std_stdout_write_u32_v0":           ComponentWindowsConsole,
	"fy_std_stdout_write_usize_v0":         ComponentWindowsConsole,
	"fy_std_stderr_write_all_v0":           ComponentWindowsConsole,
	"fy_std_stderr_write_i32_v0":           ComponentWindowsConsole,
	"fy_std_stderr_write_u32_v0":           ComponentWindowsConsole,
	"fy_std_stderr_write_usize_v0":         ComponentWindowsConsole,
	"fy_std_stdin_read_v0":                 ComponentWindowsConsole,
	"fy_std_windows_stdout_write_all_v0":   ComponentWindowsConsole,
	"fy_std_windows_file_open_v0":          ComponentWindowsFile,
	"fy_std_windows_file_remove_v0":        ComponentWindowsFile,
	"fy_std_windows_file_copy_bytes_v0":    ComponentWindowsFile,
	"fy_std_windows_directory_create_v0":   ComponentWindowsFile,
	"fy_std_windows_directory_ensure_v0":   ComponentWindowsFile,
	"fy_std_windows_directory_remove_v0":   ComponentWindowsFile,
	"fy_std_windows_dir_remove_all_v0":     ComponentWindowsFile,
	"fy_std_windows_directory_rename_v0":   ComponentWindowsFile,
	"fy_std_windows_file_rename_v0":        ComponentWindowsFile,
	"fy_std_windows_file_hard_link_v0":     ComponentWindowsFile,
	"fy_std_windows_file_read_v0":          ComponentWindowsFile,
	"fy_std_windows_file_read_exact_v0":    ComponentWindowsFile,
	"fy_std_windows_file_read_append_v0":   ComponentWindowsFileBuffer,
	"fy_std_windows_file_position_v0":      ComponentWindowsFile,
	"fy_std_windows_file_size_v0":          ComponentWindowsFile,
	"fy_std_windows_file_kind_v0":          ComponentWindowsFile,
	"fy_std_windows_file_read_only_v0":     ComponentWindowsFile,
	"fy_std_windows_file_set_readonly_v0":  ComponentWindowsFile,
	"fy_std_windows_file_link_count_v0":    ComponentWindowsFile,
	"fy_std_windows_file_alloc_size_v0":    ComponentWindowsFile,
	"fy_std_windows_file_path_kind_v0":     ComponentWindowsFile,
	"fy_std_windows_file_id_v0":            ComponentWindowsFile,
	"fy_std_windows_file_set_len_v0":       ComponentWindowsFile,
	"fy_std_windows_file_seek_current_v0":  ComponentWindowsFile,
	"fy_std_windows_file_seek_end_v0":      ComponentWindowsFile,
	"fy_std_windows_file_seek_start_v0":    ComponentWindowsFile,
	"fy_std_windows_file_write_v0":         ComponentWindowsFile,
	"fy_std_windows_file_write_all_v0":     ComponentWindowsFile,
	"fy_std_windows_file_sync_v0":          ComponentWindowsFile,
	"fy_std_windows_file_close_v0":         ComponentWindowsFile,
	"fy_std_windows_file_drop_v0":          ComponentWindowsFile,
	"fy_std_process_arg_count_v0":          ComponentWindowsProcessArgs,
	"fy_std_process_arg_v0":                ComponentWindowsProcessArgs,
	"fy_std_system_heap_allocate_v0":       ComponentSystemHeap,
	"fy_std_system_heap_deallocate_v0":     ComponentSystemHeap,
	"fy_std_buffer_initialize_v0":          ComponentBuffer,
	"fy_std_buffer_reserve_v0":             ComponentBuffer,
	"fy_std_buffer_append_v0":              ComponentBuffer,
	"fy_std_buffer_resize_zeroed_v0":       ComponentBuffer,
	"fy_std_buffer_truncate_v0":            ComponentBuffer,
	"fy_std_buffer_clear_v0":               ComponentBuffer,
	"fy_std_buffer_drop_v0":                ComponentBuffer,
	"fy_std_arena_initialize_v0":           ComponentArena,
	"fy_std_region_begin_v0":               ComponentArena,
	"fy_std_region_allocate_zeroed_v0":     ComponentArena,
	"fy_std_region_copy_bytes_v0":          ComponentArena,
	"fy_std_region_reset_v0":               ComponentArena,
	"fy_std_arena_drop_v0":                 ComponentArena,
	"fy_std_text_utf8_valid_v0":            ComponentTextView,
	"fy_std_cstr_validate_v0":              ComponentOwnedText,
	"fy_std_string_copy_v0":                ComponentOwnedText,
	"fy_std_string_as_view_v0":             ComponentOwnedText,
	"fy_std_string_append_v0":              ComponentOwnedText,
	"fy_std_string_drop_v0":                ComponentOwnedText,
	"fy_std_cstring_from_text_v0":          ComponentOwnedText,
	"fy_std_cstring_as_view_v0":            ComponentOwnedText,
	"fy_std_cstring_drop_v0":               ComponentOwnedText,
	"fy_std_string_copy_storage_v0":        ComponentOwnedText,
	"fy_std_string_append_storage_v0":      ComponentOwnedText,
	"fy_std_string_drop_storage_v0":        ComponentOwnedText,
	"fy_std_string_from_buffer_storage_v0": ComponentOwnedText,
	"fy_std_cstring_from_text_storage_v0":  ComponentOwnedText,
	"fy_std_cstring_drop_storage_v0":       ComponentOwnedText,
}

var componentLinkOrder = []Component{
	ComponentLinuxConsole,
	ComponentLinuxFile,
	ComponentWindowsConsole,
	ComponentWindowsFile,
	ComponentWindowsProcessArgs,
	ComponentSystemHeap,
	ComponentBuffer,
	ComponentWindowsFileBuffer,
	ComponentArena,
	ComponentTextView,
	ComponentOwnedText,
}

// Admitted reports whether symbol is one exact project-owned Stage 0 C entry point.
func Admitted(symbol string) bool {
	if _, found := symbolComponents[symbol]; found {
		return true
	}
	_, found := windowsOnlySymbolComponents[symbol]
	return found
}

// AdmittedForTarget reports whether one exact project-owned symbol belongs to
// the closed C standard-library contract for targetID.
func AdmittedForTarget(symbol string, targetID target.ID) bool {
	switch targetID {
	case target.X8664LinuxGNU:
		_, found := symbolComponents[symbol]
		return found
	case target.X8664WindowsMSVC:
		if _, found := windowsMSVCSymbolComponents[symbol]; found {
			return true
		}
		_, knownForAnotherTarget := symbolComponents[symbol]
		return knownForAnotherTarget
	default:
		return false
	}
}

// ComponentsForSymbols returns the minimal dependency-closed component set in deterministic C
// source order. Non-standard-library foreign symbols remain the platform linker's responsibility.
func ComponentsForSymbols(symbols []string) ([]Component, error) {
	required := make(map[Component]struct{})
	for _, symbol := range symbols {
		component, found := symbolComponents[symbol]
		if !found {
			if _, targetSpecific := windowsOnlySymbolComponents[symbol]; targetSpecific {
				return nil, fmt.Errorf("Stage 0 standard-library symbol %q has no x86_64-linux-gnu source contract", symbol)
			}
			if strings.HasPrefix(symbol, "fy_std_") {
				return nil, fmt.Errorf("unrecognized Stage 0 standard-library symbol %q", symbol)
			}
			continue
		}
		addComponentClosure(required, component)
	}
	return componentsInLinkOrder(required), nil
}

// WindowsMSVCComponentsForSymbols returns the closed standard-library subset
// with current source-level x86_64-windows-msvc execution evidence. Unlike the
// Linux product path, arbitrary foreign symbols are rejected before linking.
func WindowsMSVCComponentsForSymbols(symbols []string) ([]Component, error) {
	required := make(map[Component]struct{})
	for _, symbol := range symbols {
		component, found := windowsMSVCSymbolComponents[symbol]
		if !found {
			switch {
			case strings.HasPrefix(symbol, "fy_std_") && Admitted(symbol):
				return nil, fmt.Errorf("Stage 0 standard-library symbol %q has no x86_64-windows-msvc source contract", symbol)
			case strings.HasPrefix(symbol, "fy_std_"):
				return nil, fmt.Errorf("unrecognized Stage 0 standard-library symbol %q", symbol)
			default:
				return nil, fmt.Errorf("foreign symbol %q has no closed x86_64-windows-msvc companion contract", symbol)
			}
		}
		addComponentClosure(required, component)
	}
	return componentsInLinkOrder(required), nil
}

func componentsInLinkOrder(required map[Component]struct{}) []Component {
	components := make([]Component, 0, len(required))
	for _, component := range componentLinkOrder {
		if _, found := required[component]; found {
			components = append(components, component)
		}
	}
	return components
}

func addComponentClosure(required map[Component]struct{}, component Component) {
	required[component] = struct{}{}
	switch component {
	case ComponentBuffer:
		required[ComponentSystemHeap] = struct{}{}
	case ComponentWindowsFileBuffer:
		required[ComponentWindowsFile] = struct{}{}
		required[ComponentSystemHeap] = struct{}{}
		required[ComponentBuffer] = struct{}{}
	case ComponentArena:
		required[ComponentSystemHeap] = struct{}{}
	case ComponentOwnedText:
		required[ComponentSystemHeap] = struct{}{}
		required[ComponentBuffer] = struct{}{}
		required[ComponentTextView] = struct{}{}
	}
}
