package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/workspaceloader"
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

func TestGeneratedFairyBufferInitialFailureProbeWrapsOnlyTheGeneratedEntry(t *testing.T) {
	t.Parallel()
	generated := "#include <stddef.h>\nint fy_std_buffer_initialize_v0(void);\nint fy_std_buffer_drop_v0(void);\nint main(void) { return 42; }\n"

	probe, err := generatedFairyBufferInitialFailureProbeSource(generated)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"#define main fy_generated_buffer_initial_failure_main",
		generated,
		"fy_std_system_heap_probe_fail_after_allocations_v0(0)",
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

func TestGeneratedFairyBufferResizeFailureProbeWrapsOnlyTheGeneratedEntry(t *testing.T) {
	t.Parallel()
	generated := "#include <stddef.h>\nint fy_std_buffer_initialize_v0(void);\nint fy_std_buffer_resize_zeroed_v0(void);\nint fy_std_buffer_drop_v0(void);\nint main(void) { return 42; }\n"

	probe, err := generatedFairyBufferResizeFailureProbeSource(generated)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"#define main fy_generated_buffer_resize_failure_main",
		generated,
		"fy_std_system_heap_probe_fail_after_allocations_v0(0)",
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

func TestGeneratedFairyBufferFailureProbeRejectsIncompleteGeneratedC(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		generated string
		wrap      func(string) (string, error)
	}{
		{name: "initial empty", generated: "", wrap: generatedFairyBufferInitialFailureProbeSource},
		{name: "initial missing entry", generated: "int fy_std_buffer_initialize_v0(void);\n", wrap: generatedFairyBufferInitialFailureProbeSource},
		{name: "initial missing Buffer calls", generated: "int main(void) { return 0; }\n", wrap: generatedFairyBufferInitialFailureProbeSource},
		{name: "resize empty", generated: "", wrap: generatedFairyBufferResizeFailureProbeSource},
		{name: "resize missing entry", generated: "int fy_std_buffer_resize_zeroed_v0(void);\n", wrap: generatedFairyBufferResizeFailureProbeSource},
		{name: "resize missing Buffer calls", generated: "int main(void) { return 0; }\n", wrap: generatedFairyBufferResizeFailureProbeSource},
	} {
		if _, err := tc.wrap(tc.generated); err == nil {
			t.Fatalf("incomplete generated C %q was accepted for %s", tc.generated, tc.name)
		}
	}
}

func TestGeneratedFairyStringAppendFailureProbeWrapsOnlyTheGeneratedEntry(t *testing.T) {
	t.Parallel()
	generated := "#include <stddef.h>\nint fy_std_string_copy_storage_v0(void);\nint fy_std_string_append_storage_v0(void);\nint fy_std_string_drop_storage_v0(void);\nint main(void) { return 42; }\n"

	probe, err := generatedFairyStringAppendFailureProbeSource(generated)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"#define main fy_generated_string_append_failure_main",
		generated,
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

func TestGeneratedFairyStringAppendFailureProbeRejectsIncompleteGeneratedC(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		generated string
	}{
		{name: "empty", generated: ""},
		{name: "missing main", generated: "int fy_std_string_copy_storage_v0(void);\nint fy_std_string_append_storage_v0(void);\nint fy_std_string_drop_storage_v0(void);\n"},
		{name: "missing copy", generated: "int main(void) { return 0; }\nint fy_std_string_append_storage_v0(void);\nint fy_std_string_drop_storage_v0(void);\n"},
		{name: "missing append", generated: "int main(void) { return 0; }\nint fy_std_string_copy_storage_v0(void);\nint fy_std_string_drop_storage_v0(void);\n"},
		{name: "missing drop", generated: "int main(void) { return 0; }\nint fy_std_string_copy_storage_v0(void);\nint fy_std_string_append_storage_v0(void);\n"},
	} {
		if _, err := generatedFairyStringAppendFailureProbeSource(tc.generated); err == nil {
			t.Fatalf("incomplete generated C %q was accepted for %s", tc.generated, tc.name)
		}
	}
}

func TestGeneratedFairyCStringFailureProbeWrapsOnlyTheGeneratedEntry(t *testing.T) {
	t.Parallel()
	generated := `#include <stddef.h>
int fy_std_cstring_from_text_storage_v0(void);
int fy_std_cstring_drop_storage_v0(void);
int main(void) {
    int from_text = fy_std_cstring_from_text_storage_v0();
    int dropped = fy_std_cstring_drop_storage_v0();
    return 42;
}
`

	probe, err := generatedFairyCStringFailureProbeSource(generated)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"#define main fy_generated_cstring_failure_main",
		generated,
		"fy_generated_cstring_failure_main() != 42",
		"fy_std_system_heap_probe_fail_after_allocations_v0(0)",
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

func TestGeneratedFairyCStringFailureProbeRejectsIncompleteGeneratedC(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		generated string
	}{
		{name: "empty", generated: ""},
		{name: "missing main", generated: "int fy_std_cstring_from_text_storage_v0(void);\nint fy_std_cstring_drop_storage_v0(void);\n"},
		{name: "missing from-text", generated: "int main(void) { return 0; }\nint fy_std_cstring_drop_storage_v0(void);\n"},
		{name: "missing drop", generated: "int main(void) { return 0; }\nint fy_std_cstring_from_text_storage_v0(void);\n"},
		{name: "from-text prototype only", generated: "#include <stddef.h>\nint fy_std_cstring_from_text_storage_v0(void);\nint fy_std_cstring_drop_storage_v0(void);\nint main(void) { int dropped = fy_std_cstring_drop_storage_v0(); return 42; }\n"},
		{name: "drop prototype only", generated: "#include <stddef.h>\nint fy_std_cstring_from_text_storage_v0(void);\nint fy_std_cstring_drop_storage_v0(void);\nint main(void) { int from_text = fy_std_cstring_from_text_storage_v0(); return 42; }\n"},
	} {
		if _, err := generatedFairyCStringFailureProbeSource(tc.generated); err == nil {
			t.Fatalf("incomplete generated C %q was accepted for %s", tc.generated, tc.name)
		}
	}
}

func TestGeneratedFairyStringCopyFailureProbeWrapsOnlyTheGeneratedEntry(t *testing.T) {
	t.Parallel()
	generated := `#include <stddef.h>
int fy_std_string_copy_storage_v0(void);
int fy_std_string_drop_storage_v0(void);
int main(void) {
    int copied = fy_std_string_copy_storage_v0();
    int dropped = fy_std_string_drop_storage_v0();
    return 42;
}
`

	probe, err := generatedFairyStringCopyFailureProbeSource(generated)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"#define main fy_generated_string_copy_failure_main",
		generated,
		"fy_generated_string_copy_failure_main() != 42",
		"fy_std_system_heap_probe_fail_after_allocations_v0(0)",
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

func TestGeneratedFairyStringCopyFailureProbeRejectsIncompleteGeneratedC(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		generated string
	}{
		{name: "empty", generated: ""},
		{name: "missing main", generated: "int fy_std_string_copy_storage_v0(void);\nint fy_std_string_drop_storage_v0(void);\n"},
		{name: "missing copy", generated: "int main(void) { return 0; }\nint fy_std_string_drop_storage_v0(void);\n"},
		{name: "missing drop", generated: "int main(void) { return 0; }\nint fy_std_string_copy_storage_v0(void);\n"},
		{name: "copy prototype only", generated: "#include <stddef.h>\nint fy_std_string_copy_storage_v0(void);\nint fy_std_string_drop_storage_v0(void);\nint main(void) { int dropped = fy_std_string_drop_storage_v0(); return 42; }\n"},
		{name: "drop prototype only", generated: "#include <stddef.h>\nint fy_std_string_copy_storage_v0(void);\nint fy_std_string_drop_storage_v0(void);\nint main(void) { int copied = fy_std_string_copy_storage_v0(); return 42; }\n"},
	} {
		if _, err := generatedFairyStringCopyFailureProbeSource(tc.generated); err == nil {
			t.Fatalf("incomplete generated C %q was accepted for %s", tc.generated, tc.name)
		}
	}
}

func TestGeneratedFairyStringFromBufferProbeWrapsOnlyTheGeneratedEntry(t *testing.T) {
	t.Parallel()
	generated := `#include <stddef.h>
int fy_std_string_from_buffer_storage_v0(void);
int fy_std_string_drop_storage_v0(void);
int fy_std_buffer_drop_v0(void);
int main(void) {
    int converted = fy_std_string_from_buffer_storage_v0();
    int dropped = fy_std_string_drop_storage_v0();
    int buffer_dropped = fy_std_buffer_drop_v0();
    return 42;
}
`

	probe, err := generatedFairyStringFromBufferProbeSource(generated)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"#define main fy_generated_string_from_buffer_main",
		generated,
		"fy_generated_string_from_buffer_main() != 42",
		"fy_std_system_heap_probe_fail_after_allocations_v0(2)",
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

func TestGeneratedFairyStringFromBufferProbeRejectsIncompleteGeneratedC(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		generated string
	}{
		{name: "empty", generated: ""},
		{name: "missing main", generated: "int fy_std_string_from_buffer_storage_v0(void);\nint fy_std_string_drop_storage_v0(void);\nint fy_std_buffer_drop_v0(void);\n"},
		{name: "missing from-buffer", generated: "int main(void) { return 0; }\nint fy_std_string_drop_storage_v0(void);\nint fy_std_buffer_drop_v0(void);\n"},
		{name: "missing string drop", generated: "int main(void) { return 0; }\nint fy_std_string_from_buffer_storage_v0(void);\nint fy_std_buffer_drop_v0(void);\n"},
		{name: "missing buffer drop", generated: "int main(void) { return 0; }\nint fy_std_string_from_buffer_storage_v0(void);\nint fy_std_string_drop_storage_v0(void);\n"},
		{name: "from-buffer prototype only", generated: "#include <stddef.h>\nint fy_std_string_from_buffer_storage_v0(void);\nint fy_std_string_drop_storage_v0(void);\nint fy_std_buffer_drop_v0(void);\nint main(void) { int dropped = fy_std_string_drop_storage_v0(); int buffer_dropped = fy_std_buffer_drop_v0(); return 42; }\n"},
		{name: "string drop prototype only", generated: "#include <stddef.h>\nint fy_std_string_from_buffer_storage_v0(void);\nint fy_std_string_drop_storage_v0(void);\nint fy_std_buffer_drop_v0(void);\nint main(void) { int converted = fy_std_string_from_buffer_storage_v0(); int buffer_dropped = fy_std_buffer_drop_v0(); return 42; }\n"},
		{name: "buffer drop prototype only", generated: "#include <stddef.h>\nint fy_std_string_from_buffer_storage_v0(void);\nint fy_std_string_drop_storage_v0(void);\nint fy_std_buffer_drop_v0(void);\nint main(void) { int converted = fy_std_string_from_buffer_storage_v0(); int dropped = fy_std_string_drop_storage_v0(); return 42; }\n"},
	} {
		if _, err := generatedFairyStringFromBufferProbeSource(tc.generated); err == nil {
			t.Fatalf("incomplete generated C %q was accepted for %s", tc.generated, tc.name)
		}
	}
}

func TestGeneratedFairyReadAppendFailureProbeRejectsIncompleteGeneratedC(t *testing.T) {
	t.Parallel()
	completeCalls := []string{
		"int fy_std_windows_file_open_v0(void) { return 0; }",
		"int fy_std_windows_file_read_append_v0(void) { return 0; }",
		"int fy_std_windows_file_close_v0(void) { return 0; }",
		"int fy_std_buffer_initialize_v0(void) { return 0; }",
		"int fy_std_buffer_drop_v0(void) { return 0; }",
	}
	for index, name := range []string{"open", "read_append", "close", "initialize", "drop"} {
		calls := append([]string(nil), completeCalls...)
		calls = append(calls[:index], calls[index+1:]...)
		generated := strings.Join(calls, "\n") + "\nint main(void) { return 42; }\n"
		if _, err := generatedFairyReadAppendFailureProbeSource(generated); err == nil {
			t.Fatalf("read_append failure probe accepted generated C missing %s", name)
		}
	}
}

func TestGeneratedFairyReadToEndFailureProbeRejectsIncompleteGeneratedC(t *testing.T) {
	t.Parallel()
	completeCalls := []string{
		"int fy_std_windows_file_open_v0(void) { return 0; }",
		"int fy_std_windows_file_read_append_v0(void) { return 0; }",
		"int fy_std_windows_file_position_v0(void) { return 0; }",
		"int fy_std_windows_file_close_v0(void) { return 0; }",
		"int fy_std_buffer_initialize_v0(void) { return 0; }",
		"int fy_std_buffer_drop_v0(void) { return 0; }",
	}
	for index, name := range []string{"open", "read_append", "position", "close", "initialize", "drop"} {
		calls := append([]string(nil), completeCalls...)
		calls = append(calls[:index], calls[index+1:]...)
		generated := strings.Join(calls, "\n") + "\nint main(void) { return 42; }\n"
		if _, err := generatedFairyReadToEndFailureProbeSource(generated); err == nil {
			t.Fatalf("read_to_end failure probe accepted generated C missing %s", name)
		}
	}
}

func TestGeneratedFairyStdlibSourcesPassFullPipeline(t *testing.T) {
	t.Parallel()
	fromBufferMain, err := os.ReadFile(filepath.Join("..", "..", "examples", "std_text_from_buffer_quickstart", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	fsMain, err := os.ReadFile(filepath.Join("..", "..", "examples", "std_fs_quickstart", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	fsPathOps, err := os.ReadFile(filepath.Join("..", "..", "examples", "std_fs_quickstart", "src", "path_ops.fy"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name           string
		manifest       string
		source         string
		extraSources   map[string][]byte
		expect         []string
		foreignSymbols []string
	}{
		{
			name: "with_capacity initial allocation failure", manifest: "std_bytes_quickstart", source: generatedFairyBufferInitialFailureSource,
			expect:         []string{"fy_std_buffer_initialize_v0", "fy_std_buffer_drop_v0"},
			foreignSymbols: []string{"fy_std_buffer_drop_v0", "fy_std_buffer_initialize_v0"},
		},
		{
			name: "read_append growth allocation failure", manifest: "std_fs_quickstart", source: generatedFairyReadAppendFailureSource,
			expect: []string{
				"fy_std_buffer_drop_v0", "fy_std_buffer_initialize_v0",
				"fy_std_windows_file_close_v0", "fy_std_windows_file_drop_v0",
				"fy_std_windows_file_open_v0", "fy_std_windows_file_read_append_v0",
			},
			foreignSymbols: []string{
				"fy_std_buffer_drop_v0", "fy_std_buffer_initialize_v0",
				"fy_std_windows_file_close_v0", "fy_std_windows_file_drop_v0",
				"fy_std_windows_file_open_v0", "fy_std_windows_file_read_append_v0",
			},
		},
		{
			name: "read_to_end partial allocation failure", manifest: "std_fs_quickstart", source: generatedFairyReadToEndFailureSource,
			expect: []string{
				"fy_std_buffer_drop_v0", "fy_std_buffer_initialize_v0",
				"fy_std_windows_file_close_v0", "fy_std_windows_file_drop_v0",
				"fy_std_windows_file_open_v0", "fy_std_windows_file_position_v0",
				"fy_std_windows_file_read_append_v0",
			},
			foreignSymbols: []string{
				"fy_std_buffer_drop_v0", "fy_std_buffer_initialize_v0",
				"fy_std_windows_file_close_v0", "fy_std_windows_file_drop_v0",
				"fy_std_windows_file_open_v0", "fy_std_windows_file_position_v0",
				"fy_std_windows_file_read_append_v0",
			},
		},
		{
			name: "resize_zeroed growth failure", manifest: "std_bytes_quickstart", source: generatedFairyBufferResizeFailureSource,
			expect:         []string{"fy_std_buffer_initialize_v0", "fy_std_buffer_resize_zeroed_v0", "fy_std_buffer_drop_v0"},
			foreignSymbols: []string{"fy_std_buffer_drop_v0", "fy_std_buffer_initialize_v0", "fy_std_buffer_resize_zeroed_v0"},
		},
		{
			name: "region failure", manifest: "regions", source: generatedFairyRegionFailureSource,
			expect:         []string{"fy_std_region_allocate_zeroed_v0", "fy_std_region_reset_v0", "fy_std_arena_drop_v0", "fy_runtime_trap("},
			foreignSymbols: []string{"fy_std_arena_drop_v0", "fy_std_arena_initialize_v0", "fy_std_region_allocate_zeroed_v0", "fy_std_region_begin_v0", "fy_std_region_reset_v0"},
		},
		{
			name: "string append growth failure", manifest: "std_text_quickstart", source: generatedFairyStringAppendFailureSource,
			expect:         []string{"fy_std_string_copy_storage_v0", "fy_std_string_append_storage_v0", "fy_std_string_drop_storage_v0", "fy_runtime_trap("},
			foreignSymbols: []string{"fy_std_string_append_storage_v0", "fy_std_string_copy_storage_v0", "fy_std_string_drop_storage_v0"},
		},
		{
			name: "cstring from-text initial allocation failure", manifest: "std_text_quickstart", source: generatedFairyCStringFailureSource,
			expect:         []string{"fy_std_cstring_from_text_storage_v0", "fy_std_cstring_drop_storage_v0", "fy_runtime_trap("},
			foreignSymbols: []string{"fy_std_cstring_drop_storage_v0", "fy_std_cstring_from_text_storage_v0"},
		},
		{
			name: "string copy initial allocation failure", manifest: "std_text_quickstart", source: generatedFairyStringCopyFailureSource,
			expect:         []string{"fy_std_string_copy_storage_v0", "fy_std_string_drop_storage_v0", "fy_runtime_trap("},
			foreignSymbols: []string{"fy_std_string_copy_storage_v0", "fy_std_string_drop_storage_v0"},
		},
		{
			name: "string from buffer zero-copy used", manifest: "std_text_from_buffer_quickstart", source: string(fromBufferMain),
			expect:         []string{"fy_std_string_from_buffer_storage_v0", "fy_std_string_drop_storage_v0", "fy_std_buffer_drop_v0"},
			foreignSymbols: []string{"fy_std_buffer_append_v0", "fy_std_buffer_drop_v0", "fy_std_buffer_initialize_v0", "fy_std_string_drop_storage_v0", "fy_std_string_from_buffer_storage_v0"},
		},
		{
			name: "string from buffer unused", manifest: "std_text_from_buffer_quickstart", source: "module main;\nfn i32 main() { return 42; }\n",
			expect:         []string{},
			foreignSymbols: []string{},
		},
		{
			name: "windows std fs used", manifest: "std_fs_quickstart", source: string(fsMain),
			extraSources: map[string][]byte{"src/path_ops.fy": fsPathOps},
			expect: []string{
				"fy_std_buffer_append_v0", "fy_std_buffer_drop_v0", "fy_std_buffer_initialize_v0",
				"fy_std_windows_file_open_v0", "fy_std_windows_file_read_v0", "fy_std_windows_file_read_exact_v0", "fy_std_windows_file_position_v0",
				"fy_std_windows_file_remove_v0",
				"fy_std_windows_file_rename_v0",
				"fy_std_windows_file_read_append_v0",
				"fy_std_windows_file_seek_start_v0", "fy_std_windows_file_size_v0", "fy_std_windows_file_kind_v0", "fy_std_windows_file_id_v0", "fy_std_windows_file_set_len_v0", "fy_std_windows_file_write_all_v0", "fy_std_windows_file_write_v0",
				"fy_std_windows_file_sync_v0", "fy_std_windows_file_close_v0", "fy_std_windows_file_drop_v0",
			},
			foreignSymbols: []string{
				"fy_std_buffer_append_v0", "fy_std_buffer_drop_v0", "fy_std_buffer_initialize_v0",
				"fy_std_windows_file_close_v0", "fy_std_windows_file_drop_v0", "fy_std_windows_file_id_v0", "fy_std_windows_file_kind_v0", "fy_std_windows_file_open_v0",
				"fy_std_windows_file_position_v0", "fy_std_windows_file_read_append_v0", "fy_std_windows_file_read_exact_v0", "fy_std_windows_file_read_v0", "fy_std_windows_file_remove_v0", "fy_std_windows_file_rename_v0", "fy_std_windows_file_seek_start_v0",
				"fy_std_windows_file_set_len_v0", "fy_std_windows_file_size_v0", "fy_std_windows_file_sync_v0", "fy_std_windows_file_write_all_v0", "fy_std_windows_file_write_v0",
			},
		},
		{
			name: "windows std fs unused", manifest: "std_fs_quickstart", source: "module main;\nfn i32 main() { return 42; }\n",
			expect:         []string{},
			foreignSymbols: []string{},
		},
		{
			name: "process args used", manifest: "process_args_quickstart", source: generatedFairyProcessArgsSource,
			expect:         []string{"fy_std_process_arg_count_v0", "fy_std_process_arg_v0"},
			foreignSymbols: []string{"fy_std_process_arg_count_v0", "fy_std_process_arg_v0"},
		},
		{
			name: "process args unused", manifest: "process_args_quickstart", source: "module main;\nfn i32 main() { return 42; }\n",
			expect:         []string{},
			foreignSymbols: []string{},
		},
		{
			name: "io scalar u32 used", manifest: "std_io_scalar_quickstart",
			source:         "module main;\nimport std::io as io;\n@effects(console)\nfn i32 check() {\n    return switch (io::write_u32(7)) {\n        case .ok(written) : { if (written == 1) { 0 } else { 1 } }\n        case .err(_) : { 1 }\n    };\n}\n@effects(console)\nfn i32 main() { return check(); }\n",
			expect:         []string{"fy_std_stdout_write_u32_v0"},
			foreignSymbols: []string{"fy_std_stdout_write_u32_v0"},
		},
		{
			name: "io scalar usize used", manifest: "std_io_scalar_quickstart",
			source:         "module main;\nimport std::io as io;\n@effects(console)\nfn i32 check() {\n    return switch (io::write_usize(7)) {\n        case .ok(written) : { if (written == 1) { 0 } else { 1 } }\n        case .err(_) : { 1 }\n    };\n}\n@effects(console)\nfn i32 main() { return check(); }\n",
			expect:         []string{"fy_std_stdout_write_usize_v0"},
			foreignSymbols: []string{"fy_std_stdout_write_usize_v0"},
		},
		{
			name: "io scalar bool used", manifest: "std_io_scalar_quickstart",
			source:         "module main;\nimport std::io as io;\n@effects(console)\nfn i32 check() {\n    return switch (io::write_bool(true)) {\n        case .ok(written) : { if (written == 4) { 0 } else { 1 } }\n        case .err(_) : { 1 }\n    };\n}\n@effects(console)\nfn i32 main() { return check(); }\n",
			expect:         []string{"fy_std_stdout_write_all_v0"},
			foreignSymbols: []string{"fy_std_stdout_write_all_v0"},
		},
		{
			name: "io scalar unused", manifest: "std_io_scalar_quickstart", source: "module main;\nfn i32 main() { return 42; }\n",
			expect:         []string{},
			foreignSymbols: []string{},
		},
	} {
		manifestPath := filepath.Join("..", "..", "examples", tc.manifest, "fairy.toml")
		manifest, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatalf("%s: read manifest: %v", tc.name, err)
		}
		workspaceRoot := t.TempDir()
		sources := map[string][]byte{
			"fairy.toml":  manifest,
			"src/main.fy": []byte(tc.source),
		}
		for name, contents := range tc.extraSources {
			sources[name] = contents
		}
		for name, contents := range sources {
			destination := filepath.Join(workspaceRoot, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
				t.Fatalf("%s: create pipeline path %s: %v", tc.name, name, err)
			}
			if err := os.WriteFile(destination, contents, 0o600); err != nil {
				t.Fatalf("%s: write pipeline path %s: %v", tc.name, name, err)
			}
		}
		workspace, err := workspaceloader.Load(workspaceRoot)
		if err != nil {
			t.Fatalf("%s: load workspace: %v", tc.name, err)
		}
		input, err := workspace.SemanticInput()
		if err != nil {
			t.Fatalf("%s: prepare semantic input: %v", tc.name, err)
		}
		checked, err := compiler.CheckPackages(input)
		if err != nil {
			t.Fatalf("%s: compile: %v", tc.name, err)
		}
		if !checked.Verified || checked.Semantic.CompletedThrough != semantic.PhaseFlow || checked.Semantic.FailedPhase != semantic.PhaseNone ||
			len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
			t.Fatalf("%s: no clean verified MIR: phase=%q failed=%q diagnostics=%+v unsupported=%+v", tc.name, checked.Semantic.CompletedThrough, checked.Semantic.FailedPhase, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
		}
		rootPackage, err := workspace.RootPackageID()
		if err != nil {
			t.Fatalf("%s: resolve root package: %v", tc.name, err)
		}
		entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
		unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatalf("%s: emit C: %v", tc.name, err)
		}
		if !slices.Equal(unit.ForeignSymbols, tc.foreignSymbols) {
			t.Fatalf("%s: ForeignSymbols = %v, want the entry-reachable minimal set %v", tc.name, unit.ForeignSymbols, tc.foreignSymbols)
		}
		for _, fragment := range tc.expect {
			if !strings.Contains(unit.Source, fragment) {
				t.Fatalf("%s: emitted C lost required fragment %q", tc.name, fragment)
			}
		}
	}
}

func TestGeneratedFairyProbeImplementationJoinsScalarRuntime(t *testing.T) {
	t.Parallel()
	repoRoot := filepath.Join("..", "..")
	for _, name := range []string{"Region and Buffer closure"} {
		closure, err := generatedFairyProbeImplementation(repoRoot, []string{
			"fy_std_region_allocate_zeroed_v0",
			"fy_std_buffer_initialize_v0",
			"fy_std_string_copy_storage_v0",
			"fy_std_string_append_storage_v0",
			"fy_std_string_drop_storage_v0",
			"fy_std_stdout_write_all_v0",
			"fy_std_text_utf8_valid_v0",
		})
		if err != nil {
			t.Fatalf("%s: join scalar runtime: %v", name, err)
		}
		for _, fragment := range []string{
			"UINT32_C(128)",
			"void fy_runtime_trap(uint32_t category)",
			"fy_std_system_heap_probe_fail_after_allocations_v0",
			"fy_std_system_heap_probe_live_allocations_v0",
			"fy_std_region_allocate_zeroed_v0",
			"fy_std_buffer_initialize_v0",
			"fy_std_string_copy_storage_v0",
			"fy_std_string_append_storage_v0",
			"fy_std_string_drop_storage_v0",
			"fy_std_stdout_write_all_v0",
			"fy_std_text_utf8_valid_v0",
		} {
			if !strings.Contains(closure, fragment) {
				t.Fatalf("%s closure lost required fragment %q", name, fragment)
			}
		}
		if strings.Count(closure, "void fy_runtime_trap(") != 1 {
			t.Fatalf("%s closure must define fy_runtime_trap exactly once", name)
		}
		for _, load := range []func() (string, error){
			cstdlib.WindowsMSVCArenaProbeImplementationSource,
			cstdlib.WindowsMSVCBufferProbeImplementationSource,
		} {
			productImplementation, err := load()
			if err != nil {
				t.Fatalf("%s: load product implementation: %v", name, err)
			}
			if strings.Contains(productImplementation, "void fy_runtime_trap(") {
				t.Fatalf("%s product implementation must not define the scalar runtime trap", name)
			}
		}
	}
}

func TestGeneratedFairyProbeImplementationAcceptsProcessArgsSymbols(t *testing.T) {
	t.Parallel()
	closure, err := generatedFairyProbeImplementation(filepath.Join("..", ".."), []string{
		"fy_std_process_arg_count_v0",
		"fy_std_process_arg_v0",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"fy_std_process_arg_count_v0",
		"fy_std_process_arg_v0",
		"fy_std_system_heap_probe_fail_after_allocations_v0",
		"fy_std_system_heap_probe_live_allocations_v0",
		"void fy_runtime_trap(uint32_t category)",
	} {
		if !strings.Contains(closure, fragment) {
			t.Fatalf("process args closure lost required fragment %q", fragment)
		}
	}
	if strings.Count(closure, "void fy_runtime_trap(") != 1 {
		t.Fatalf("process args closure must define fy_runtime_trap exactly once")
	}
}

func TestGeneratedFairyProbeImplementationRejectsIncompleteOrDuplicate(t *testing.T) {
	t.Parallel()
	runtimeSource, err := cruntime.ScalarSource(generatedFairyProbeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Join("..", "..")
	for _, tc := range []struct {
		name       string
		sourceRoot string
		symbols    []string
	}{
		{name: "empty symbols", sourceRoot: repoRoot, symbols: []string{}},
		{name: "missing adapter source", sourceRoot: filepath.Join(repoRoot, "missing"), symbols: []string{"fy_std_buffer_initialize_v0"}},
	} {
		if _, err := generatedFairyProbeImplementation(tc.sourceRoot, tc.symbols); err == nil {
			t.Fatalf("incomplete or duplicate closure was accepted for %s", tc.name)
		}
	}
	adapterBytes, err := os.ReadFile(filepath.Join(repoRoot, "internal", "cstdlib", "stage0_system_heap_probe_allocator.c"))
	if err != nil {
		t.Fatal(err)
	}
	adapter := string(adapterBytes)
	for _, tc := range []struct {
		name           string
		adapter        string
		implementation string
		runtimeSource  string
	}{
		{name: "non-canonical adapter", adapter: "int x;\r\n", implementation: "int y;\n", runtimeSource: runtimeSource},
		{name: "adapter without failure controls", adapter: "int x;\n", implementation: "int y;\n", runtimeSource: runtimeSource},
		{name: "duplicate trap definition", adapter: adapter, implementation: runtimeSource, runtimeSource: runtimeSource},
		{name: "runtime without trap definition", adapter: adapter, implementation: "int y;\n", runtimeSource: "int z;\n"},
	} {
		if _, err := generatedFairyProbeClosureSource(tc.adapter, tc.implementation, tc.runtimeSource); err == nil {
			t.Fatalf("incomplete or duplicate closure was accepted for %s", tc.name)
		}
	}
}
