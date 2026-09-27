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

// The two real-scanner source forms run in one image. Adjacent timed calls
// remove process-launch and cross-image scheduling from the paired comparison.
func TestPrivateVectorSDKCandidateDualScannerPairedExecution(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_LEXER_INPROCESS") != "1" {
		t.Skip("opt-in dual-scanner in-process comparison")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("Windows AMD64 MSVC required")
	}
	revision := os.Getenv("FAIRYC_VECTOR_LEXER_SOURCE")
	if revision == "" {
		t.Fatal("source revision required")
	}
	pairs, rounds := 12, 250000
	if os.Getenv("FAIRYC_VECTOR_LEXER_INPROCESS_SMOKE") == "1" {
		pairs, rounds = 1, 1000
	}
	mainSource := dualScannerPairedMain
	for _, replacement := range [][2]string{{"@@PAIRS@@", strconv.Itoa(pairs + 1)}, {"@@ROUNDS@@", strconv.Itoa(rounds)}} {
		if strings.Count(mainSource, replacement[0]) == 0 {
			t.Fatalf("missing source placeholder %s", replacement[0])
		}
		mainSource = strings.ReplaceAll(mainSource, replacement[0], replacement[1])
	}
	program, entry := dualScannerCheckedProgram(t, mainSource)
	artifact, unit := buildRealScannerVectorNative(t, program, entry, cdriver.OptimizedProfile, dualScannerTimingSupport)
	imageHash := fmt.Sprintf("%x", sha256.Sum256(artifact.Bytes))
	outcome, err := cdriver.ExecuteWindowsHost(artifact)
	if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
		t.Fatalf("paired scanner execution: %+v %v", outcome, err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(artifact.Bytes)) != imageHash {
		t.Fatal("paired scanner artifact changed during execution")
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
		t.Fatal("independent scanner oracle lost cooked or diagnostic coverage")
	}
	fields := strings.Fields(outcome.Stdout)
	if len(fields) != 1+(pairs+1)*4 {
		t.Fatalf("paired scanner output field count=%d want=%d: %q", len(fields), 1+(pairs+1)*4, outcome.Stdout)
	}
	frequency, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil || frequency == 0 {
		t.Fatalf("invalid QPC frequency %q: %v", fields[0], err)
	}
	rows := make([]map[string]any, 0, pairs+1)
	for pair := 0; pair <= pairs; pair++ {
		base := 1 + pair*4
		values := [4]uint64{}
		for i := range values {
			values[i], err = strconv.ParseUint(fields[base+i], 10, 64)
			if err != nil {
				t.Fatalf("pair %d field %d: %v", pair, i, err)
			}
		}
		if values[0] == 0 || values[1] == 0 || values[2] != uint64(perRound*rounds) || values[3] != uint64(perRound*rounds) {
			t.Fatalf("pair %d: ticks=%d,%d sums=%d,%d want=%d", pair, values[0], values[1], values[2], values[3], perRound*rounds)
		}
		rows = append(rows, map[string]any{
			"repetition": pair - 1, "warmup": pair == 0,
			"first":       []string{"index", "candidate"}[pair%2],
			"index_ticks": values[0], "candidate_ticks": values[1],
			"index_ns":     values[0] * 1000000000 / frequency,
			"candidate_ns": values[1] * 1000000000 / frequency,
			"result":       values[2],
		})
	}
	variants := realScannerVectorVariants(t)
	harness, err := os.ReadFile("vector_lexer_inprocess_test.go")
	if err != nil {
		t.Fatal(err)
	}
	report := map[string]any{
		"schema": "fairyc.dual-scanner-inprocess/v1", "compiler_source": revision,
		"go": runtime.Version(), "msvc": artifact.Toolchain.Version,
		"gomaxprocs": runtime.GOMAXPROCS(0), "profile": artifact.Profile.Name,
		"rounds": rounds, "warmup_pairs": 1, "measured_pairs": pairs,
		"per_round_result": perRound, "qpc_frequency": frequency,
		"exe_sha256": imageHash, "exe_bytes": len(artifact.Bytes),
		"generated_c_bytes": len(unit.Source), "functions": len(unit.Functions),
		"foreign_symbols":          len(unit.ForeignSymbols),
		"indexed_scanner_sha256":   fmt.Sprintf("%x", sha256.Sum256([]byte(variants[0].scanner))),
		"candidate_scanner_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(variants[4].scanner))),
		"main_sha256":              fmt.Sprintf("%x", sha256.Sum256([]byte(mainSource))),
		"support_sha256":           fmt.Sprintf("%x", sha256.Sum256([]byte(dualScannerTimingSupport))),
		"harness_sha256":           fmt.Sprintf("%x", sha256.Sum256(harness)),
		"samples":                  rows,
		"scope":                    "The actual indexed and compiler-private vector-loop scanner modules coexist in one optimized image. Each pair times equivalent collect/storage work on the same source text with QPC, alternating which lane runs first. Output and cleanup are outside the timed calls. The token reporter is not timed. This combined image is not a standalone-image size or C-relative budget result. No retries, exclusions or adaptive stopping; host load and CPU frequency remain uncontrolled.",
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("DUAL_SCANNER_INPROCESS_REPORT " + string(encoded))
}

