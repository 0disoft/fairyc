package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"
)

// The same indexed source enters both compiler lanes. Only the private
// vector-for admission path changes, so this measures unused-source cost.
func TestPrivateVectorSDKCandidateUnusedCheckCost(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_LEXER_UNUSED_CHECK_COST") != "1" {
		t.Skip("opt-in private unused-source check-cost measurement")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" {
		t.Fatal("real scanner variants changed")
	}
	stable := variants[0]
	private := stable
	private.name = "candidate mode, indexed source"
	private.candidate = true
	lanes := []realScannerVectorVariant{stable, private}
	rows := make([]scannerCheckCostRow, 0, 18)
	for pair := 0; pair < 9; pair++ {
		order := []int{0, 1}
		if pair%2 != 0 {
			order[0], order[1] = order[1], order[0]
		}
		for position, index := range order {
			variant := lanes[index]
			var phases scannerBudgetCheckPhases
			_, _, specializations := scannerBudgetFairyProgramWithMainTiming(t, variant, realScannerExecutionMain, &phases)
			checkNS := phases.LoadNS + phases.MetadataNS + phases.AnalyzeNS + phases.LowerNS
			if checkNS <= 0 || phases.PrepareNS <= 0 || phases.AnalyzeNS <= 0 || phases.LowerNS <= 0 || specializations != 21 {
				t.Fatalf("%s incomplete scanner check: %+v specializations=%d", variant.name, phases, specializations)
			}
			rows = append(rows, scannerCheckCostRow{
				Pair: pair, Warmup: pair == 0, Order: position, Lane: variant.name,
				Phases: phases, CheckNS: checkNS, Specializations: specializations,
			})
		}
	}
	report, err := json.Marshal(map[string]any{
		"schema":         "fairyc.vector-lexer-unused-check-cost/v1",
		"go":             runtime.Version(),
		"gomaxprocs":     runtime.GOMAXPROCS(0),
		"warmup_pairs":   1,
		"measured_pairs": 8,
		"scanner_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(stable.scanner))),
		"main_sha256":    fmt.Sprintf("%x", sha256.Sum256([]byte(realScannerExecutionMain))),
		"rows":           rows,
		"scope":          "The exact same indexed scanner source and benchmark main enter stable SDK storage analysis or private vector-for candidate analysis. Each invocation uses fresh temporary package files. check_ns sums loading, metadata, semantic analysis and verified-MIR lowering. Preparation, C emission, native build, process startup and persistent-cache hits are excluded. Ambient filesystem cache, process load and Go GC remain uncontrolled.",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("VECTOR_LEXER_UNUSED_CHECK_COST " + string(report))
}
