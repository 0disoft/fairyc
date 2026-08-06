// Package cstdlib owns embedded C sources for the target standard-library implementation.
package cstdlib

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/cstdlibcontract"
)

//go:embed stage0_system_heap.c
var stage0SystemHeap string

//go:embed stage0_system_heap_probe.c
var stage0SystemHeapProbe string

//go:embed stage0_buffer.c
var stage0Buffer string

//go:embed stage0_buffer_probe.c
var stage0BufferProbe string

//go:embed stage0_arena.c
var stage0Arena string

//go:embed stage0_arena_probe.c
var stage0ArenaProbe string

//go:embed stage0_text_view.c
var stage0TextView string

//go:embed stage0_text_view_probe.c
var stage0TextViewProbe string

//go:embed stage0_owned_text.c
var stage0OwnedText string

//go:embed stage0_owned_text_probe.c
var stage0OwnedTextProbe string

//go:embed stage0_system_heap_probe_allocator.c
var stage0SystemHeapProbeAllocator string

//go:embed stage0_linux_file.c
var stage0LinuxFile string

//go:embed stage0_linux_file_probe.c
var stage0LinuxFileProbe string

//go:embed stage0_linux_console.c
var stage0LinuxConsole string

//go:embed stage0_linux_console_probe.c
var stage0LinuxConsoleProbe string

//go:embed stage0_windows_console.c
var stage0WindowsConsole string

//go:embed stage0_windows_file.c
var stage0WindowsFile string

//go:embed stage0_windows_file_buffer.c
var stage0WindowsFileBuffer string

//go:embed stage0_windows_file_probe.c
var stage0WindowsFileProbe string

//go:embed stage0_windows_file_copy_probe.c
var stage0WindowsFileCopyProbe string

//go:embed stage0_windows_process_args.c
var stage0WindowsProcessArgs string

// SystemHeapSource returns the provisional x86_64-linux-gnu SystemHeap implementation.
func SystemHeapSource() (string, error) {
	return canonicalSource("SystemHeap", stage0SystemHeap)
}

// SystemHeapProbeSource returns the closed native conformance probe for SystemHeap.
func SystemHeapProbeSource() (string, error) {
	return canonicalSource("SystemHeap probe", stage0SystemHeapProbe)
}

// StorageSource returns the provisional SystemHeap and Buffer implementation as one C unit.
func StorageSource() (string, error) {
	heap, err := SystemHeapSource()
	if err != nil {
		return "", err
	}
	buffer, err := canonicalSource("Buffer", stage0Buffer)
	if err != nil {
		return "", err
	}
	return heap + "\n" + buffer, nil
}

// BufferProbeSource returns the closed native conformance probe for Buffer.
func BufferProbeSource() (string, error) {
	return canonicalSource("Buffer probe", stage0BufferProbe)
}

// BufferProbeImplementationSource returns the portable Buffer implementation with the probe-only
// deterministic SystemHeap allocator adapter. The adapter is never part of product linkage.
func BufferProbeImplementationSource() (string, error) {
	implementation, err := StorageSource()
	if err != nil {
		return "", err
	}
	return systemHeapProbeImplementationSource("Buffer", implementation)
}

// WindowsMSVCBufferProbeImplementationSource returns the target-specific Windows Buffer closure
// with the same probe-only deterministic SystemHeap allocator adapter.
func WindowsMSVCBufferProbeImplementationSource() (string, error) {
	implementation, err := WindowsMSVCSourceForSymbols([]string{"fy_std_buffer_initialize_v0"})
	if err != nil {
		return "", err
	}
	return systemHeapProbeImplementationSource("Windows MSVC Buffer", implementation)
}

// ArenaSource returns the provisional SystemHeap and Arena implementation as one C unit.
func ArenaSource() (string, error) {
	heap, err := SystemHeapSource()
	if err != nil {
		return "", err
	}
	arena, err := canonicalSource("Arena", stage0Arena)
	if err != nil {
		return "", err
	}
	return heap + "\n" + arena, nil
}

// ArenaProbeSource returns the closed native conformance probe for Arena and Region.
func ArenaProbeSource() (string, error) {
	return canonicalSource("Arena probe", stage0ArenaProbe)
}

// ArenaProbeImplementationSource returns the portable Arena implementation with the probe-only
// deterministic SystemHeap allocator adapter. The adapter is never part of product linkage.
func ArenaProbeImplementationSource() (string, error) {
	implementation, err := ArenaSource()
	if err != nil {
		return "", err
	}
	return systemHeapProbeImplementationSource("Arena", implementation)
}

