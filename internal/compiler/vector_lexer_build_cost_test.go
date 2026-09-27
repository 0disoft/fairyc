package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

type scannerBuildCostRow struct {
	Pair            int                      `json:"pair"`
	Warmup          bool                     `json:"warmup"`
	Order           int                      `json:"order"`
	Lane            string                   `json:"lane"`
	CheckNS         int64                    `json:"check_ns"`
	Build           scannerNativeBuildPhases `json:"build"`
	Specializations int                      `json:"specializations"`
	CSourceBytes    int                      `json:"c_source_bytes"`
	EXEBytes        int                      `json:"exe_bytes"`
}

func scannerBuildLaneOrder(pair int, balanced bool) []int {
	if balanced {
		if pair%2 == 0 {
			return []int{0, 1, 1, 0}
		}
		return []int{1, 0, 0, 1}
	}
	if pair%2 == 0 {
		return []int{0, 1}
	}
	return []int{1, 0}
}

func TestScannerBalancedBuildLaneOrder(t *testing.T) {
	for pair := 0; pair < 25; pair++ {
		want := []int{0, 1, 1, 0}
		if pair%2 != 0 {
			want = []int{1, 0, 0, 1}
		}
		if got := scannerBuildLaneOrder(pair, true); !slices.Equal(got, want) {
			t.Fatalf("pair %d order = %v, want %v", pair, got, want)
		}
		legacy := []int{0, 1}
		if pair%2 != 0 {
			legacy = []int{1, 0}
		}
		if got := scannerBuildLaneOrder(pair, false); !slices.Equal(got, legacy) {
			t.Fatalf("legacy pair %d order = %v, want %v", pair, got, legacy)
		}
	}
}

