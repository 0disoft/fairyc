//go:build windows

package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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
	checkScannerDebugPair(t, "independent C", cRelease, cDebug, nil)
	variants := realScannerVectorVariants(t)
	var indexed map[string]uint32
	for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		program, entry, _ := scannerBudgetFairyProgram(t, variant)
		release, releaseUnit := buildRealScannerVectorNativeWithStorageAndMap(t, program, entry,
			cdriver.OptimizedProfile, extraSupport, "", true)
		debug, debugUnit := buildRealScannerVectorNativeWithDebugMap(t, program, entry,
			cdriver.OptimizedProfile, extraSupport)
		if releaseUnit.Source != debugUnit.Source {
			t.Fatalf("%s generated C changed between optimized builds", variant.name)
		}
		functions := checkScannerDebugPair(t, variant.name, release, debug, debugUnit.Functions)
		if indexed == nil {
			indexed = functions
			continue
		}
		for _, name := range []string{"scanner::report_work", "scanner::storage"} {
			left, leftOK := indexed[name]
			right, rightOK := functions[name]
			if !leftOK || !rightOK {
				t.Fatalf("missing linked scanner procedure %s: indexed=%v vector=%v", name, leftOK, rightOK)
			}
			t.Logf("linked .text %s: indexed=%d vector=%d delta=%+d bytes", name,
				left, right, int64(right)-int64(left))
		}
	}
}

