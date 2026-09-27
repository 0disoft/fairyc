package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

// Compares fresh indexed and private vector-loop source through optimized
// Windows native image construction. Warmup and lane order are fixed in advance.
func TestPrivateVectorSDKCandidateScannerBuildCost(t *testing.T) {
	nullControl := os.Getenv("FAIRYC_VECTOR_LEXER_BUILD_NULL") == "1"
	candidateComparison := os.Getenv("FAIRYC_VECTOR_LEXER_BUILD_COST") == "1"
	if nullControl && candidateComparison {
		t.Fatal("choose exactly one scanner build-cost mode")
	}
	if !nullControl && !candidateComparison {
		t.Skip("opt-in private scanner build-cost or identical-source null measurement")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
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
	rows := make([]scannerBuildCostRow, 0, 12)
	var toolchainVersion string
	for pair := 0; pair < 6; pair++ {
		order := []int{0, 1}
		if pair%2 != 0 {
			order[0], order[1] = order[1], order[0]
		}
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
		for pair := 0; pair < 6; pair++ {
			first, second := rows[pair*2], rows[pair*2+1]
			if first.CSourceBytes != second.CSourceBytes || first.EXEBytes != second.EXEBytes ||
				first.Specializations != second.Specializations {
				t.Fatalf("identical-source pair %d changed artifacts: %+v %+v", pair, first, second)
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
		"measured_pairs": 5,
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
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(prefix + string(encoded))
}
