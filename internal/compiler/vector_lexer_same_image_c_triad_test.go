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

// This harness is for a host whose immediately preceding C-C-C null control
// passes. A passing result here still cannot decide standalone executable size.
func TestPrivateVectorSDKCandidateSameImageCScannerTriads(t *testing.T) {
	if os.Getenv("FAIRYC_SCANNER_SAME_IMAGE_C_TRIADS") != "1" {
		t.Skip("opt-in same-image C/Fairy/C scanner triads")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("Windows AMD64 MSVC required")
	}
	smokeMode := os.Getenv("FAIRYC_SCANNER_SAME_IMAGE_C_TRIADS_SMOKE")
	if smokeMode != "" && smokeMode != "1" {
		t.Fatalf("unknown same-image triad smoke mode %q", smokeMode)
	}
	smoke := smokeMode == "1"
	measuredTriads, rounds := 24, 250000
	if smoke {
		measuredTriads, rounds = 1, 1000
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
	mainSource := sameImageCScannerTriadMain
	for _, replacement := range [][2]string{
		{"@@TRIADS@@", strconv.Itoa(measuredTriads + 1)},
		{"@@ROUNDS@@", strconv.Itoa(rounds)},
	} {
		if !strings.Contains(mainSource, replacement[0]) {
			t.Fatalf("missing triad source placeholder %s", replacement[0])
		}
		mainSource = strings.ReplaceAll(mainSource, replacement[0], replacement[1])
	}
	if strings.Contains(mainSource, "@@") {
		t.Fatal("unsubstituted triad source placeholder")
	}
	support := sameImageCScannerSupport(t) + sameImageCRoundsSupport +
		dualScannerTimingSupport + sameImageCycleAffinitySupport
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("scanner variants changed")
	}
	images := make([]map[string]any, 0, 2)
	toolchainVersion := ""
	for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		program, entry, specializations := scannerBudgetFairyProgramWithMain(t, variant, mainSource)
		artifact, unit := buildRealScannerVectorNative(t, program, entry, cdriver.OptimizedProfile, support)
		if toolchainVersion == "" {
			toolchainVersion = artifact.Toolchain.Version
		} else if artifact.Toolchain.Version != toolchainVersion {
			t.Fatal("triad toolchain changed between images")
		}
		outcome, err := cdriver.ExecuteWindowsHost(artifact)
		if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
			t.Fatalf("%s C/Fairy/C triads: %+v %v", variant.name, outcome, err)
		}
		fields := strings.Fields(outcome.Stdout)
		if len(fields) != 2+(measuredTriads+1)*6 {
			t.Fatalf("%s triad fields=%d want=%d: %q", variant.name, len(fields), 2+(measuredTriads+1)*6, outcome.Stdout)
		}
		pinnedProcessor, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			t.Fatalf("%s invalid pinned processor %q: %v", variant.name, fields[0], err)
		}
		frequency, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil || frequency == 0 {
			t.Fatalf("%s invalid QPC frequency %q: %v", variant.name, fields[1], err)
		}
		rows := make([]map[string]any, 0, measuredTriads+1)
		ratios := make([]float64, 0, measuredTriads)
		for triad := 0; triad <= measuredTriads; triad++ {
			base := 2 + triad*6
			values := [6]uint64{}
			for i := range values {
				values[i], err = strconv.ParseUint(fields[base+i], 10, 64)
				if err != nil {
					t.Fatalf("%s triad %d field %d: %v", variant.name, triad, i, err)
				}
			}
			if values[0] == 0 || values[1] == 0 || values[2] == 0 ||
				values[3] != uint64(expected) || values[4] != uint64(expected) || values[5] != uint64(expected) {
				t.Fatalf("%s triad %d: ticks=%v expected result=%d", variant.name, triad, values, expected)
			}
			if triad > 0 {
				ratios = append(ratios, float64(values[1])/math.Sqrt(float64(values[0])*float64(values[2])))
			}
			rows = append(rows, map[string]any{
				"triad": triad - 1, "warmup": triad == 0,
				"c_before_qpc_ticks": values[0], "fairy_qpc_ticks": values[1],
				"c_after_qpc_ticks": values[2], "c_before_result": values[3],
				"fairy_result": values[4], "c_after_result": values[5],
			})
		}
		sort.Float64s(ratios)
		median := (ratios[(measuredTriads-1)/2] + ratios[measuredTriads/2]) / 2
		span := (ratios[measuredTriads-1] - ratios[0]) / median
		image := map[string]any{
			"lane": variant.name, "qpc_frequency": frequency,
			"affinity": "current-logical-processor", "pinned_logical_processor": pinnedProcessor,
			"median_fairy_over_c": median, "relative_full_span": span,
			"exe_sha256": fmt.Sprintf("%x", sha256.Sum256(artifact.Bytes)),
			"exe_bytes":  len(artifact.Bytes), "generated_c_bytes": len(unit.Source),
			"scanner_sha256":  fmt.Sprintf("%x", sha256.Sum256([]byte(variant.scanner))),
			"specializations": specializations, "samples": rows,
		}
		if !smoke {
			image["passes_10_percent_stability"] = span <= 0.10
		}
		images = append(images, image)
	}
	report := map[string]any{
		"schema": "fairyc.same-image-c-scanner-triads/v1",
		"go":     runtime.Version(), "msvc": toolchainVersion,
		"profile":         cdriver.OptimizedProfile.Name,
		"rounds_per_call": rounds, "warmup_triads": 1,
		"measured_triads_per_image": measuredTriads,
		"expected_result_per_call":  expected,
		"timing_gate_eligible":      !smoke,
		"null_control_required":     true,
		"main_sha256":               fmt.Sprintf("%x", sha256.Sum256([]byte(mainSource))),
		"support_sha256":            fmt.Sprintf("%x", sha256.Sum256([]byte(support))),
		"images":                    images,
		"scope":                     "Each combined image pins its calling thread to one logical processor, then runs C-Fairy-C triads with separately checked QPC calls and Go-oracle work checksums. The immediately preceding complete pinned C-C-C null control must pass before any ratio is interpreted. No retry or exclusion; combined EXE size is not the standalone budget.",
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("SAME_IMAGE_C_SCANNER_TRIADS " + string(encoded))
}

