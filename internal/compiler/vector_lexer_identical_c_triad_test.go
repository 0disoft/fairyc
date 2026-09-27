package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

// The middle call in each triad is an identical C scanner call, not Fairy.
// Its ratio to the two surrounding C calls measures timing-position bias.
func TestPrivateVectorSDKCandidateIdenticalCScannerTriads(t *testing.T) {
	if os.Getenv("FAIRYC_SCANNER_IDENTICAL_C_TRIADS") != "1" {
		t.Skip("opt-in identical-C scanner triad control")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("Windows AMD64 MSVC required")
	}
	smokeMode := os.Getenv("FAIRYC_SCANNER_IDENTICAL_C_TRIADS_SMOKE")
	if smokeMode != "" && smokeMode != "1" {
		t.Fatalf("unknown identical-C triad smoke mode %q", smokeMode)
	}
	smoke := smokeMode == "1"
	highPriority := scannerTriadHighPriority(t)
	lowestAllowed := scannerTriadLowestAllowed(t, highPriority)
	triads, rounds := 24, 250000
	if smoke {
		triads, rounds = 1, 1000
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
		t.Fatal("triad scanner sample lost cooked or diagnostic coverage")
	}
	expected := perRound * rounds
	support := sameImageCycleAffinitySupport
	mainSource := identicalCScannerTriadMain
	if highPriority {
		if strings.Count(mainSource, "fc_pair_pin_thread()") != 1 {
			t.Fatal("C-only triad pin call changed")
		}
		pin := "fc_pair_pin_thread_priority()"
		if lowestAllowed {
			support += scannerTriadLowestAllowedSupport
			pin = "fc_pair_pin_lowest_allowed_priority()"
		} else {
			support += scannerTriadHighPrioritySupport
		}
		mainSource = strings.Replace(mainSource, "fc_pair_pin_thread()", pin, 1)
	}
	cSource := "#include <windows.h>\n" + sameImageCScannerSupport(t) +
		sameImageCRoundsSupport + support + mainSource
	for _, replacement := range [][2]string{
		{"@@TRIADS@@", strconv.Itoa(triads)},
		{"@@ROUNDS@@", strconv.Itoa(rounds)},
		{"@@EXPECTED@@", strconv.Itoa(expected)},
	} {
		if !strings.Contains(cSource, replacement[0]) {
			t.Fatalf("missing triad source placeholder %s", replacement[0])
		}
		cSource = strings.ReplaceAll(cSource, replacement[0], replacement[1])
	}
	if strings.Contains(cSource, "@@") {
		t.Fatal("unsubstituted triad source placeholder")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, cdriver.OptimizedProfile, cSource)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := cdriver.ExecuteWindowsHost(artifact)
	if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
		t.Fatalf("identical-C scanner triads: %+v %v", outcome, err)
	}
	fields := strings.Fields(outcome.Stdout)
	if len(fields) != 2+2*triads*5 {
		t.Fatalf("triad fields=%d want=%d: %q", len(fields), 2+2*triads*5, outcome.Stdout)
	}
	frequency, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil || frequency == 0 {
		t.Fatalf("invalid QPC frequency %q: %v", fields[0], err)
	}
	pinnedProcessor, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		t.Fatalf("invalid pinned processor %q: %v", fields[1], err)
	}
	runs := make([]map[string]any, 0, 2)
	for run := 0; run < 2; run++ {
		rows := make([]map[string]any, 0, triads)
		ratios := make([]float64, 0, triads)
		for triad := 0; triad < triads; triad++ {
			base := 2 + (run*triads+triad)*5
			values := [5]uint64{}
			for i := range values {
				values[i], err = strconv.ParseUint(fields[base+i], 10, 64)
				if err != nil {
					t.Fatalf("run %d triad %d field %d: %v", run, triad, i, err)
				}
			}
			if values[0] != uint64(run) || values[1] != uint64(triad) || values[2] == 0 || values[3] == 0 || values[4] == 0 {
				t.Fatalf("invalid run %d triad %d row: %v", run, triad, values)
			}
			ratios = append(ratios, float64(values[3])/math.Sqrt(float64(values[2])*float64(values[4])))
			rows = append(rows, map[string]any{
				"triad": triad, "before_qpc_ticks": values[2],
				"middle_qpc_ticks": values[3], "after_qpc_ticks": values[4],
			})
		}
		sort.Float64s(ratios)
		median := (ratios[(triads-1)/2] + ratios[triads/2]) / 2
		span := (ratios[triads-1] - ratios[0]) / median
		runReport := map[string]any{
			"run": run, "samples": rows, "median_middle_over_c": median,
			"relative_full_span": span,
		}
		if !smoke {
			runReport["passes_null_control"] = span <= 0.10 && math.Abs(median-1) <= 0.02
		}
		runs = append(runs, runReport)
	}
	report := map[string]any{
		"schema": "fairyc.identical-c-scanner-triads/v1",
		"go":     runtime.Version(), "msvc": artifact.Toolchain.Version,
		"profile": cdriver.OptimizedProfile.Name, "qpc_frequency": frequency,
		"affinity": "current-logical-processor", "pinned_logical_processor": pinnedProcessor,
		"triads_per_run": triads, "rounds_per_call": rounds,
		"timing_gate_eligible": !smoke, "expected_result_per_call": expected,
		"source_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(cSource))),
		"exe_sha256":    fmt.Sprintf("%x", sha256.Sum256(artifact.Bytes)),
		"runs":          runs,
		"scope":         "Two complete C-C-C runs on one pinned logical processor; one untimed warmup triad per run. The middle/geo-mean(outer calls) null ratio uses every triad; each call checks the Go-oracle work checksum. No retry or exclusion. This does not measure Fairy/C runtime or executable size.",
	}
	if highPriority {
		report["thread_priority"] = "highest"
	}
	if lowestAllowed {
		report["affinity"] = "lowest-allowed-logical-processor"
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("IDENTICAL_C_SCANNER_TRIADS " + string(encoded))
}

