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
	"unsafe"

	"github.com/0disoft/fairyc/internal/cbackend"
)

var scannerQueryProcessCycleTime = syscall.NewLazyDLL("kernel32.dll").NewProc("QueryProcessCycleTime")
var scannerQueryThreadCycleTime = syscall.NewLazyDLL("kernel32.dll").NewProc("QueryThreadCycleTime")
var scannerGetCurrentThread = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentThread")

func scannerProcessCycles() (uint64, error) {
	handle, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0, err
	}
	var cycles uint64
	ok, _, callErr := scannerQueryProcessCycleTime.Call(uintptr(handle), uintptr(unsafe.Pointer(&cycles)))
	if ok == 0 {
		return 0, fmt.Errorf("QueryProcessCycleTime: %v", callErr)
	}
	return cycles, nil
}

func scannerThreadCycles() (uint64, error) {
	handle, _, _ := scannerGetCurrentThread.Call()
	if handle == 0 {
		return 0, fmt.Errorf("GetCurrentThread returned a null handle")
	}
	var cycles uint64
	ok, _, callErr := scannerQueryThreadCycleTime.Call(handle, uintptr(unsafe.Pointer(&cycles)))
	if ok == 0 {
		return 0, fmt.Errorf("QueryThreadCycleTime: %v", callErr)
	}
	return cycles, nil
}

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
	ProcessCycles   uint64 `json:"process_cycles,omitempty"`
	ThreadCycles    uint64 `json:"thread_cycles,omitempty"`
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
	cycleNull := os.Getenv("FAIRYC_VECTOR_LEXER_PROCESS_CYCLES_NULL") == "1"
	cycleCandidate := os.Getenv("FAIRYC_VECTOR_LEXER_PROCESS_CYCLES_COST") == "1"
	threadNull := os.Getenv("FAIRYC_VECTOR_LEXER_THREAD_CYCLES_NULL") == "1"
	threadCandidate := os.Getenv("FAIRYC_VECTOR_LEXER_THREAD_CYCLES_COST") == "1"
	selected := 0
	for _, mode := range []bool{ordinaryNull, ordinaryCandidate, gcNull, gcCandidate, cycleNull, cycleCandidate, threadNull, threadCandidate} {
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
	null := ordinaryNull || gcNull || cycleNull || threadNull
	cycleControlled := cycleNull || cycleCandidate
	threadControlled := threadNull || threadCandidate
	gcControlled := gcNull || gcCandidate || cycleControlled || threadControlled
	if runtime.GOMAXPROCS(0) != 1 {
		t.Fatal("scanner process-CPU measurement requires GOMAXPROCS=1")
	}
	var affinityMask string
	if threadControlled {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		var restore func() error
		var err error
		affinityMask, restore, err = pinScannerBuildLowestAllowedCPU()
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := restore(); err != nil {
				t.Error(err)
			}
		}()
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
	pairs := 25
	if threadControlled {
		buildsPerLanePerPair, pairs = 8, 13
	}
	rows := make([]scannerProcessCPURow, 0, pairs*buildsPerLanePerPair*2)
	for pair := 0; pair < pairs; pair++ {
		laneOrder := scannerBuildLaneOrder(pair, true)
		if threadControlled {
			base := laneOrder
			laneOrder = make([]int, 0, 4*len(base))
			for repeat := 0; repeat < 4; repeat++ {
				laneOrder = append(laneOrder, base...)
			}
		} else if gcControlled {
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
			var startCycles uint64
			if cycleControlled {
				startCycles, err = scannerProcessCycles()
				if err != nil {
					t.Fatal(err)
				}
			}
			var startThreadCycles uint64
			if threadControlled {
				startThreadCycles, err = scannerThreadCycles()
				if err != nil {
					t.Fatal(err)
				}
			}
			startWall := time.Now()
			var check scannerBudgetCheckPhases
			program, entry, specializations := scannerBudgetFairyProgramWithMainTiming(t, variant, realScannerExecutionMain, &check)
			emitStart := time.Now()
			unit, err := cbackend.EmitStorageTransfers(program, entry, cbackend.Stage0WindowsMSVCProfile())
			emitNS := time.Since(emitStart).Nanoseconds()
			wallNS := time.Since(startWall).Nanoseconds()
			endCPU, cpuErr := scannerProcessCPUNS()
			var endCycles uint64
			var cycleErr error
			if cycleControlled {
				endCycles, cycleErr = scannerProcessCycles()
			}
			var endThreadCycles uint64
			var threadErr error
			if threadControlled {
				endThreadCycles, threadErr = scannerThreadCycles()
			}
			if gcControlled {
				var gcAfter runtime.MemStats
				runtime.ReadMemStats(&gcAfter)
				if gcAfter.NumGC != gcBefore.NumGC {
					t.Fatalf("unexpected GC during timed scanner build: %d to %d", gcBefore.NumGC, gcAfter.NumGC)
				}
			}
			if err != nil || cpuErr != nil || cycleErr != nil || threadErr != nil {
				t.Fatalf("%s C emission or process timer: %v %v %v %v", variant.name, err, cpuErr, cycleErr, threadErr)
			}
			cpuNS := endCPU - startCPU
			if cycleControlled && endCycles <= startCycles {
				t.Fatalf("%s process cycles did not advance: %d to %d", variant.name, startCycles, endCycles)
			}
			if threadControlled && endThreadCycles <= startThreadCycles {
				t.Fatalf("%s thread cycles did not advance: %d to %d", variant.name, startThreadCycles, endThreadCycles)
			}
			checkNS := check.LoadNS + check.MetadataNS + check.AnalyzeNS + check.LowerNS
			if cpuNS <= 0 || wallNS <= 0 || checkNS <= 0 || emitNS <= 0 ||
				specializations != 21 || len(unit.Functions) != 80 || len(unit.Source) == 0 {
				t.Fatalf("%s incomplete scanner CPU build: cpu=%d wall=%d check=%d emit=%d specializations=%d functions=%d C=%d",
					variant.name, cpuNS, wallNS, checkNS, emitNS, specializations, len(unit.Functions), len(unit.Source))
			}
			rows = append(rows, scannerProcessCPURow{
				Pair: pair, Warmup: pair == 0, Order: order, Lane: variant.name,
				ProcessCPUNS: cpuNS, WallNS: wallNS, CheckNS: checkNS, EmitNS: emitNS,
				ProcessCycles:   endCycles - startCycles,
				ThreadCycles:    endThreadCycles - startThreadCycles,
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
	if cycleControlled {
		schema, prefix = "fairyc.vector-lexer-process-cycles-cost/v1", "VECTOR_LEXER_PROCESS_CYCLES_COST "
		if null {
			schema, prefix = "fairyc.vector-lexer-process-cycles-null/v1", "VECTOR_LEXER_PROCESS_CYCLES_NULL "
		}
	}
	if threadControlled {
		schema, prefix = "fairyc.vector-lexer-thread-cycles-cost/v1", "VECTOR_LEXER_THREAD_CYCLES_COST "
		if null {
			schema, prefix = "fairyc.vector-lexer-thread-cycles-null/v1", "VECTOR_LEXER_THREAD_CYCLES_NULL "
		}
	}
	report := map[string]any{
		"schema": schema, "go": runtime.Version(), "gomaxprocs": runtime.GOMAXPROCS(0),
		"target":       string(cbackend.Stage0WindowsMSVCProfile().Target),
		"warmup_pairs": 1, "measured_pairs": pairs - 1, "builds_per_lane_per_pair": buildsPerLanePerPair,
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
	if cycleControlled {
		report["scope"] = "The GC-controlled 200-build scanner schedule also brackets each build with QueryProcessCycleTime. Process cycles sum all process threads and include kernel work, but are not elapsed time; CPU frequency, thread migration and ambient load may vary. Process CPU, wall, check and emit durations remain diagnostic. A cycle null must pass before a separate candidate cycle screen. Neither result alone establishes normal-GC check latency, complete native build time or the C-relative runtime gate."
	}
	if threadControlled {
		report["pair_order"] = "A-B-B-A x4 / B-A-A-B x4 alternating"
		report["affinity_mask"] = affinityMask
		report["osthread_locked"] = true
		report["scope"] = "A locked Go OS thread pinned with the process to the lowest permitted logical processor brackets each fresh scanner preparation, load, analysis, verified-MIR lowering and C emission with QueryThreadCycleTime. Eight builds per lane per pair reduce per-build variation; GC runs outside each interval and automatic GC is disabled. Thread cycles are not elapsed time and omit work on other threads, off-CPU waits, native MSVC children and ordinary GC. Source/C hashes and counts remain checked. An identical-source null must pass before the candidate screen; neither result alone establishes normal check/build latency or the C-relative runtime gate."
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(prefix + string(encoded))
}
