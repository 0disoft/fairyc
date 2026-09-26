package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
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

// This C-only control must be run before interpreting a new C/Fairy scanner
// timing receipt on a host. Both timed lanes call the exact same C function.
func TestPrivateVectorSDKCandidateIdenticalCScannerControl(t *testing.T) {
	if os.Getenv("FAIRYC_SCANNER_IDENTICAL_C_CONTROL") != "1" {
		t.Skip("opt-in identical-C scanner timing control")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("Windows AMD64 MSVC required")
	}
	pairs, rounds := 24, 250000
	smokeMode := os.Getenv("FAIRYC_SCANNER_IDENTICAL_C_CONTROL_SMOKE")
	if smokeMode != "" && smokeMode != "1" {
		t.Fatalf("unknown identical-C control smoke mode %q", smokeMode)
	}
	smoke := smokeMode == "1"
	if smoke {
		pairs, rounds = 1, 1000
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
		t.Fatal("scanner control lost cooked or diagnostic coverage")
	}
	expected := perRound * rounds
	cSource := sameImageCScannerSupport(t) + sameImageCRoundsSupport + identicalCScannerControlMain
	cSource = strings.ReplaceAll(cSource, "@@PAIRS@@", strconv.Itoa(pairs))
	cSource = strings.ReplaceAll(cSource, "@@ROUNDS@@", strconv.Itoa(rounds))
	cSource = strings.ReplaceAll(cSource, "@@EXPECTED@@", strconv.Itoa(expected))
	if strings.Contains(cSource, "@@") {
		t.Fatal("unsubstituted scanner control source placeholder")
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
		t.Fatalf("identical-C scanner control: %+v %v", outcome, err)
	}
	fields := strings.Fields(outcome.Stdout)
	if len(fields) != 1+2*pairs*4 {
		t.Fatalf("control fields=%d want=%d: %q", len(fields), 1+2*pairs*4, outcome.Stdout)
	}
	frequency, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil || frequency == 0 {
		t.Fatalf("invalid QPC frequency %q: %v", fields[0], err)
	}
	runs := make([]map[string]any, 0, 2)
	for run := 0; run < 2; run++ {
		rows := make([]map[string]any, 0, pairs)
		ratios := make([]float64, 0, pairs)
		for pair := 0; pair < pairs; pair++ {
			base := 1 + (run*pairs+pair)*4
			values := [4]uint64{}
			for i := range values {
				values[i], err = strconv.ParseUint(fields[base+i], 10, 64)
				if err != nil {
					t.Fatalf("run %d pair %d field %d: %v", run, pair, i, err)
				}
			}
			if values[0] != uint64(run) || values[1] != uint64(pair) || values[2] == 0 || values[3] == 0 {
				t.Fatalf("invalid run %d pair %d control row: %v", run, pair, values)
			}
			ratios = append(ratios, float64(values[3])/float64(values[2]))
			rows = append(rows, map[string]any{
				"pair": pair, "first": []string{"A", "B"}[pair%2],
				"a_qpc_ticks": values[2], "b_qpc_ticks": values[3],
			})
		}
		sort.Float64s(ratios)
		median := (ratios[(pairs-1)/2] + ratios[pairs/2]) / 2
		span := (ratios[pairs-1] - ratios[0]) / median
		runReport := map[string]any{
			"run": run, "samples": rows, "median_b_over_a": median,
			"relative_full_span": span,
		}
		if !smoke {
			runReport["passes_10_percent"] = span <= 0.10
		}
		runs = append(runs, runReport)
	}
	report := map[string]any{
		"schema": "fairyc.identical-c-scanner-control/v1",
		"go":     runtime.Version(), "msvc": artifact.Toolchain.Version,
		"profile": cdriver.OptimizedProfile.Name, "qpc_frequency": frequency,
		"pairs_per_run": pairs, "rounds_per_call": rounds,
		"timing_gate_eligible":     !smoke,
		"expected_result_per_call": expected,
		"source_sha256":            fmt.Sprintf("%x", sha256.Sum256([]byte(cSource))),
		"exe_sha256":               fmt.Sprintf("%x", sha256.Sum256(artifact.Bytes)),
		"runs":                     runs,
		"scope":                    "Two complete consecutive runs; one untimed warmup per run; alternating adjacent identical C calls. Every call checks the Go-oracle checksum. No pair is retried or discarded. This control does not establish Fairy/C runtime or size parity.",
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("IDENTICAL_C_SCANNER_CONTROL " + string(encoded))
}

const identicalCScannerControlMain = `
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

static uint64_t fc_control_timed_call(void) {
    LARGE_INTEGER before, after;
    if (!QueryPerformanceCounter(&before)) abort();
    size_t result = fc_same_image_work_rounds(@@ROUNDS@@);
    if (!QueryPerformanceCounter(&after) || result != @@EXPECTED@@ || after.QuadPart <= before.QuadPart) abort();
    return (uint64_t)(after.QuadPart - before.QuadPart);
}

int main(void) {
    LARGE_INTEGER frequency;
    if (!QueryPerformanceFrequency(&frequency) || frequency.QuadPart <= 0) return 1;
    printf("%llu\n", (unsigned long long)frequency.QuadPart);
    for (int run = 0; run < 2; ++run) {
        if (fc_same_image_work_rounds(@@ROUNDS@@) != @@EXPECTED@@) abort();
        for (int pair = 0; pair < @@PAIRS@@; ++pair) {
            uint64_t a, b;
            if ((pair & 1) == 0) {
                a = fc_control_timed_call();
                b = fc_control_timed_call();
            } else {
                b = fc_control_timed_call();
                a = fc_control_timed_call();
            }
            printf("%d %d %llu %llu\n", run, pair,
                (unsigned long long)a, (unsigned long long)b);
        }
    }
    return 0;
}
`
