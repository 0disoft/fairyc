package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

// The QPC and current-thread CPU observations cover the same adjacent scanner
// calls. CPU time excludes descheduling; neither clock alone proves a C budget.
func TestPrivateVectorSDKCandidateDualScannerCPUTime(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_LEXER_CPU_TIME") != "1" {
		t.Skip("opt-in dual-scanner CPU-time comparison")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("Windows AMD64 MSVC required")
	}
	revision := os.Getenv("FAIRYC_VECTOR_LEXER_SOURCE")
	if revision == "" {
		t.Fatal("source revision required")
	}
	pairs, rounds := 12, 250000
	if os.Getenv("FAIRYC_VECTOR_LEXER_CPU_TIME_SMOKE") == "1" {
		pairs, rounds = 1, 1000
	}
	mainSource := dualScannerCPUMainSource(t)
	for _, replacement := range [][2]string{{"@@PAIRS@@", strconv.Itoa(pairs + 1)}, {"@@ROUNDS@@", strconv.Itoa(rounds)}} {
		if strings.Count(mainSource, replacement[0]) == 0 {
			t.Fatalf("missing source placeholder %s", replacement[0])
		}
		mainSource = strings.ReplaceAll(mainSource, replacement[0], replacement[1])
	}
	program, entry := dualScannerCheckedProgram(t, mainSource)
	support := dualScannerTimingSupport + dualScannerCPUTimeSupport
	artifact, unit := buildRealScannerVectorNative(t, program, entry, cdriver.OptimizedProfile, support)
	imageHash := fmt.Sprintf("%x", sha256.Sum256(artifact.Bytes))
	outcome, err := cdriver.ExecuteWindowsHost(artifact)
	if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
		t.Fatalf("CPU-time scanner execution: %+v %v", outcome, err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(artifact.Bytes)) != imageHash {
		t.Fatal("CPU-time scanner image changed during execution")
	}
	const sample = "let n=12345; /*open"
	lexed := lexer.Lex(source.NewFile("input.fy", []byte(sample)), source.NewBudget(source.DefaultLimits(), len(sample)))
	perRound := len(sample) + len(lexed.Tokens) + len(lexed.Diagnostics)
	for _, token := range lexed.Tokens {
		perRound += len(token.Cooked)
	}
	for _, diagnostic := range lexed.Diagnostics {
		perRound += len(diagnostic.Token)
	}
	if perRound <= len(sample) || len(lexed.Diagnostics) == 0 {
		t.Fatal("independent scanner checksum lost cooked or diagnostic coverage")
	}
	fields := strings.Fields(outcome.Stdout)
	if len(fields) != 1+(pairs+1)*6 {
		t.Fatalf("CPU-time scanner fields=%d want=%d: %q", len(fields), 1+(pairs+1)*6, outcome.Stdout)
	}
	frequency, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil || frequency == 0 {
		t.Fatalf("invalid QPC frequency %q: %v", fields[0], err)
	}
	rows := make([]map[string]any, 0, pairs+1)
	for pair := 0; pair <= pairs; pair++ {
		base := 1 + pair*6
		values := [6]uint64{}
		for i := range values {
			values[i], err = strconv.ParseUint(fields[base+i], 10, 64)
			if err != nil {
				t.Fatalf("pair %d field %d: %v", pair, i, err)
			}
		}
		if values[0] == 0 || values[1] == 0 || values[4] != uint64(perRound*rounds) || values[5] != uint64(perRound*rounds) {
			t.Fatalf("pair %d: qpc=%d,%d sums=%d,%d want=%d", pair, values[0], values[1], values[4], values[5], perRound*rounds)
		}
		rows = append(rows, map[string]any{
			"repetition": pair - 1, "warmup": pair == 0,
			"first":           []string{"index", "candidate"}[pair%2],
			"index_qpc_ticks": values[0], "candidate_qpc_ticks": values[1],
			"index_cpu_100ns": values[2], "candidate_cpu_100ns": values[3],
			"index_qpc_ns":     values[0] * 1000000000 / frequency,
			"candidate_qpc_ns": values[1] * 1000000000 / frequency,
			"result":           values[4],
		})
	}
	variants := realScannerVectorVariants(t)
	harness, err := os.ReadFile("vector_lexer_cpu_time_test.go")
	if err != nil {
		t.Fatal(err)
	}
	report := map[string]any{
		"schema": "fairyc.dual-scanner-cpu-time/v1", "compiler_source": revision,
		"go": runtime.Version(), "msvc": artifact.Toolchain.Version,
		"gomaxprocs": runtime.GOMAXPROCS(0), "profile": artifact.Profile.Name,
		"rounds": rounds, "warmup_pairs": 1, "measured_pairs": pairs,
		"per_round_result": perRound, "qpc_frequency": frequency,
		"cpu_unit_ns": 100,
		"exe_sha256":  imageHash, "exe_bytes": len(artifact.Bytes),
		"generated_c_bytes": len(unit.Source), "functions": len(unit.Functions),
		"foreign_symbols":          len(unit.ForeignSymbols),
		"indexed_scanner_sha256":   fmt.Sprintf("%x", sha256.Sum256([]byte(variants[0].scanner))),
		"candidate_scanner_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(variants[4].scanner))),
		"main_sha256":              fmt.Sprintf("%x", sha256.Sum256([]byte(mainSource))),
		"support_sha256":           fmt.Sprintf("%x", sha256.Sum256([]byte(support))),
		"harness_sha256":           fmt.Sprintf("%x", sha256.Sum256(harness)),
		"samples":                  rows,
		"scope":                    "The actual indexed and compiler-private candidate scanner modules coexist in one optimized image. Each pair times the same collect/storage work with QPC and GetThreadTimes for the calling thread. CPU units are 100ns of user plus kernel time; they exclude descheduling but retain frequency and core-migration effects. Output and cleanup are outside timed calls. The token reporter is not timed. This combined image is not a standalone-image size or independent-C budget result. No retries, exclusions or adaptive stopping.",
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("DUAL_SCANNER_CPU_TIME_REPORT " + string(encoded))
}