// Compares fresh indexed and private vector-loop source through optimized
// Windows native image construction. Warmup and lane order are fixed in advance.
func TestPrivateVectorSDKCandidateScannerBuildCost(t *testing.T) {
	plainNull := os.Getenv("FAIRYC_VECTOR_LEXER_BUILD_NULL") == "1"
	affinityNull := os.Getenv("FAIRYC_VECTOR_LEXER_BUILD_AFFINITY_NULL") == "1"
	balancedNull := os.Getenv("FAIRYC_VECTOR_LEXER_BUILD_BALANCED_NULL") == "1"
	balancedCandidate := os.Getenv("FAIRYC_VECTOR_LEXER_BUILD_BALANCED_COST") == "1"
	robustNull := os.Getenv("FAIRYC_VECTOR_LEXER_BUILD_ROBUST_NULL") == "1"
	robustCandidate := os.Getenv("FAIRYC_VECTOR_LEXER_BUILD_ROBUST_COST") == "1"
	plainCandidate := os.Getenv("FAIRYC_VECTOR_LEXER_BUILD_COST") == "1"
	nullControl := plainNull || affinityNull || balancedNull || robustNull
	candidateComparison := plainCandidate || balancedCandidate || robustCandidate
	robust := robustNull || robustCandidate
	balanced := balancedNull || balancedCandidate || robust
	modes := 0
	for _, active := range []bool{plainNull, affinityNull, balancedNull, robustNull, plainCandidate, balancedCandidate, robustCandidate} {
		if active {
			modes++
		}
	}
	if modes > 1 {
		t.Fatal("choose exactly one scanner build-cost mode")
	}
	if !nullControl && !candidateComparison {
		t.Skip("opt-in private scanner build-cost or identical-source null measurement")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	affinityMask := ""
	if affinityNull {
		mask, restore, err := pinScannerBuildLowestAllowedCPU()
		if err != nil {
			t.Fatal(err)
		}
		affinityMask = mask
		t.Cleanup(func() {
			if err := restore(); err != nil {
				t.Error(err)
			}
		})
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	lanes := []realScannerVectorVariant{variants[0], variants[4]}
	if nullControl {
		lanes[1] = variants[0]
		lanes[1].name = "explicit index clone"
		if lanes[0].scanner != lanes[1].scanner || lanes[0].candidate != lanes[1].candidate {
			t.Fatal("null control changed scanner source or compiler lane")
		}
	}
	support, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "support.c"))
	if err != nil {
		t.Fatal(err)
	}
	extraSupport := "\n" + string(support) + "\nvoid fc_budget_flush(void) { fflush(stdout); }\n"
	buildsPerPair := 2
	if balanced {
		buildsPerPair = 4
	}
	pairCount := 6
	if robust {
		pairCount = 25
	}
	rows := make([]scannerBuildCostRow, 0, pairCount*buildsPerPair)
	var toolchainVersion string
	for pair := 0; pair < pairCount; pair++ {
		order := scannerBuildLaneOrder(pair, balanced)
		for position, index := range order {
			variant := lanes[index]
			var check scannerBudgetCheckPhases
			program, entry, specializations := scannerBudgetFairyProgramWithMainTiming(t, variant, realScannerExecutionMain, &check)
			checkNS := check.LoadNS + check.MetadataNS + check.AnalyzeNS + check.LowerNS
			var build scannerNativeBuildPhases
			artifact, unit := buildRealScannerVectorNativeWithStorageAndSymbolsTimed(t, program, entry, cdriver.OptimizedProfile, extraSupport, "", false, false, false, &build)
			if checkNS <= 0 || build.EmitNS <= 0 || build.NativeNS <= 0 ||
				specializations != 21 || len(unit.Source) == 0 || len(artifact.Bytes) == 0 {
				t.Fatalf("%s incomplete scanner build: check=%d build=%+v specializations=%d", variant.name, checkNS, build, specializations)
			}
			if toolchainVersion == "" {
				toolchainVersion = artifact.Toolchain.Version
			} else if toolchainVersion != artifact.Toolchain.Version {
				t.Fatal("MSVC toolchain changed during paired measurement")
			}
			rows = append(rows, scannerBuildCostRow{
				Pair: pair, Warmup: pair == 0, Order: position, Lane: variant.name,
				CheckNS: checkNS, Build: build, Specializations: specializations,
				CSourceBytes: len(unit.Source), EXEBytes: len(artifact.Bytes),
			})
		}
	}
	if nullControl {
		for pair := 0; pair < pairCount; pair++ {
			first := rows[pair*buildsPerPair]
			for _, next := range rows[pair*buildsPerPair+1 : (pair+1)*buildsPerPair] {
				if first.CSourceBytes != next.CSourceBytes || first.EXEBytes != next.EXEBytes ||
					first.Specializations != next.Specializations {
					t.Fatalf("identical-source pair %d changed artifacts: %+v %+v", pair, first, next)
				}
			}
		}
	}
	schema, prefix := "fairyc.vector-lexer-build-cost/v1", "VECTOR_LEXER_BUILD_COST "
	scope := "One Go test process builds fresh temporary packages in alternating order. check_ns covers load, metadata, analysis and verified MIR lowering; prepare and toolchain resolution are excluded. emit_ns is generated C emission, support_ns assembles runtime C source and may read zero below clock resolution, and native_ns includes MSVC compile/link plus artifact validation and reads. No persistent cache is requested. Ambient filesystem cache, process load and Go GC remain uncontrolled. This compares Fairy source forms, not independent C build cost or cold stable CLI startup."
	if nullControl {
		schema, prefix = "fairyc.vector-lexer-build-null/v1", "VECTOR_LEXER_BUILD_NULL "
		scope = "Identical explicit-index Fairy source is compiled in both lanes with the same fresh-package and optimized MSVC phases as the candidate build comparison. Alternating lane labels and one warmup pair precede five measured pairs. Pairwise C-source length, specialization count and EXE size must match. Ambient filesystem cache, process load and Go GC remain uncontrolled. This calibrates host and harness variation; it cannot establish candidate compiler cost."
	}
	report := map[string]any{
		"schema":         schema,
		"go":             runtime.Version(),
		"gomaxprocs":     runtime.GOMAXPROCS(0),
		"msvc":           toolchainVersion,
		"profile":        cdriver.OptimizedProfile.Name,
		"warmup_pairs":   1,
		"measured_pairs": pairCount - 1,
		"indexed_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(lanes[0].scanner))),
		"vector_sha256":  fmt.Sprintf("%x", sha256.Sum256([]byte(lanes[1].scanner))),
		"main_sha256":    fmt.Sprintf("%x", sha256.Sum256([]byte(realScannerExecutionMain))),
		"support_sha256": fmt.Sprintf("%x", sha256.Sum256(support)),
		"rows":           rows,
		"scope":          scope,
	}
	if nullControl {
		report["clone_sha256"] = report["vector_sha256"]
		delete(report, "vector_sha256")
	}
	if affinityNull {
		report["schema"] = "fairyc.vector-lexer-build-affinity-null/v1"
		report["affinity_mode"] = "lowest-allowed-logical-processor"
		report["affinity_mask"] = affinityMask
		report["scope"] = scope + " The Go test process is pinned to one allowed logical processor before source preparation; Windows child processes inherit its affinity. The original mask is restored after the test. Priority class is unchanged."
		prefix = "VECTOR_LEXER_BUILD_AFFINITY_NULL "
	}
	if balanced {
		report["builds_per_lane_per_pair"] = 2
		report["pair_order"] = "A-B-B-A / B-A-A-B alternating"
		if balancedNull {
			report["schema"] = "fairyc.vector-lexer-build-balanced-null/v1"
			report["scope"] = "Identical explicit-index scanner sources occupy A and B in one warmup and five measured four-build pairs. ABBA/BAAB order alternates, and the two complete builds per lane are summed before forming each pair ratio. All phase times and observations are retained; this calibrates build noise, not candidate cost. No affinity or priority override."
			prefix = "VECTOR_LEXER_BUILD_BALANCED_NULL "
		} else {
			report["schema"] = "fairyc.vector-lexer-build-balanced-cost/v1"
			report["scope"] = "Explicit-index and private vector-loop scanner sources occupy A and B in one warmup and five measured four-build pairs. ABBA/BAAB order alternates, and the two complete builds per lane are summed before forming each pair ratio. All phase times and observations are retained. This compares steady-state Fairy build cost, not independent-C cost or cold CLI startup. No affinity or priority override."
			prefix = "VECTOR_LEXER_BUILD_BALANCED_COST "
		}
	}
	if robust {
		report["schema"] = "fairyc.vector-lexer-build-robust-cost/v1"
		report["scope"] = "Explicit-index and private vector-loop scanner sources occupy A and B in one warmup and 24 measured ABBA/BAAB pairs. All 100 fresh builds retain check, emit, support, native and artifact observations. Sort the 24 measured pair ratios and use the predeclared 18th ratio at most 1.02 as the candidate screen; report the full span without excluding any pair. This is steady-state Fairy build cost, not independent-C cost or cold CLI startup. No affinity or priority override."
		prefix = "VECTOR_LEXER_BUILD_ROBUST_COST "
		if robustNull {
			report["schema"] = "fairyc.vector-lexer-build-robust-null/v1"
			report["scope"] = "Identical explicit-index scanner sources occupy A and B in one warmup and 24 measured ABBA/BAAB pairs. All 100 fresh builds retain check, emit, support, native and artifact observations. Sort the 24 measured pair ratios and require both the 7th and 18th ratios inside [0.98, 1.02] before a separately recorded candidate comparison. Report the full span without excluding any pair. No affinity or priority override."
			prefix = "VECTOR_LEXER_BUILD_ROBUST_NULL "
		}
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(prefix + string(encoded))
}
