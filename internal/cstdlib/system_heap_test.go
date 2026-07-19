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
		"alignment <= _Alignof(max_align_t)",
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
	if !strings.Contains(probe, "abcdefbcde") || !strings.Contains(probe, "SIZE_MAX") {
		t.Fatal("Buffer probe does not exercise overlap and overflow rollback")
	}
	repeated, err := StorageSource()
	if err != nil || repeated != implementation || strings.Contains(implementation, "\r") || !strings.HasSuffix(implementation, "\n") {
		t.Fatalf("StorageSource() is not deterministic canonical C: %v", err)
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
	if !strings.Contains(probe, "fy_std_region_begin_v0(&arena, &second)") ||
		!strings.Contains(probe, "fy_std_region_allocate_zeroed_v0(&region, 1, 1, &empty)") {
		t.Fatal("Arena probe does not exercise concurrent and stale Region rejection")
	}
	repeated, err := ArenaSource()
	if err != nil || repeated != implementation || strings.Contains(implementation, "\r") || !strings.HasSuffix(implementation, "\n") {
		t.Fatalf("ArenaSource() is not deterministic canonical C: %v", err)
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
	for _, required := range []string{
		"fy_std_string_from_buffer_v0",
		"FY_STD_OWNED_TEXT_INVALID_UTF8_V0",
		"fy_std_string_append_v0",
		"fy_std_cstr_validate_v0",
		"FY_STD_OWNED_TEXT_INTERIOR_NUL_V0",
		"fy_std_cstring_from_text_v0",
	} {
		if !strings.Contains(implementation, required) {
			t.Errorf("owned text implementation is missing %q", required)
		}
	}
	for _, required := range []string{"invalid_buffer.block.data != invalid_data", "invalid_cstr", "cview.data[cview.length] != 0"} {
		if !strings.Contains(probe, required) {
			t.Errorf("owned text probe is missing %q", required)
		}
	}
	repeated, err := OwnedTextSource()
	if err != nil || repeated != implementation || strings.Contains(implementation, "\r") || !strings.HasSuffix(implementation, "\n") {
		t.Fatalf("OwnedTextSource() is not deterministic canonical C: %v", err)
	}
}

func TestLinuxFileSourcesConsumeCloseAndExposePartialIO(t *testing.T) {
	t.Parallel()
	implementation, err := LinuxFileSource()
	if err != nil {
		t.Fatal(err)
	}
	probe, err := LinuxFileProbeSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"fy_std_linux_file_read_v0", "fy_std_linux_file_write_v0", "fy_std_linux_file_write_all_v0", "file->descriptor = -1", "(void)close(descriptor)"} {
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

func TestWindowsMSVCSourceForSymbolsKeepsClosedPortableSlice(t *testing.T) {
	t.Parallel()
	source, err := WindowsMSVCSourceForSymbols([]string{"fy_std_text_utf8_valid_v0"})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"fy_std_text_utf8_valid_v0", "fy_std_text_validate_utf8_v0"} {
		if !strings.Contains(source, required) {
			t.Errorf("Windows source is missing %q", required)
		}
	}
	for _, forbidden := range []string{"fy_std_system_heap_allocate_v0", "fy_std_linux_file_open_v0", "_POSIX_C_SOURCE"} {
		if strings.Contains(source, forbidden) {
			t.Errorf("Windows source unexpectedly contains %q", forbidden)
		}
	}
	for _, symbol := range []string{"fairyc_external_helper", "fy_std_system_heap_allocate_v0"} {
		if _, err := WindowsMSVCSourceForSymbols([]string{symbol}); err == nil {
			t.Errorf("WindowsMSVCSourceForSymbols(%q) unexpectedly succeeded", symbol)
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
