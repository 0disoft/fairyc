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
	if os.Getenv("FAIRYC_VECTOR_LEXER_BUILD_COST") != "1" {
		t.Skip("opt-in private scanner build-cost measurement")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	lanes := []realScannerVectorVariant{variants[0], variants[4]}
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
	report, err := json.Marshal(map[string]any{
		"schema":         "fairyc.vector-lexer-build-cost/v1",
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
		"scope":          "One Go test process builds fresh temporary packages in alternating order. check_ns covers load, metadata, analysis and verified MIR lowering; prepare and toolchain resolution are excluded. emit_ns is generated C emission, support_ns assembles runtime C source and may read zero below clock resolution, and native_ns includes MSVC compile/link plus artifact validation and reads. No persistent cache is requested. Ambient filesystem cache, process load and Go GC remain uncontrolled. This compares Fairy source forms, not independent C build cost or cold stable CLI startup.",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("VECTOR_LEXER_BUILD_COST " + string(report))
}
