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