const sameImageCScannerTriadMain = `module main;
import scanner as scanner;
import std::bytes as bytes;
import std::io as io;
@link_name("fc_pair_ticks") unsafe extern C fn usize pair_ticks();
@link_name("fc_pair_frequency") unsafe extern C fn usize pair_frequency();
@link_name("fc_pair_pin_thread") unsafe extern C fn usize pair_pin_thread();
@link_name("fc_same_image_work_rounds") unsafe extern C fn usize c_work_rounds(usize rounds);
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
fn usize run_c(usize rounds) { return unsafe { c_work_rounds(rounds) }; }
fn usize run_fairy(usize rounds) {
    var usize total = 0;
    var usize iteration = 0;
    while (iteration < rounds) {
        let bytes::Buffer owned = switch (bytes::copy(b"let n=12345; /*open")) {
            case .ok(value): { value }
            case .err(_): { assert(false); bytes::empty() }
        };
        total += switch (scanner::collect(owned, false, 32768, 1048576, 256, 16777216)) {
            case .ok(stream): {
                let scanner::Storage usage = scanner::storage(ref stream);
                usage.source_bytes + usage.cooked_bytes + usage.diagnostic_bytes + usage.tokens + usage.diagnostics
            }
            case .err(_): { assert(false); 0 }
        };
        iteration += 1;
    }
    return total;
}
@effects(console) fn void number(usize value) {
    assert(switch (io::write_usize(value)) { case .ok(_): { true } case .err(_): { false } });
    assert(switch (io::write(" ")) { case .ok(_): { true } case .err(_): { false } });
}
@effects(console) fn i32 main() {
    let usize pinned = unsafe { pair_pin_thread() };
    number(pinned);
    let usize frequency = unsafe { pair_frequency() };
    assert(frequency > 0);
    number(frequency);
    var usize triad = 0;
    while (triad < @@TRIADS@@) {
        let usize before_start = unsafe { pair_ticks() };
        let usize before_result = run_c(@@ROUNDS@@);
        let usize before_end = unsafe { pair_ticks() };
        assert(before_end > before_start);
        let usize fairy_start = unsafe { pair_ticks() };
        let usize fairy_result = run_fairy(@@ROUNDS@@);
        let usize fairy_end = unsafe { pair_ticks() };
        assert(fairy_end > fairy_start);
        let usize after_start = unsafe { pair_ticks() };
        let usize after_result = run_c(@@ROUNDS@@);
        let usize after_end = unsafe { pair_ticks() };
        assert(after_end > after_start);
        number(before_end - before_start);
        number(fairy_end - fairy_start);
        number(after_end - after_start);
        number(before_result);
        number(fairy_result);
        number(after_result);
        triad += 1;
    }
    assert(unsafe { clean() } == 1);
    return 0;
}`
