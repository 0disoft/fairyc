package cstdlib

import (
	"strings"
	"testing"
)

func TestLinuxGNUVectorStorageSourceUsesLinuxHeap(t *testing.T) {
	heap, err := SourceForSymbols([]string{"fy_std_system_heap_allocate_v0"})
	if err != nil {
		t.Fatal(err)
	}
	storage, err := LinuxGNUVectorStorageSource()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(storage, heap+"\n") ||
		strings.Count(storage, "uint32_t fy_std_system_heap_allocate_v0(") != 1 ||
		strings.Count(storage, "uint32_t fy_private_vector_initialize_v0(") != 1 ||
		strings.Contains(storage, "#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)") {
		t.Fatal("Linux vector source is not one Linux heap followed by the private storage transport")
	}
	repeated, err := LinuxGNUVectorStorageSource()
	if err != nil || repeated != storage {
		t.Fatalf("Linux vector source is not deterministic: %v", err)
	}
}
