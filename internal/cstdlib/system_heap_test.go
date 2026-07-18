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