func scannerTriadHighPriority(t *testing.T) bool {
	t.Helper()
	mode := os.Getenv("FAIRYC_SCANNER_TRIAD_PRIORITY")
	if mode != "" && mode != "highest" {
		t.Fatalf("unknown scanner triad priority %q", mode)
	}
	return mode == "highest"
}

func scannerTriadLowestAllowed(t *testing.T, highPriority bool) bool {
	t.Helper()
	mode := os.Getenv("FAIRYC_SCANNER_TRIAD_AFFINITY")
	if mode != "" && mode != "lowest-allowed" {
		t.Fatalf("unknown scanner triad affinity %q", mode)
	}
	if mode == "lowest-allowed" && !highPriority {
		t.Fatal("lowest-allowed affinity requires checked high priority")
	}
	return mode == "lowest-allowed"
}

const scannerTriadHighPrioritySupport = `
size_t fc_pair_pin_thread_priority(void) {
    size_t pinned = fc_pair_pin_thread();
    if (!SetThreadPriority(GetCurrentThread(), THREAD_PRIORITY_HIGHEST) ||
        GetThreadPriority(GetCurrentThread()) != THREAD_PRIORITY_HIGHEST) abort();
    return pinned;
}
`

const scannerTriadLowestAllowedSupport = `
size_t fc_pair_pin_lowest_allowed_priority(void) {
    GROUP_AFFINITY allowed, selected;
    if (!GetThreadGroupAffinity(GetCurrentThread(), &allowed)) abort();
    unsigned number = 0;
    for (; number < 64u; ++number) {
        if ((allowed.Mask & (((KAFFINITY)1) << number)) != 0) break;
    }
    if (number == 64u) abort();
    selected = allowed;
    selected.Mask = ((KAFFINITY)1) << number;
    if (!SetThreadGroupAffinity(GetCurrentThread(), &selected, NULL) ||
        !SetThreadPriority(GetCurrentThread(), THREAD_PRIORITY_HIGHEST) ||
        GetThreadPriority(GetCurrentThread()) != THREAD_PRIORITY_HIGHEST) abort();
    return (size_t)allowed.Group * 64u + number;
}
`

const identicalCScannerTriadMain = `
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

static uint64_t fc_triad_timed_call(void) {
    LARGE_INTEGER before, after;
    if (!QueryPerformanceCounter(&before)) abort();
    size_t result = fc_same_image_work_rounds(@@ROUNDS@@);
    if (!QueryPerformanceCounter(&after) || result != @@EXPECTED@@ || after.QuadPart <= before.QuadPart) abort();
    return (uint64_t)(after.QuadPart - before.QuadPart);
}

int main(void) {
    LARGE_INTEGER frequency;
    if (!QueryPerformanceFrequency(&frequency) || frequency.QuadPart <= 0) return 1;
    size_t pinned = fc_pair_pin_thread();
    printf("%llu %zu\n", (unsigned long long)frequency.QuadPart, pinned);
    for (int run = 0; run < 2; ++run) {
        if (fc_same_image_work_rounds(@@ROUNDS@@) != @@EXPECTED@@ ||
            fc_same_image_work_rounds(@@ROUNDS@@) != @@EXPECTED@@ ||
            fc_same_image_work_rounds(@@ROUNDS@@) != @@EXPECTED@@) abort();
        for (int triad = 0; triad < @@TRIADS@@; ++triad) {
            uint64_t before = fc_triad_timed_call();
            uint64_t middle = fc_triad_timed_call();
            uint64_t after = fc_triad_timed_call();
            printf("%d %d %llu %llu %llu\n", run, triad,
                (unsigned long long)before,
                (unsigned long long)middle,
                (unsigned long long)after);
        }
    }
    return 0;
}
`
