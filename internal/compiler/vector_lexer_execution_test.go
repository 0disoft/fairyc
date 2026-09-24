package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// This opt-in comparison heats the two storage loops in the checked-in scanner.
// The token reporter runs once outside QPC; its runtime is a separate gate.
func TestPrivateVectorSDKCandidateRealScannerExecution(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_LEXER_EXECUTION") != "1" {
		t.Skip("opt-in real scanner execution comparison")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("Windows AMD64 MSVC required")
	}
	revision := os.Getenv("FAIRYC_VECTOR_LEXER_SOURCE")
	if revision == "" {
		t.Fatal("source revision required")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	sourceHashes := map[string]string{}
	read := func(name string) string {
		data, readErr := os.ReadFile(filepath.Join(example, filepath.FromSlash(name)))
		if readErr != nil {
			t.Fatal(readErr)
		}
		sourceHashes[name] = fmt.Sprintf("%x", sha256.Sum256(data))
		return string(data)
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("scanner variant order changed")
	}
	read("src/scanner.fy")
	harness, harnessErr := os.ReadFile("vector_lexer_execution_test.go")
	if harnessErr != nil {
		t.Fatal(harnessErr)
	}
	sourceHashes["internal/compiler/vector_lexer_execution_test.go"] = fmt.Sprintf("%x", sha256.Sum256(harness))
	sourceHashes["benchmark-main.fy"] = fmt.Sprintf("%x", sha256.Sum256([]byte(realScannerExecutionMain)))
	const sample = "let n=12345; /*open"
	lexed := lexer.Lex(source.NewFile("input.fy", []byte(sample)), source.NewBudget(source.DefaultLimits(), len(sample)))
	perRound := len(sample) + len(lexed.Tokens) + len(lexed.Diagnostics)
	for _, token := range lexed.Tokens {
		perRound += len(token.Cooked)
	}
	for _, diagnostic := range lexed.Diagnostics {
		perRound += len(diagnostic.Token)
	}
	if len(lexed.Tokens) == 0 || len(lexed.Diagnostics) == 0 || perRound <= len(sample) {
		t.Fatal("scanner sample lost required token, cooked or diagnostic coverage")
	}
	rounds := 10000
	if os.Getenv("FAIRYC_VECTOR_LEXER_LONG") == "1" {
		rounds = 250000
	}
	if perRound*rounds >= 1<<32 {
		t.Fatal("checksum no longer fits u32")
	}
	files := []string{"fairy.toml", "src/numbers.fy", "src/diagnostics.fy", "src/utf8.fy", "src/strings.fy", "src/budget.fy"}
	type image struct {
		name     string
		artifact cdriver.WindowsHostArtifact
		hash     string
	}
	images := make([]image, 2)
	builds := make([]map[string]any, 0, 2)
	for lane, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		root := t.TempDir()
		for _, name := range files {
			writeVectorCacheFile(t, root, name, read(name))
		}
		writeVectorCacheFile(t, root, "src/scanner.fy", variant.scanner)
		writeVectorCacheFile(t, root, "src/main.fy", realScannerExecutionMain)
		workspace, loadErr := workspaceloader.LoadVectorCandidate(root)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		input, inputErr := workspace.SemanticInput()
		coordinates, coordinateErr := workspace.PackageCoordinates()
		owners, ownerErr := workspace.VectorCandidateStorageOwners()
		if inputErr != nil || coordinateErr != nil || ownerErr != nil || len(owners) != 1 {
			t.Fatalf("scanner package authority: %v %v %v owners=%v", inputErr, coordinateErr, ownerErr, owners)
		}
		if variant.candidate {
			stable, stableErr := compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
			if stableErr != nil || len(stable.Analysis.Semantic.Unsupported) == 0 {
				t.Fatalf("stable candidate gate changed: %v %+v", stableErr, stable.Analysis.Semantic)
			}
		}
		checkStart := time.Now()
		var analysis compiler.GenericAnalysis
		if variant.candidate {
			analysis, err = compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
		} else {
			analysis, err = compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
		}
		if err != nil || analysis.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow || len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
			t.Fatalf("scanner Flow: %v diagnostics=%+v unsupported=%+v", err, analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
		}
		checked, lowerErr := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
		if lowerErr != nil || !checked.Verified {
			t.Fatalf("scanner MIR: %v diagnostics=%+v", lowerErr, checked.Semantic.Diagnostics)
		}
		checkNS := time.Since(checkStart).Nanoseconds()
		rootID, rootErr := workspace.RootPackageID()
		if rootErr != nil {
			t.Fatal(rootErr)
		}
		unit, emitErr := cbackend.EmitStorageTransfers(checked.MIR, hir.QualifiedModulePath(rootID, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
		if emitErr != nil {
			t.Fatal(emitErr)
		}
		scalar, scalarErr := cruntime.ScalarSource(192)
		storage, storageErr := cstdlib.WindowsMSVCVectorStorageSource()
		if scalarErr != nil || storageErr != nil {
			t.Fatalf("scanner support: %v %v", scalarErr, storageErr)
		}
		symbols := []string{"fy_std_system_heap_allocate_v0"}
		for _, symbol := range unit.ForeignSymbols {
			if strings.HasPrefix(symbol, "fy_std_") {
				symbols = append(symbols, symbol)
			} else if !strings.HasPrefix(symbol, "fy_private_vector_") && symbol != "fc_budget_seed" && symbol != "fc_budget_rounds" && symbol != "fc_budget_begin" && symbol != "fc_budget_finish" && symbol != "fc_budget_flush" && symbol != "fairyc_fixture_vector_clean" {
				t.Fatalf("unexpected foreign symbol %s", symbol)
			}
		}
		sdk, sdkErr := cstdlib.WindowsMSVCSourceForSymbols(symbols)
		heap, heapErr := cstdlib.WindowsMSVCSourceForSymbols(symbols[:1])
		if sdkErr != nil || heapErr != nil || !strings.HasPrefix(storage, heap) {
			t.Fatalf("scanner support closure: %v %v", sdkErr, heapErr)
		}
		budgetC, budgetErr := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "support.c"))
		if budgetErr != nil {
			t.Fatal(budgetErr)
		}
		sourceHashes["tools/benchmarks/c-budget/support.c"] = fmt.Sprintf("%x", sha256.Sum256(budgetC))
		buildStart := time.Now()
		artifact, buildErr := cdriver.BuildWindowsHost(toolchain, cdriver.OptimizedProfile, unit.Source,
			scalar+vectorStorageAllocator+sdk+strings.TrimPrefix(storage, heap)+vectorForNativeProbe+"\n"+string(budgetC)+"\nvoid fc_budget_flush(void) { fflush(stdout); }\n")
		if buildErr != nil {
			t.Fatal(buildErr)
		}
		buildNS := time.Since(buildStart).Nanoseconds()
		hash := fmt.Sprintf("%x", sha256.Sum256(artifact.Bytes))
		images[lane] = image{name: variant.name, artifact: artifact, hash: hash}
		builds = append(builds, map[string]any{"lane": variant.name, "scanner_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(variant.scanner))), "exe_sha256": hash, "exe_bytes": len(artifact.Bytes), "unit_bytes": len(unit.Source), "analysis_lower_ns": checkNS, "native_build_ns": buildNS, "specializations": len(analysis.Plan.Specializations)})
	}
	// Each process performs the same bounded scan/cleanup work; QPC excludes
	// process launch, input parsing and the one reporter call.
	reporterPrefix := ""
	reporterSeen := false
	execute := func(lane, repetition, position int, warmup bool) map[string]any {
		name := images[lane].name
		outcome, runErr := cdriver.ExecuteWindowsMSVCTargetWithInput(images[lane].artifact, strings.NewReader(fmt.Sprintf("12345 %d\n", rounds)))
		if runErr != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
			t.Fatalf("%s execution: %+v %v", name, outcome, runErr)
		}
		fields := strings.Fields(outcome.Stdout)
		if len(fields) < 4 {
			t.Fatalf("%s timing output: %q", name, outcome.Stdout)
		}
		prefix := strings.Join(fields[:len(fields)-4], " ")
		if !reporterSeen {
			reporterPrefix, reporterSeen = prefix, true
		} else if prefix != reporterPrefix {
			t.Fatalf("%s reporter output changed", name)
		}
		completed, completedErr := strconv.ParseUint(fields[len(fields)-4], 10, 32)
		ticks, ticksErr := strconv.ParseInt(fields[len(fields)-3], 10, 64)
		frequency, frequencyErr := strconv.ParseInt(fields[len(fields)-2], 10, 64)
		value, valueErr := strconv.ParseUint(fields[len(fields)-1], 10, 64)
		if completedErr != nil || valueErr != nil || ticksErr != nil || frequencyErr != nil || completed != uint64(rounds) || value != uint64(perRound*rounds) || ticks <= 0 || frequency <= 0 {
			t.Fatalf("%s result or QPC mismatch: %v %v %v %v got=%d want=%d", name, completedErr, valueErr, ticksErr, frequencyErr, value, perRound*rounds)
		}
		return map[string]any{"lane": name, "repetition": repetition, "position": position, "warmup": warmup, "kernel_ns": ticks * 1000000000 / frequency, "result": value, "exe_sha256": images[lane].hash}
	}
	pairs := 12
	if os.Getenv("FAIRYC_VECTOR_LEXER_EXECUTION_SMOKE") == "1" {
		pairs = 1
	}
	rows := make([]map[string]any, 0, (pairs+1)*2)
	for lane := range images {
		rows = append(rows, execute(lane, -1, lane, true))
	}
	for repetition := 0; repetition < pairs; repetition++ {
		for position, lane := range [2]int{repetition % 2, (repetition + 1) % 2} {
			rows = append(rows, execute(lane, repetition, position, false))
		}
	}
	for _, lane := range images {
		if fmt.Sprintf("%x", sha256.Sum256(lane.artifact.Bytes)) != lane.hash {
			t.Fatal("scanner image changed during measurement")
		}
	}
	report := map[string]any{"schema": "fairyc.real-scanner-vector-execution/v1", "compiler_source": revision, "go": runtime.Version(), "msvc": toolchain.Version, "gomaxprocs": runtime.GOMAXPROCS(0), "source_sha256": sourceHashes, "builds": builds, "samples": rows, "warmup_pairs": 1, "measured_pairs": pairs, "rounds": rounds, "per_round_result": perRound, "scope": "Actual checked-in scanner source; two storage loops execute in every timed round. The token reporter executes once before QPC. This is indexed-versus-private-candidate Fairy, not an independent C comparison. Analysis/lowering and native build intervals are one-shot diagnostics, not cold CLI latency. No outlier removal or retry; host load and frequency are uncontrolled."}
	encoded, marshalErr := json.Marshal(report)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	t.Log("VECTOR_LEXER_EXECUTION_REPORT " + string(encoded))
}

