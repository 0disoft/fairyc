// Package cstdlibcontract owns the closed Stage 0 C standard-library symbol and dependency contract.
package cstdlibcontract

import (
	"fmt"
	"strings"
)

// Component identifies one source-owned standard-library implementation slice.
type Component string

const (
	ComponentLinuxFile  Component = "linux-file"
	ComponentSystemHeap Component = "system-heap"
	ComponentBuffer     Component = "buffer"
	ComponentArena      Component = "arena"
	ComponentTextView   Component = "text-view"
	ComponentOwnedText  Component = "owned-text"
)

var symbolComponents = map[string]Component{
	"fy_std_system_heap_allocate_v0":   ComponentSystemHeap,
	"fy_std_system_heap_deallocate_v0": ComponentSystemHeap,
	"fy_std_buffer_initialize_v0":      ComponentBuffer,
	"fy_std_buffer_resize_zeroed_v0":   ComponentBuffer,
	"fy_std_buffer_drop_v0":            ComponentBuffer,
	"fy_std_arena_initialize_v0":       ComponentArena,
	"fy_std_region_begin_v0":           ComponentArena,
	"fy_std_region_allocate_zeroed_v0": ComponentArena,
	"fy_std_region_reset_v0":           ComponentArena,
	"fy_std_arena_drop_v0":             ComponentArena,
	"fy_std_text_utf8_valid_v0":        ComponentTextView,
	"fy_std_text_validate_utf8_v0":     ComponentTextView,
	"fy_std_cstr_validate_v0":          ComponentOwnedText,
	"fy_std_string_copy_v0":            ComponentOwnedText,
	"fy_std_string_as_view_v0":         ComponentOwnedText,
	"fy_std_string_append_v0":          ComponentOwnedText,
	"fy_std_string_drop_v0":            ComponentOwnedText,
	"fy_std_cstring_from_text_v0":      ComponentOwnedText,
	"fy_std_cstring_as_view_v0":        ComponentOwnedText,
	"fy_std_cstring_drop_v0":           ComponentOwnedText,
	"fy_std_linux_file_open_v0":        ComponentLinuxFile,
	"fy_std_linux_file_read_v0":        ComponentLinuxFile,
	"fy_std_linux_file_write_all_v0":   ComponentLinuxFile,
	"fy_std_linux_file_sync_v0":        ComponentLinuxFile,
	"fy_std_linux_file_close_v0":       ComponentLinuxFile,
}

var windowsMSVCSymbolComponents = map[string]Component{
	"fy_std_system_heap_allocate_v0":   ComponentSystemHeap,
	"fy_std_system_heap_deallocate_v0": ComponentSystemHeap,
	"fy_std_text_utf8_valid_v0":        ComponentTextView,
}

var componentLinkOrder = []Component{
	ComponentLinuxFile,
	ComponentSystemHeap,
	ComponentBuffer,
	ComponentArena,
	ComponentTextView,
	ComponentOwnedText,
}

// Admitted reports whether symbol is one exact project-owned Stage 0 C entry point.
func Admitted(symbol string) bool {
	_, found := symbolComponents[symbol]
	return found
}

// ComponentsForSymbols returns the minimal dependency-closed component set in deterministic C
// source order. Non-standard-library foreign symbols remain the platform linker's responsibility.
func ComponentsForSymbols(symbols []string) ([]Component, error) {
	required := make(map[Component]struct{})
	for _, symbol := range symbols {
		component, found := symbolComponents[symbol]
		if !found {
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
	case ComponentArena:
		required[ComponentSystemHeap] = struct{}{}
	case ComponentOwnedText:
		required[ComponentSystemHeap] = struct{}{}
		required[ComponentBuffer] = struct{}{}
		required[ComponentTextView] = struct{}{}
	}
}
