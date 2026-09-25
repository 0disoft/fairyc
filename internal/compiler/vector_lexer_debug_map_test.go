//go:build windows

package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/pecoff"
)

// A PDB can name release-image intervals only when the optimized debug build
// retains every validated function boundary at the same RVA.
func TestPrivateVectorSDKCandidateScannerDebugMapLayout(t *testing.T) {
	if runtime.GOARCH != "amd64" {
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
		contributions, err := pecoff.InspectPDBSectionContributions(debug.Bytes, debug.ProgramDatabase)
		if err != nil || len(contributions.Contributions) == 0 {
			t.Fatalf("scanner PDB section contributions: %d records, %v", len(contributions.Contributions), err)
		}
		t.Logf("PDB section contributions=%d version=%#x", len(contributions.Contributions), contributions.Version)
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
		starts := make([]uint32, 0, len(debugUnwind.Functions))
		for _, function := range debugUnwind.Functions {
			starts = append(starts, function.BeginRVA)
		}
		symbols, err := cdriver.ResolveWindowsDebugSymbols(debug, starts)
		if err != nil {
			t.Fatal(err)
		}
		pe, err := pecoff.InspectExecutable(release.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		publicStarts := make(map[uint32]struct{})
		for _, symbol := range scannerBudgetAllMapSymbols(release.LinkMap, nil) {
			if symbol.PreferredAddress >= pe.ImageBase && symbol.PreferredAddress-pe.ImageBase <= uint64(^uint32(0)) {
				publicStarts[uint32(symbol.PreferredAddress-pe.ImageBase)] = struct{}{}
			}
		}
		type privateInterval struct {
			name  string
			bytes uint32
		}
		private := make([]privateInterval, 0)
		var privateBytes, unresolvedBytes uint64
		for i, function := range releaseUnwind.Functions {
			if _, public := publicStarts[function.BeginRVA]; public {
				continue
			}
			width := function.EndRVA - function.BeginRVA
			symbol := symbols[i]
			if !symbol.Found || symbol.Displacement != 0 || symbol.RVA != function.BeginRVA {
				unresolvedBytes += uint64(width)
				continue
			}
			privateBytes += uint64(width)
			name := symbol.Name
			if len(name) > 128 || strings.ContainsAny(name, "\\/") {
				name = "<nonportable>"
			}
			private = append(private, privateInterval{name: name, bytes: width})
		}
		privateCount := len(private)
		sort.Slice(private, func(i, j int) bool {
			if private[i].bytes != private[j].bytes {
				return private[i].bytes > private[j].bytes
			}
			return private[i].name < private[j].name
		})
		if len(private) > 10 {
			private = private[:10]
		}
		t.Logf("release EXE=%d debug EXE=%d PDB=%d intervals=%d private exact=%d/%d bytes unresolved=%d bytes top private=%v",
			len(release.Bytes), len(debug.Bytes), len(debug.ProgramDatabase), len(releaseUnwind.Functions),
			privateCount, privateBytes, unresolvedBytes, private)
	})
}
