package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

// This observes SDK system-heap requests, not CRT internals, RSS or latency.
func TestWindowsFairyLexerStreamHeap(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	heavy := "module main;\n" + strings.Repeat("fn i32 f() { return 42; }\n", 600)
	heavy += strings.Repeat(" ", 16384-len(heavy))
	scenarios := []struct {
		name, input string
		fail        int
	}{
		{"empty", "", 0}, {"small", "module main; fn i32 f() { return 42; }", 0},
		{"token-heavy-16k", heavy, 0}, {"literal-10k", `"` + strings.Repeat("a", 10000) + `"`, 0},
		{"diagnostic", "/* unfinished", 0},
		{"fail-source", "true 42", 1}, {"fail-token-vector", "true 42", 2}, {"fail-cooked", "true 42", 3},
		{"fail-cooked-table", "true 42", 4},
		{"fail-token-growth", "module main; fn i32 f() { return 42; }", 3},
		{"fail-small-6", "module main; fn i32 f() { return 42; }", 6},
	}
	for _, scenario := range scenarios {
		root := t.TempDir()
		copyLexerStreamExample(t, root)
		program := `module main; import scanner as scanner; import std::bytes as bytes;
@link_name("fairyc_heap_snapshot") unsafe extern C fn void snapshot();
@link_name("fairyc_heap_finish") unsafe extern C fn i32 finish(i32 status);
fn i32 exercise() {
    var bool allocated = true;
    let bytes::Buffer input = switch (bytes::copy(b` + strconv.Quote(scenario.input) + `)) { case .ok(value): { value } case .err(_): { allocated = false; bytes::empty() } };
    if (!allocated) { return 3; }
    return switch (scanner::collect(input, false, 32768, 1048576, 256, 16777216)) {
        case .ok(stream): { assert(scanner::token_count(ref stream) > 0); unsafe { snapshot(); } 0 }
        case .err(status): { status }
    };
}
fn i32 main() { let i32 status = exercise(); return unsafe { finish(status) }; }`
		writeTestSource(t, root, "src/main.fy", program)
		var stderr strings.Builder
		lowered, code := lowerWorkspaceExecutable(root, false, workspaceCompilePolicy{stableDiagnostics: true}, "heap observation", &stderr)
		if code != ExitSuccess {
			t.Fatalf("lower %s: %d %s", scenario.name, code, stderr.String())
		}
		unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatal(err)
		}
		var symbols []string
		for _, symbol := range unit.ForeignSymbols {
			if !strings.HasPrefix(symbol, "fairyc_heap_") {
				symbols = append(symbols, symbol)
			}
		}
		support, err := windowsVectorSupportSource(symbols, false)
		if err != nil {
			t.Fatal(err)
		}
		support = fmt.Sprintf("#define HEAP_FAIL_AT %d\n", scenario.fail) + lexerHeapProbe + support
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			artifact, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, unit.Source, support)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, strings.NewReader(""))
			if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" {
				t.Fatalf("%s/%s: %+v %v", scenario.name, profile.Name, outcome, err)
			}
			var row struct {
				Attempts, Allocations, Frees, Failures, PeakBytes, RetainedBytes, LiveBytes, LiveBlocks uint64
				Status                                                                                  int
			}
			if err := json.Unmarshal([]byte(outcome.Stdout), &row); err != nil {
				t.Fatalf("heap output: %q %v", outcome.Stdout, err)
			}
			expected := 0
			if scenario.fail != 0 {
				expected = 3
			}
			if row.Status != expected || row.Allocations != row.Frees || row.LiveBytes != 0 || row.LiveBlocks != 0 || row.Attempts != row.Allocations+row.Failures || row.RetainedBytes > row.PeakBytes {
				t.Fatalf("heap ledger %s: %+v", scenario.name, row)
			}
			if scenario.fail != 0 && row.Failures != 1 {
				t.Fatalf("failure not observed: %+v", row)
			}
			if scenario.fail == 0 && (row.Failures != 0 || row.PeakBytes == 0 || row.RetainedBytes == 0) {
				t.Fatalf("empty observation: %+v", row)
			}
			// Same-request baselines from heap-baseline-2026-09-10.md. Keep the
			// memory win explicit without presenting this instrumented build as fast.
			baselinePeak := map[string]uint64{"empty": 448, "small": 2726, "token-heavy-16k": 1396488, "literal-10k": 75538, "diagnostic": 975}
			if peak, ok := baselinePeak[scenario.name]; ok && row.PeakBytes > peak {
				t.Fatalf("peak regression %s: %+v", scenario.name, row)
			}
			if scenario.name == "token-heavy-16k" && (row.PeakBytes > 1396488*3/4 || row.RetainedBytes > 938688*3/4 || row.Allocations > 1213*102/100) {
				t.Fatalf("sparse token storage budget: %+v", row)
			}
			record := map[string]any{"schema": "fairyc.lexer-heap-observation/v1", "scenario": scenario.name, "profile": profile.Name, "input_bytes": len(scenario.input), "cycles": 1, "counts": row, "go": runtime.Version(), "msvc": artifact.Toolchain.Version, "os": runtime.GOOS, "arch": runtime.GOARCH, "generated_c_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(unit.Source))), "support_c_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(support)))}
			encoded, _ := json.Marshal(record)
			t.Log(string(encoded))
		}
	}
}

const lexerHeapProbe = `
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
struct heap_slot { void *pointer; size_t bytes; };
static struct heap_slot heap_slots[32768];
static size_t heap_attempts, heap_allocations, heap_frees, heap_failures;
static size_t heap_live, heap_blocks, heap_peak, heap_retained;
static void *lexer_malloc(size_t size) {
    heap_attempts++;
    if (heap_attempts == HEAP_FAIL_AT) { heap_failures++; return NULL; }
    size_t slot = 0;
    while (slot < 32768 && heap_slots[slot].pointer != NULL) slot++;
    if (slot == 32768 || size > SIZE_MAX - heap_live) abort();
    void *data = malloc(size);
    if (data == NULL) { heap_failures++; return NULL; }
    heap_slots[slot].pointer = data; heap_slots[slot].bytes = size;
    heap_allocations++; heap_blocks++; heap_live += size;
    if (heap_live > heap_peak) heap_peak = heap_live;
    return data;
}
static void lexer_free(void *data) {
    if (data == NULL) return;
    size_t slot = 0;
    while (slot < 32768 && heap_slots[slot].pointer != data) slot++;
    if (slot == 32768 || heap_blocks == 0 || heap_live < heap_slots[slot].bytes) abort();
    heap_live -= heap_slots[slot].bytes; heap_slots[slot].pointer = NULL;
    heap_frees++; heap_blocks--; free(data);
}
#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) lexer_malloc(size)
#define FY_STD_SYSTEM_HEAP_FREE_V0(data) lexer_free(data)
void fairyc_heap_snapshot(void) { heap_retained = heap_live; }
int32_t fairyc_heap_finish(int32_t status) {
    printf("{\"Attempts\":%zu,\"Allocations\":%zu,\"Frees\":%zu,\"Failures\":%zu,\"PeakBytes\":%zu,\"RetainedBytes\":%zu,\"LiveBytes\":%zu,\"LiveBlocks\":%zu,\"Status\":%d}\n",
        heap_attempts,heap_allocations,heap_frees,heap_failures,heap_peak,heap_retained,heap_live,heap_blocks,(int)status);
    return heap_live == 0 && heap_blocks == 0 && heap_allocations == heap_frees ? 0 : 1;
}
`
