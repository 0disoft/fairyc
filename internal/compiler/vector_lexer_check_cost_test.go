package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"
)

type scannerCheckCostRow struct {
	Pair            int                      `json:"pair"`
	Warmup          bool                     `json:"warmup"`
	Order           int                      `json:"order"`
	Lane            string                   `json:"lane"`
	Phases          scannerBudgetCheckPhases `json:"phases"`
	CheckNS         int64                    `json:"check_ns"`
	Specializations int                      `json:"specializations"`
}

// Measures source loading through verified MIR without native C emission.
// Every pair uses fresh temporary package files and alternates lane order.
func TestPrivateVectorSDKCandidateScannerCheckCost(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_LEXER_CHECK_COST") != "1" {
		t.Skip("opt-in private scanner check-cost measurement")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	lanes := []realScannerVectorVariant{variants[0], variants[4]}
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
		"schema":         "fairyc.vector-lexer-check-cost/v1",
		"go":             runtime.Version(),
		"gomaxprocs":     runtime.GOMAXPROCS(0),
		"warmup_pairs":   1,
		"measured_pairs": 8,
		"indexed_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(lanes[0].scanner))),
		"vector_sha256":  fmt.Sprintf("%x", sha256.Sum256([]byte(lanes[1].scanner))),
		"main_sha256":    fmt.Sprintf("%x", sha256.Sum256([]byte(realScannerExecutionMain))),
		"rows":           rows,
		"scope":          "In-process fresh temporary packages. check_ns is load plus metadata plus analysis plus verified MIR lowering. Source preparation, C emission, MSVC build/link, process startup and stable workspace commands are excluded. No persistent cache is requested; ambient filesystem cache and GC remain uncontrolled.",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("VECTOR_LEXER_CHECK_COST " + string(report))
}
