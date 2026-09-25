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

// One process pairs independently authored C scanner work with each actual
// Fairy scanner form. The image is instrumented and cannot decide EXE size.
func TestPrivateVectorSDKCandidateSameImageCScannerTiming(t *testing.T) {
	if os.Getenv("FAIRYC_SCANNER_SAME_IMAGE_C_TIMING") != "1" {
		t.Skip("opt-in same-image C/Fairy scanner timing")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("Windows AMD64 MSVC required")
	}
	pairs, rounds := 12, 250000
	if os.Getenv("FAIRYC_SCANNER_SAME_IMAGE_C_TIMING_SMOKE") == "1" {
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
		t.Fatal("scanner sample lost cooked or diagnostic coverage")
	}
	mainSource := sameImageCPairedMainSource(t, pairs, rounds)
	support := sameImageCScannerSupport(t) + sameImageCRoundsSupport + dualScannerTimingSupport + dualScannerCPUTimeSupport
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
			t.Fatal("same-image scanner toolchain changed between variants")
		}
		outcome, err := cdriver.ExecuteWindowsHost(artifact)
		if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
			t.Fatalf("%s same-image timing: %+v %v", variant.name, outcome, err)
		}
		fields := strings.Fields(outcome.Stdout)
		if len(fields) != 1+(pairs+1)*6 {
			t.Fatalf("%s timing fields=%d want=%d: %q", variant.name, len(fields), 1+(pairs+1)*6, outcome.Stdout)
		}
		frequency, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil || frequency == 0 {
			t.Fatalf("%s invalid QPC frequency %q: %v", variant.name, fields[0], err)
		}
		rows := make([]map[string]any, 0, pairs+1)
		for pair := 0; pair <= pairs; pair++ {
			base := 1 + pair*6
			values := [6]uint64{}
			for i := range values {
				values[i], err = strconv.ParseUint(fields[base+i], 10, 64)
				if err != nil {
					t.Fatalf("%s pair %d field %d: %v", variant.name, pair, i, err)
				}
			}
			if values[0] == 0 || values[1] == 0 || values[4] != uint64(perRound*rounds) || values[5] != uint64(perRound*rounds) {
				t.Fatalf("%s pair %d: qpc=%d,%d sums=%d,%d want=%d", variant.name, pair,
					values[0], values[1], values[4], values[5], perRound*rounds)
			}
			rows = append(rows, map[string]any{
				"pair": pair - 1, "warmup": pair == 0,
				"first":       []string{"independent C", variant.name}[pair%2],
				"c_qpc_ticks": values[0], "fairy_qpc_ticks": values[1],
				"c_cpu_100ns": values[2], "fairy_cpu_100ns": values[3],
				"c_result": values[4], "fairy_result": values[5],
			})
		}
		images = append(images, map[string]any{
			"lane": variant.name, "qpc_frequency": frequency,
			"exe_sha256": fmt.Sprintf("%x", sha256.Sum256(artifact.Bytes)),
			"exe_bytes":  len(artifact.Bytes), "generated_c_bytes": len(unit.Source),
			"scanner_sha256":  fmt.Sprintf("%x", sha256.Sum256([]byte(variant.scanner))),
			"specializations": specializations, "samples": rows,
		})
	}
	report := map[string]any{
		"schema": "fairyc.same-image-c-scanner-pairs/v1", "go": runtime.Version(),
		"msvc":       toolchainVersion,
		"gomaxprocs": runtime.GOMAXPROCS(0), "profile": cdriver.OptimizedProfile.Name,
		"rounds": rounds, "warmup_pairs": 1, "measured_pairs": pairs,
		"per_round_result": perRound, "cpu_unit_ns": 100,
		"sample_sha256":  fmt.Sprintf("%x", sha256.Sum256([]byte(sample))),
		"main_sha256":    fmt.Sprintf("%x", sha256.Sum256([]byte(mainSource))),
		"support_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(support))),
		"images":         images,
		"scope":          "Independent C and one Fairy scanner form share each instrumented optimized image. One fixed warmup and alternating adjacent calls follow per image; QPC and current-thread GetThreadTimes samples retain every pair. Each call verifies the same Go-oracle work checksum, with no retry or exclusion. This is a timing probe only: the combined EXEs cannot decide standalone size, whole-application runtime or stable source admission.",
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("SAME_IMAGE_C_SCANNER_PAIRS " + string(encoded))
}

func sameImageCPairedMainSource(t *testing.T, pairs, rounds int) string {
	t.Helper()
	// The reused timer calls its first lane "indexed"; here that lane is C.
	src := dualScannerCPUMainSource(t)
	const imports = "import indexed as indexed;\nimport candidate as candidate;"
	if strings.Count(src, imports) != 1 {
		t.Fatal("same-image scanner imports changed")
	}
	src = strings.Replace(src, imports, "import scanner as scanner;", 1)
	const indexedStart = "fn usize run_indexed(usize rounds) {"
	const candidateStart = "fn usize run_candidate(usize rounds) {"
	start, end := strings.Index(src, indexedStart), strings.Index(src, candidateStart)
	if start < 0 || end <= start || strings.Count(src, indexedStart) != 1 || strings.Count(src, candidateStart) != 1 {
		t.Fatal("same-image scanner function anchors changed")
	}
	src = src[:start] + "fn usize run_indexed(usize rounds) { return unsafe { c_work_rounds(rounds) }; }\n" + src[end:]
	if strings.Count(src, "candidate::") != 3 {
		t.Fatal("same-image Fairy scanner calls changed")
	}
	src = strings.ReplaceAll(src, "candidate::", "scanner::")
	const frequency = `@link_name("fc_pair_frequency") unsafe extern C fn usize pair_frequency();`
	if strings.Count(src, frequency) != 1 {
		t.Fatal("same-image timing FFI changed")
	}
	src = strings.Replace(src, frequency, frequency+"\n"+`@link_name("fc_same_image_work_rounds") unsafe extern C fn usize c_work_rounds(usize rounds);`, 1)
	for _, replacement := range [][2]string{{"@@PAIRS@@", strconv.Itoa(pairs + 1)}, {"@@ROUNDS@@", strconv.Itoa(rounds)}} {
		if strings.Count(src, replacement[0]) == 0 {
			t.Fatalf("missing source placeholder %s", replacement[0])
		}
		src = strings.ReplaceAll(src, replacement[0], replacement[1])
	}
	return src
}

const sameImageCRoundsSupport = `
size_t fc_same_image_work_rounds(size_t rounds) {
    static const unsigned char sample[] = "let n=12345; /*open";
    if (rounds == 0 || rounds > 250000u) abort();
    uint64_t total = 0;
    for (size_t i = 0; i < rounds; i++) {
        uint64_t work = 0;
        int status = fc_workload_round(sample, sizeof(sample) - 1u, 0, &work);
        if (status != 0 || UINT64_MAX - total < work) abort();
        total += work;
    }
    return (size_t)total;
}
`