func checkScannerDebugPair(t *testing.T, name string, release, debug cdriver.WindowsHostArtifact, sourceFunctions []string) map[string]uint32 {
	t.Helper()
	scannerFunctions := make(map[string]uint32)
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
		debugPE, err := pecoff.InspectExecutable(debug.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		ownership := scannerPDBUnwindOwners(debugPE, debugUnwind.Functions, contributions.Contributions)
		if ownership.TotalBytes != ownership.Attributed+ownership.Unmatched+ownership.Ambiguous {
			t.Fatalf("PDB unwind ownership does not partition validated code: %+v", ownership)
		}
		// DBI contributions include leaf code and padding omitted by .pdata.
		// Report them separately: a contribution is not an exact function span.
		codeContributions := make(map[uint16]uint64)
		var codeContributionBytes uint64
		for _, contribution := range contributions.Contributions {
			if debugPE.Sections[contribution.Section-1].Name != ".text" || contribution.Characteristics&0x20 == 0 {
				continue
			}
			codeContributions[contribution.ModuleIndex] += uint64(contribution.Size)
			codeContributionBytes += uint64(contribution.Size)
		}
		type moduleSize struct {
			index       uint16
			name        string
			object      string
			unwindBytes uint64
			codeBytes   uint64
		}
		modules := make([]moduleSize, 0, len(contributions.Modules))
		for index, bytes := range codeContributions {
			if int(index) >= len(contributions.Modules) {
				t.Fatalf("PDB module %d has no validated module record", index)
			}
			module := contributions.Modules[index]
			modules = append(modules, moduleSize{index: index, name: module.Name, object: module.ObjectName,
				unwindBytes: ownership.ByModule[index], codeBytes: bytes})
		}
		sort.Slice(modules, func(i, j int) bool {
			if modules[i].codeBytes != modules[j].codeBytes {
				return modules[i].codeBytes > modules[j].codeBytes
			}
			return modules[i].index < modules[j].index
		})
		if len(modules) > 8 {
			modules = modules[:8]
		}
		t.Logf("PDB unwind owner coverage attributed=%d unmatched=%d ambiguous=%d total=%d intervals=%d; .text code contribution sum=%d top modules=%v",
			ownership.Attributed, ownership.Unmatched, ownership.Ambiguous, ownership.TotalBytes,
			ownership.MatchedEntries, codeContributionBytes, modules)
		procedureModules := map[string]bool{"probe.obj": name == "independent C", "unit.obj": name != "independent C", "runtime.obj": name != "independent C"}
		for _, module := range contributions.Modules {
			if !procedureModules[module.Name] {
				continue
			}
			procedures, err := pecoff.InspectPDBModuleProcedureRecords(debug.Bytes, debug.ProgramDatabase, module.Index)
			if err != nil {
				t.Fatalf("PDB procedures for %s: %v", module.Name, err)
			}
			var matched, unmatched, codeBytes uint64
			seen := make(map[[3]uint32]struct{})
			linked := make([]pecoff.PDBProcedureRecord, 0, len(procedures))
			for _, procedure := range procedures {
				if procedure.Section == 0 || int(procedure.Section) > len(debugPE.Sections) ||
					debugPE.Sections[procedure.Section-1].Name != ".text" {
					unmatched++
					continue
				}
				key := [3]uint32{uint32(procedure.Section), procedure.Offset, procedure.Size}
				if _, duplicate := seen[key]; duplicate {
					// MSVC may fold identical functions into one owned code range.
					// Count the record, but account for its bytes only once.
					matched++
					t.Logf("PDB module %s folded procedure %s at section=%d offset=%d size=%d",
						module.Name, procedure.Name, procedure.Section, procedure.Offset, procedure.Size)
					continue
				}
				owned := false
				for _, contribution := range contributions.Contributions {
					if contribution.ModuleIndex == module.Index && contribution.Section == procedure.Section &&
						contribution.Characteristics&0x20 != 0 && contribution.Offset <= procedure.Offset &&
						uint64(procedure.Offset)+uint64(procedure.Size) <= uint64(contribution.Offset)+uint64(contribution.Size) {
						matched++
						owned = true
						seen[key] = struct{}{}
						codeBytes += uint64(procedure.Size)
						linked = append(linked, procedure)
						break
					}
				}
				if !owned {
					unmatched++
				}
			}
			sort.Slice(linked, func(i, j int) bool { return linked[i].Offset < linked[j].Offset })
			for i := 1; i < len(linked); i++ {
				if uint64(linked[i-1].Offset)+uint64(linked[i-1].Size) > uint64(linked[i].Offset) {
					t.Fatalf("PDB module %s has overlapping procedure records: %+v and %+v", module.Name, linked[i-1], linked[i])
				}
			}
			if unmatched != 0 || matched != uint64(len(procedures)) || codeBytes != codeContributions[module.Index] {
				t.Fatalf("PDB module %s procedures do not partition linked .text: matched=%d unmatched=%d bytes=%d contribution=%d",
					module.Name, matched, unmatched, codeBytes, codeContributions[module.Index])
			}
			if module.Name == "unit.obj" {
				for _, procedure := range linked {
					if procedure.Name == "main" {
						continue
					}
					indexText, ok := strings.CutPrefix(procedure.Name, "fy_fn_")
					index, parseErr := strconv.Atoi(indexText)
					if !ok || parseErr != nil || index < 0 || index >= len(sourceFunctions) {
						t.Fatalf("PDB procedure %q has no emitted Fairy source function", procedure.Name)
					}
					source := sourceFunctions[index]
					var target string
					for _, name := range []string{"scanner::report_work", "scanner::storage"} {
						if strings.HasSuffix(source, "::"+name) {
							target = name
							break
						}
					}
					if target == "" {
						continue
					}
					if _, duplicate := scannerFunctions[target]; duplicate {
						t.Fatalf("duplicate linked scanner procedure %s", target)
					}
					scannerFunctions[target] = procedure.Size
				}
			}
			sort.Slice(linked, func(i, j int) bool { return linked[i].Size > linked[j].Size })
			if len(linked) > 8 {
				linked = linked[:8]
			}
			type sourceProcedure struct {
				Name   string
				Source string
				Bytes  uint32
			}
			top := make([]sourceProcedure, 0, len(linked))
			for _, procedure := range linked {
				row := sourceProcedure{Name: procedure.Name, Bytes: procedure.Size}
				if module.Name == "unit.obj" {
					indexText, ok := strings.CutPrefix(procedure.Name, "fy_fn_")
					index, parseErr := strconv.Atoi(indexText)
					if !ok || parseErr != nil || index < 0 || index >= len(sourceFunctions) {
						t.Fatalf("PDB procedure %q has no emitted Fairy source function", procedure.Name)
					}
					row.Source = sourceFunctions[index]
				}
				top = append(top, row)
			}
			t.Logf("PDB module %s procedure records=%d linked functions=%d code bytes=%d top=%v",
				module.Name, len(procedures), matched, codeBytes, top)
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
	return scannerFunctions
}