const dualScannerTimingSupport = `
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stddef.h>
#include <stdlib.h>
size_t fc_pair_ticks(void) {
    LARGE_INTEGER value;
    if (!QueryPerformanceCounter(&value) || value.QuadPart < 0) abort();
    return (size_t)value.QuadPart;
}
size_t fc_pair_frequency(void) {
    LARGE_INTEGER value;
    if (!QueryPerformanceFrequency(&value) || value.QuadPart <= 0) abort();
    return (size_t)value.QuadPart;
}
`

const dualScannerPairedMain = `module main;
import indexed as indexed;
import candidate as candidate;
import std::bytes as bytes;
import std::io as io;
@link_name("fc_pair_ticks") unsafe extern C fn usize pair_ticks();
@link_name("fc_pair_frequency") unsafe extern C fn usize pair_frequency();
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
fn usize run_indexed(usize rounds) {
    var usize total = 0;
    var usize iteration = 0;
    while (iteration < rounds) {
        let bytes::Buffer owned = switch (bytes::copy(b"let n=12345; /*open")) {
            case .ok(value) : { value }
            case .err(_) : { assert(false); }
        };
        total += switch (indexed::collect(owned, false, 32768, 1048576, 256, 16777216)) {
            case .ok(stream) : {
                let indexed::Storage usage = indexed::storage(ref stream);
                usage.source_bytes + usage.cooked_bytes + usage.diagnostic_bytes + usage.tokens + usage.diagnostics
            }
            case .err(_) : { assert(false); }
        };
        iteration += 1;
    }
    return total;
}
fn usize run_candidate(usize rounds) {
    var usize total = 0;
    var usize iteration = 0;
    while (iteration < rounds) {
        let bytes::Buffer owned = switch (bytes::copy(b"let n=12345; /*open")) {
            case .ok(value) : { value }
            case .err(_) : { assert(false); }
        };
        total += switch (candidate::collect(owned, false, 32768, 1048576, 256, 16777216)) {
            case .ok(stream) : {
                let candidate::Storage usage = candidate::storage(ref stream);
                usage.source_bytes + usage.cooked_bytes + usage.diagnostic_bytes + usage.tokens + usage.diagnostics
            }
            case .err(_) : { assert(false); }
        };
        iteration += 1;
    }
    return total;
}
@effects(console) fn void number(usize value) {
    assert(switch (io::write_usize(value)) { case .ok(_) : { true } case .err(_) : { false } });
    assert(switch (io::write(" ")) { case .ok(_) : { true } case .err(_) : { false } });
}
@effects(console) fn i32 main() {
    let usize frequency = unsafe { pair_frequency() };
    assert(frequency > 0);
    number(frequency);
    var usize pair = 0;
    while (pair < @@PAIRS@@) {
        var usize indexed_ticks = 0;
        var usize candidate_ticks = 0;
        var usize indexed_sum = 0;
        var usize candidate_sum = 0;
        if (pair % 2 == 0) {
            let usize indexed_start = unsafe { pair_ticks() };
            indexed_sum = run_indexed(@@ROUNDS@@);
            let usize indexed_end = unsafe { pair_ticks() };
            assert(indexed_end >= indexed_start);
            indexed_ticks = indexed_end - indexed_start;
            let usize candidate_start = unsafe { pair_ticks() };
            candidate_sum = run_candidate(@@ROUNDS@@);
            let usize candidate_end = unsafe { pair_ticks() };
            assert(candidate_end >= candidate_start);
            candidate_ticks = candidate_end - candidate_start;
        } else {
            let usize candidate_start = unsafe { pair_ticks() };
            candidate_sum = run_candidate(@@ROUNDS@@);
            let usize candidate_end = unsafe { pair_ticks() };
            assert(candidate_end >= candidate_start);
            candidate_ticks = candidate_end - candidate_start;
            let usize indexed_start = unsafe { pair_ticks() };
            indexed_sum = run_indexed(@@ROUNDS@@);
            let usize indexed_end = unsafe { pair_ticks() };
            assert(indexed_end >= indexed_start);
            indexed_ticks = indexed_end - indexed_start;
        }
        number(indexed_ticks); number(candidate_ticks);
        number(indexed_sum); number(candidate_sum);
        pair += 1;
    }
    assert(unsafe { clean() } == 1);
    return 0;
}`