// WindowsMSVCArenaProbeImplementationSource returns the target-specific Windows Arena closure with
// the same probe-only deterministic SystemHeap allocator adapter.
func WindowsMSVCArenaProbeImplementationSource() (string, error) {
	implementation, err := WindowsMSVCSourceForSymbols(arenaProbeSymbols())
	if err != nil {
		return "", err
	}
	return systemHeapProbeImplementationSource("Windows MSVC Arena", implementation)
}

// TextViewSource returns the allocation-free UTF-8 validation and slicing implementation.
func TextViewSource() (string, error) {
	return canonicalSource("text view", stage0TextView)
}

// TextViewProbeSource returns the closed native conformance probe for UTF-8 text views.
func TextViewProbeSource() (string, error) {
	return canonicalSource("text view probe", stage0TextViewProbe)
}

// OwnedTextSource returns SystemHeap, Buffer, text-view, String, and CString as one C unit.
func OwnedTextSource() (string, error) {
	storage, err := StorageSource()
	if err != nil {
		return "", err
	}
	view, err := TextViewSource()
	if err != nil {
		return "", err
	}
	owned, err := canonicalSource("owned text", stage0OwnedText)
	if err != nil {
		return "", err
	}
	return storage + "\n" + view + "\n" + owned, nil
}

// OwnedTextProbeSource returns the closed native conformance probe for String and CString.
func OwnedTextProbeSource() (string, error) {
	return canonicalSource("owned text probe", stage0OwnedTextProbe)
}

// OwnedTextProbeImplementationSource returns the portable owned-text implementation with the
// probe-only deterministic allocator adapter. The adapter is never part of product linkage.
func OwnedTextProbeImplementationSource() (string, error) {
	implementation, err := OwnedTextSource()
	if err != nil {
		return "", err
	}
	return systemHeapProbeImplementationSource("owned text", implementation)
}

// WindowsMSVCOwnedTextProbeImplementationSource returns the target-specific Windows owned-text
// closure with the same probe-only deterministic allocator adapter.
func WindowsMSVCOwnedTextProbeImplementationSource() (string, error) {
	implementation, err := WindowsMSVCSourceForSymbols(ownedTextStorageBridgeSymbols())
	if err != nil {
		return "", err
	}
	return systemHeapProbeImplementationSource("Windows MSVC owned text", implementation)
}

// LinuxFileSource returns the target-specific partial-I/O and consuming-close implementation.
func LinuxFileSource() (string, error) {
	return canonicalSource("Linux File", stage0LinuxFile)
}

// LinuxFileProbeSource returns the closed native conformance probe for Linux File.
func LinuxFileProbeSource() (string, error) {
	return canonicalSource("Linux File probe", stage0LinuxFileProbe)
}

// LinuxConsoleSource returns the target-specific bounded stdin, stdout, and stderr implementation.
func LinuxConsoleSource() (string, error) {
	return canonicalSource("Linux console", stage0LinuxConsole)
}

// LinuxConsoleProbeSource returns the closed native conformance probe for Linux stdin, stdout, and stderr.
func LinuxConsoleProbeSource() (string, error) {
	return canonicalSource("Linux console probe", stage0LinuxConsoleProbe)
}

// WindowsConsoleSource returns the target-specific bounded stdin, stdout, and stderr implementation.
func WindowsConsoleSource() (string, error) {
	return canonicalSource("Windows console", stage0WindowsConsole)
}

// WindowsFileSource returns the target-specific partial-I/O and consuming-close implementation.
func WindowsFileSource() (string, error) {
	return canonicalSource("Windows File", stage0WindowsFile)
}

// WindowsFileProbeSource returns the closed native conformance probe for Windows File.
func WindowsFileProbeSource() (string, error) {
	probe, err := canonicalSource("Windows File probe", stage0WindowsFileProbe)
	if err != nil {
		return "", err
	}
	copyProbe, err := canonicalSource("Windows File copy probe", stage0WindowsFileCopyProbe)
	if err != nil {
		return "", err
	}
	return probe + "\n" + copyProbe, nil
}

// WindowsProcessArgsSource returns the target-specific UTF-8 process argument implementation.
func WindowsProcessArgsSource() (string, error) {
	return canonicalSource("Windows process args", stage0WindowsProcessArgs)
}

