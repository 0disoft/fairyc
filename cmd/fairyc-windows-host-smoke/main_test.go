package main

import (
	"strings"
	"testing"
)

func TestBoundedBufferStoresPrefixWithoutShortWrite(t *testing.T) {
	t.Parallel()
	buffer := boundedBuffer{limit: 5}

	written, err := buffer.Write([]byte("Fairy C"))
	if err != nil {
		t.Fatal(err)
	}
	if written != len("Fairy C") {
		t.Fatalf("Write() = %d; want %d", written, len("Fairy C"))
	}
	if buffer.Len() != 5 || buffer.buffer.String() != "Fairy" || !buffer.truncated {
		t.Fatalf("buffer = len %d, raw %q, truncated %t", buffer.Len(), buffer.buffer.String(), buffer.truncated)
	}
	if buffer.String() != "Fairy\n[output truncated]" {
		t.Fatalf("String() = %q", buffer.String())
	}
}

func TestBoundedBufferAcceptsExactLimit(t *testing.T) {
	t.Parallel()
	buffer := boundedBuffer{limit: 5}

	written, err := buffer.Write([]byte("Fairy"))
	if err != nil {
		t.Fatal(err)
	}
	if written != 5 || buffer.Len() != 5 || buffer.String() != "Fairy" || buffer.truncated {
		t.Fatalf("buffer = written %d, len %d, value %q, truncated %t", written, buffer.Len(), buffer.String(), buffer.truncated)
	}
}

func TestGeneratedFairyRegionFailureProbeWrapsOnlyTheGeneratedEntry(t *testing.T) {
	t.Parallel()
	generated := "#include <stddef.h>\nint fy_std_region_allocate_zeroed_v0(void);\nint fy_std_region_reset_v0(void);\nint fy_std_arena_drop_v0(void);\nint main(void) { return 42; }\n"

	probe, err := generatedFairyRegionFailureProbeSource(generated)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"#define main fy_generated_region_failure_main",
		generated,
		"fy_std_system_heap_probe_fail_after_allocations_v0(0)",
		"fy_std_system_heap_probe_fail_after_allocations_v0(1)",
		"fy_std_system_heap_probe_live_allocations_v0() != 0",
	} {
		if !strings.Contains(probe, fragment) {
			t.Fatalf("probe source lost required fragment %q", fragment)
		}
	}
	if strings.Count(probe, "int main(void)") != 2 {
		t.Fatalf("probe source must contain the renamed generated entry and one test entry: %q", probe)
	}
}

func TestGeneratedFairyRegionFailureProbeRejectsIncompleteGeneratedC(t *testing.T) {
	t.Parallel()
	for _, generated := range []string{"", "int main(void) { return 0; }\n"} {
		if _, err := generatedFairyRegionFailureProbeSource(generated); err == nil {
			t.Fatalf("incomplete generated C %q was accepted", generated)
		}
	}
}
