package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/pecoff"
)

// A PDB can name release-image intervals only when the optimized debug build
// retains every validated function boundary at the same RVA.
func TestPrivateVectorSDKCandidateScannerDebugMapLayout(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	support, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "support.c"))
	if err != nil {
		t.Fatal(err)
	}
	extraSupport := "\n" + string(support) + "\nvoid fc_budget_flush(void) { fflush(stdout); }\n"
	cSource := scannerCBenchmarkSource(t)
	cRelease, err := cdriver.BuildWindowsMSVCProbeWithMap(toolchain, cdriver.OptimizedProfile, cSource)
	if err != nil {
		t.Fatal(err)
	}
	cDebug, err := cdriver.BuildWindowsMSVCProbeWithDebugMap(toolchain, cdriver.OptimizedProfile, cSource)
	if err != nil {
		t.Fatal(err)
	}
	checkScannerDebugPair(t, "independent C", cRelease, cDebug)
	variants := realScannerVectorVariants(t)
	for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		program, entry, _ := scannerBudgetFairyProgram(t, variant)
		release, releaseUnit := buildRealScannerVectorNativeWithStorageAndMap(t, program, entry,
			cdriver.OptimizedProfile, extraSupport, "", true)
		debug, debugUnit := buildRealScannerVectorNativeWithDebugMap(t, program, entry,
			cdriver.OptimizedProfile, extraSupport)
		if releaseUnit.Source != debugUnit.Source {
			t.Fatalf("%s generated C changed between optimized builds", variant.name)
		}
		checkScannerDebugPair(t, variant.name, release, debug)
	}
}

func checkScannerDebugPair(t *testing.T, name string, release, debug cdriver.WindowsHostArtifact) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		if len(debug.ProgramDatabase) == 0 || len(debug.LinkMap) == 0 {
			t.Fatal("debug image omitted PDB or map")
		}
		if _, err := pecoff.InspectProgramDatabaseAssociation(debug.Bytes, debug.ProgramDatabase); err != nil {
			t.Fatal(err)
		}
		execute := func(artifact cdriver.WindowsHostArtifact) string {
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, strings.NewReader("12345 1000\n"))
			if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("scanner debug-layout execution: %+v %v", outcome, err)
			}
			fields := strings.Fields(outcome.Stdout)
			if len(fields) < 5 || fields[len(fields)-4] != "1000" {
				t.Fatalf("scanner debug-layout timing report is incomplete: %q", outcome.Stdout)
			}
			return strings.Join(fields[:len(fields)-4], " ") + " " + fields[len(fields)-4] + " " + fields[len(fields)-1]
		}
		if left, right := execute(release), execute(debug); left != right {
			t.Fatal("debug build changed scanner output")
		}
		releaseUnwind, err := pecoff.InspectAMD64Unwind(release.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		debugUnwind, err := pecoff.InspectAMD64Unwind(debug.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if len(releaseUnwind.Functions) != len(debugUnwind.Functions) {
			t.Fatalf("unwind interval count changed: release=%d debug=%d", len(releaseUnwind.Functions), len(debugUnwind.Functions))
		}
		for i, left := range releaseUnwind.Functions {
			right := debugUnwind.Functions[i]
			if left.BeginRVA != right.BeginRVA || left.EndRVA != right.EndRVA {
				t.Fatalf("unwind interval %d moved: release=%#x-%#x debug=%#x-%#x", i,
					left.BeginRVA, left.EndRVA, right.BeginRVA, right.EndRVA)
			}
		}
		t.Logf("release EXE=%d debug EXE=%d PDB=%d matching unwind intervals=%d",
			len(release.Bytes), len(debug.Bytes), len(debug.ProgramDatabase), len(releaseUnwind.Functions))
	})
}