const realScannerExecutionMain = `module main;
import scanner as scanner;
import std::bytes as bytes;
import std::io as io;
@link_name("fc_budget_seed") unsafe extern C fn u32 benchmark_seed();
@link_name("fc_budget_rounds") unsafe extern C fn u32 benchmark_rounds();
@link_name("fc_budget_begin") unsafe extern C fn void benchmark_begin();
@link_name("fc_budget_finish") unsafe extern C fn void benchmark_finish(u32 result);
@link_name("fc_budget_flush") unsafe extern C fn void benchmark_flush();
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
@effects(console) fn i32 main() {
    assert(scanner::report_work(b"let n=12345; /*open"[..], false, 32768, 1048576, 256, 16777216) == 0);
    let u32 seed = unsafe { benchmark_seed() };
    let u32 rounds = unsafe { benchmark_rounds() };
    if (seed != 12345 || rounds > 250000) { return 2; }
    unsafe { benchmark_begin(); }
    var usize total = 0;
    var u32 iteration = 0;
    while (iteration < rounds) {
        let bytes::Buffer owned = switch (bytes::copy(b"let n=12345; /*open")) {
            case .ok(value) : { value }
            case .err(_) : { assert(false); bytes::empty() }
        };
        total += switch (scanner::collect(owned, false, 32768, 1048576, 256, 16777216)) {
            case .ok(stream) : {
                let scanner::Storage usage = scanner::storage(ref stream);
                usage.source_bytes + usage.cooked_bytes + usage.diagnostic_bytes + usage.tokens + usage.diagnostics
            }
            case .err(_) : { assert(false); 0 }
        };
        iteration += 1;
    }
    unsafe { benchmark_finish(iteration); }
    unsafe { benchmark_flush(); }
    assert(switch (io::write_usize(total)) { case .ok(_) : { true } case .err(_) : { false } });
    assert(unsafe { clean() } == 1);
    return 0;
}`
