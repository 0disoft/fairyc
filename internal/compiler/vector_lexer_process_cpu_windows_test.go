package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"syscall"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func scannerProcessCPUNS() (int64, error) {
	handle, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0, err
	}
	var created, exited, kernel, user syscall.Filetime
	if err := syscall.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		return 0, err
	}
	filetimeTicks := func(value syscall.Filetime) uint64 {
		return uint64(value.HighDateTime)<<32 | uint64(value.LowDateTime)
	}
	return int64((filetimeTicks(kernel) + filetimeTicks(user)) * 100), nil
}

type scannerProcessCPURow struct {
	Pair            int    `json:"pair"`
	Warmup          bool   `json:"warmup"`
	Order           int    `json:"order"`
	Lane            string `json:"lane"`
	ProcessCPUNS    int64  `json:"process_cpu_ns"`
	WallNS          int64  `json:"wall_ns"`
	CheckNS         int64  `json:"check_ns"`
	EmitNS          int64  `json:"emit_ns"`
	Specializations int    `json:"specializations"`
	FunctionCount   int    `json:"function_count"`
	CSourceBytes    int    `json:"c_source_bytes"`
	CSourceSHA256   string `json:"c_source_sha256"`
}

// The same-source control must be recorded and accepted before the candidate
// mode is used. CPU time covers the Go test process, not MSVC child processes.
func TestPrivateVectorSDKCandidateScannerProcessCPUCost(t *testing.T) {
	ordinaryNull := os.Getenv("FAIRYC_VECTOR_LEXER_PROCESS_CPU_NULL") == "1"
	ordinaryCandidate := os.Getenv("FAIRYC_VECTOR_LEXER_PROCESS_CPU_COST") == "1"
	gcNull := os.Getenv("FAIRYC_VECTOR_LEXER_PROCESS_CPU_GC_NULL") == "1"
	gcCandidate := os.Getenv("FAIRYC_VECTOR_LEXER_PROCESS_CPU_GC_COST") == "1"
	selected := 0
	for _, mode := range []bool{ordinaryNull, ordinaryCandidate, gcNull, gcCandidate} {
		if mode {
			selected++
		}
	}
	if selected != 1 {
		if selected != 0 {
			t.Fatal("choose exactly one scanner process-CPU mode")
		}
		t.Skip("opt-in scanner process-CPU cost measurement")
	}
	null := ordinaryNull || gcNull
	gcControlled := gcNull || gcCandidate
	if runtime.GOMAXPROCS(0) != 1 {
		t.Fatal("scanner process-CPU measurement requires GOMAXPROCS=1")
	}
	if gcControlled {
		previousGCPercent := debug.SetGCPercent(-1)
		defer debug.SetGCPercent(previousGCPercent)
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	lanes := []realScannerVectorVariant{variants[0], variants[4]}
	if null {
		lanes[1] = variants[0]
		lanes[1].name = "explicit index clone"
		if lanes[0].scanner != lanes[1].scanner || lanes[0].candidate != lanes[1].candidate {
			t.Fatal("null control changed source or compiler lane")
		}
	} else if lanes[0].scanner == lanes[1].scanner {
		t.Fatal("candidate scanner source matches indexed source")
	}
	buildsPerLanePerPair := 2
	if gcControlled {
		buildsPerLanePerPair = 4
	}
	rows := make([]scannerProcessCPURow, 0, 25*buildsPerLanePerPair*2)
	for pair := 0; pair < 25; pair++ {
		laneOrder := scannerBuildLaneOrder(pair, true)
		if gcControlled {
			laneOrder = append(laneOrder, laneOrder...)
		}
		for order, index := range laneOrder {
			variant := lanes[index]
			var gcBefore runtime.MemStats
			if gcControlled {
				// Keep explicit collection outside every timed CPU interval.
				runtime.GC()
				runtime.ReadMemStats(&gcBefore)
			}
			startCPU, err := scannerProcessCPUNS()
			if err != nil {
				t.Fatal(err)
			}
			startWall := time.Now()
			var check scannerBudgetCheckPhases
			program, entry, specializations := scannerBudgetFairyProgramWithMainTiming(t, variant, realScannerExecutionMain, &check)
			emitStart := time.Now()
			unit, err := cbackend.EmitStorageTransfers(program, entry, cbackend.Stage0WindowsMSVCProfile())
			emitNS := time.Since(emitStart).Nanoseconds()
			wallNS := time.Since(startWall).Nanoseconds()
			endCPU, cpuErr := scannerProcessCPUNS()
			if gcControlled {
				var gcAfter runtime.MemStats
				runtime.ReadMemStats(&gcAfter)
				if gcAfter.NumGC != gcBefore.NumGC {
					t.Fatalf("unexpected GC during timed scanner build: %d to %d", gcBefore.NumGC, gcAfter.NumGC)
				}
			}
			if err != nil || cpuErr != nil {
				t.Fatalf("%s C emission or CPU timer: %v %v", variant.name, err, cpuErr)
			}
			cpuNS := endCPU - startCPU
			checkNS := check.LoadNS + check.MetadataNS + check.AnalyzeNS + check.LowerNS
			if cpuNS <= 0 || wallNS <= 0 || checkNS <= 0 || emitNS <= 0 ||
				specializations != 21 || len(unit.Functions) != 80 || len(unit.Source) == 0 {
				t.Fatalf("%s incomplete scanner CPU build: cpu=%d wall=%d check=%d emit=%d specializations=%d functions=%d C=%d",
					variant.name, cpuNS, wallNS, checkNS, emitNS, specializations, len(unit.Functions), len(unit.Source))
			}
			rows = append(rows, scannerProcessCPURow{
				Pair: pair, Warmup: pair == 0, Order: order, Lane: variant.name,
				ProcessCPUNS: cpuNS, WallNS: wallNS, CheckNS: checkNS, EmitNS: emitNS,
				Specializations: specializations, FunctionCount: len(unit.Functions),
				CSourceBytes: len(unit.Source), CSourceSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(unit.Source))),
			})
		}
	}
	if null {
		first := rows[0]
		for _, row := range rows[1:] {
			if row.CSourceSHA256 != first.CSourceSHA256 || row.CSourceBytes != first.CSourceBytes ||
				row.Specializations != first.Specializations || row.FunctionCount != first.FunctionCount {
				t.Fatal("identical-source CPU control changed generated artifacts")
			}
		}
	}
	schema, prefix := "fairyc.vector-lexer-process-cpu-cost/v1", "VECTOR_LEXER_PROCESS_CPU_COST "
	if null {
		schema, prefix = "fairyc.vector-lexer-process-cpu-null/v1", "VECTOR_LEXER_PROCESS_CPU_NULL "
	}
	if gcControlled {
		schema, prefix = "fairyc.vector-lexer-process-cpu-gc-cost/v1", "VECTOR_LEXER_PROCESS_CPU_GC_COST "
		if null {
			schema, prefix = "fairyc.vector-lexer-process-cpu-gc-null/v1", "VECTOR_LEXER_PROCESS_CPU_GC_NULL "
		}
	}
	report := map[string]any{
		"schema": schema, "go": runtime.Version(), "gomaxprocs": runtime.GOMAXPROCS(0),
		"target":       string(cbackend.Stage0WindowsMSVCProfile().Target),
		"warmup_pairs": 1, "measured_pairs": 24, "builds_per_lane_per_pair": buildsPerLanePerPair,
		"pair_order":     "A-B-B-A / B-A-A-B alternating",
		"indexed_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(lanes[0].scanner))),
		"second_sha256":  fmt.Sprintf("%x", sha256.Sum256([]byte(lanes[1].scanner))),
		"main_sha256":    fmt.Sprintf("%x", sha256.Sum256([]byte(realScannerExecutionMain))),
		"rows":           rows,
		"scope":          "Fresh real-scanner package preparation, load, analysis, verified-MIR lowering and C emission in one Go process. GetProcessTimes user+kernel CPU counts all process threads and excludes off-CPU waits, native compiler/linker children and persistent-cache hits. Wall and phase times are diagnostics. Go GC and host CPU accounting remain uncontrolled; no total-build or C-relative verdict follows from this CPU-only report.",
	}
	if gcControlled {
		report["pair_order"] = "A-B-B-A twice / B-A-A-B twice alternating"
		report["forced_gc_before_each_build"] = true
		report["automatic_gc_disabled_during_build"] = true
		report["scope"] = "Fresh real-scanner preparation, load, analysis, verified-MIR lowering and C emission in one Go process. GetProcessTimes user+kernel CPU covers all process threads. Explicit Go GC runs before every timed build and automatic GC is disabled throughout this test; GC outside the intervals and native MSVC work are excluded. Four builds per lane per pair reduce but do not eliminate the observed 15.625 ms clock quantization. This is a GC-excluded compiler-work screen, not complete build time, normal-GC check latency or an independent-C verdict."
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(prefix + string(encoded))
}