// SourceForSymbols returns the minimal dependency-closed Stage 0 C implementation for the exact
// standard-library symbols called by one generated unit. An empty result needs no stdlib object.
func SourceForSymbols(symbols []string) (string, error) {
	components, err := cstdlibcontract.ComponentsForSymbols(symbols)
	if err != nil {
		return "", err
	}
	return sourceForComponents(components)
}

// WindowsMSVCSourceForSymbols returns only standard-library C components with
// current source-level x86_64-windows-msvc execution evidence.
func WindowsMSVCSourceForSymbols(symbols []string) (string, error) {
	components, err := cstdlibcontract.WindowsMSVCComponentsForSymbols(symbols)
	if err != nil {
		return "", err
	}
	source, err := sourceForComponents(components)
	if err != nil {
		return "", err
	}
	for _, component := range components {
		if component == cstdlibcontract.ComponentSystemHeap {
			return "#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)\n" + source, nil
		}
	}
	return source, nil
}

func ownedTextStorageBridgeSymbols() []string {
	return []string{
		"fy_std_string_copy_storage_v0",
		"fy_std_string_append_storage_v0",
		"fy_std_string_drop_storage_v0",
		"fy_std_string_from_buffer_storage_v0",
		"fy_std_cstring_from_text_storage_v0",
		"fy_std_cstring_drop_storage_v0",
	}
}

func arenaProbeSymbols() []string {
	return []string{
		"fy_std_arena_initialize_v0",
		"fy_std_region_begin_v0",
		"fy_std_region_allocate_zeroed_v0",
		"fy_std_region_copy_bytes_v0",
		"fy_std_region_reset_v0",
		"fy_std_arena_drop_v0",
	}
}

func systemHeapProbeImplementationSource(name, implementation string) (string, error) {
	adapter, err := canonicalSource("SystemHeap probe allocator", stage0SystemHeapProbeAllocator)
	if err != nil {
		return "", err
	}
	if implementation == "" || strings.Contains(implementation, "\r") || !strings.HasSuffix(implementation, "\n") {
		return "", fmt.Errorf("%s probe implementation C source is empty or non-canonical", name)
	}
	return adapter + "\n" + implementation, nil
}

func sourceForComponents(components []cstdlibcontract.Component) (string, error) {
	var output strings.Builder
	for _, component := range components {
		name, source, err := componentSource(component)
		if err != nil {
			return "", err
		}
		canonical, err := canonicalSource(name, source)
		if err != nil {
			return "", err
		}
		if output.Len() != 0 {
			output.WriteByte('\n')
		}
		output.WriteString(canonical)
	}
	return output.String(), nil
}

func componentSource(component cstdlibcontract.Component) (string, string, error) {
	switch component {
	case cstdlibcontract.ComponentLinuxConsole:
		return "Linux console", stage0LinuxConsole, nil
	case cstdlibcontract.ComponentLinuxFile:
		return "Linux File", stage0LinuxFile, nil
	case cstdlibcontract.ComponentWindowsConsole:
		return "Windows console", stage0WindowsConsole, nil
	case cstdlibcontract.ComponentWindowsFile:
		return "Windows File", stage0WindowsFile, nil
	case cstdlibcontract.ComponentWindowsFileBuffer:
		return "Windows File Buffer adapter", stage0WindowsFileBuffer, nil
	case cstdlibcontract.ComponentWindowsProcessArgs:
		return "Windows process args", stage0WindowsProcessArgs, nil
	case cstdlibcontract.ComponentSystemHeap:
		return "SystemHeap", stage0SystemHeap, nil
	case cstdlibcontract.ComponentBuffer:
		return "Buffer", stage0Buffer, nil
	case cstdlibcontract.ComponentArena:
		return "Arena", stage0Arena, nil
	case cstdlibcontract.ComponentTextView:
		return "text view", stage0TextView, nil
	case cstdlibcontract.ComponentOwnedText:
		return "owned text", stage0OwnedText, nil
	default:
		return "", "", fmt.Errorf("unsupported Stage 0 standard-library component %q", component)
	}
}

func canonicalSource(name, source string) (string, error) {
	if source == "" || strings.Contains(source, "\r") || !strings.HasSuffix(source, "\n") {
		return "", fmt.Errorf("%s C source is empty or non-canonical", name)
	}
	return source, nil
}