func dualScannerCPUMainSource(t *testing.T) string {
	t.Helper()
	src := dualScannerPairedMain
	for _, edit := range []struct {
		before, after string
		count         int
	}{
		{`@link_name("fc_pair_frequency") unsafe extern C fn usize pair_frequency();`, `@link_name("fc_pair_frequency") unsafe extern C fn usize pair_frequency();
@link_name("fc_pair_cpu_ticks") unsafe extern C fn usize pair_cpu_ticks();`, 1},
		{`var usize candidate_ticks = 0;`, `var usize candidate_ticks = 0;
        var usize indexed_cpu_ticks = 0;
        var usize candidate_cpu_ticks = 0;`, 1},
		{`let usize indexed_start = unsafe { pair_ticks() };`, `let usize indexed_cpu_start = unsafe { pair_cpu_ticks() };
            let usize indexed_start = unsafe { pair_ticks() };`, 2},
		{`let usize candidate_start = unsafe { pair_ticks() };`, `let usize candidate_cpu_start = unsafe { pair_cpu_ticks() };
            let usize candidate_start = unsafe { pair_ticks() };`, 2},
		{`let usize indexed_end = unsafe { pair_ticks() };`, `let usize indexed_end = unsafe { pair_ticks() };
            let usize indexed_cpu_end = unsafe { pair_cpu_ticks() };`, 2},
		{`let usize candidate_end = unsafe { pair_ticks() };`, `let usize candidate_end = unsafe { pair_ticks() };
            let usize candidate_cpu_end = unsafe { pair_cpu_ticks() };`, 2},
		{`indexed_ticks = indexed_end - indexed_start;`, `indexed_ticks = indexed_end - indexed_start;
            assert(indexed_cpu_end >= indexed_cpu_start);
            indexed_cpu_ticks = indexed_cpu_end - indexed_cpu_start;`, 2},
		{`candidate_ticks = candidate_end - candidate_start;`, `candidate_ticks = candidate_end - candidate_start;
            assert(candidate_cpu_end >= candidate_cpu_start);
            candidate_cpu_ticks = candidate_cpu_end - candidate_cpu_start;`, 2},
		{`number(indexed_ticks); number(candidate_ticks);`, `number(indexed_ticks); number(candidate_ticks);
        number(indexed_cpu_ticks); number(candidate_cpu_ticks);`, 1},
	} {
		if strings.Count(src, edit.before) != edit.count {
			t.Fatalf("CPU timing source anchor changed: %q", edit.before)
		}
		src = strings.ReplaceAll(src, edit.before, edit.after)
	}
	return src
}

const dualScannerCPUTimeSupport = `
#include <stdint.h>
_Static_assert(sizeof(size_t) == sizeof(uint64_t), "CPU-time probe requires 64-bit size_t");
size_t fc_pair_cpu_ticks(void) {
    FILETIME created, exited, kernel, user;
    if (!GetThreadTimes(GetCurrentThread(), &created, &exited, &kernel, &user)) abort();
    uint64_t kernel_ticks = ((uint64_t)kernel.dwHighDateTime << 32) | kernel.dwLowDateTime;
    uint64_t user_ticks = ((uint64_t)user.dwHighDateTime << 32) | user.dwLowDateTime;
    if (UINT64_MAX - kernel_ticks < user_ticks) abort();
    return (size_t)(kernel_ticks + user_ticks);
}
`
