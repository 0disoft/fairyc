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

func TestPrivateVectorSDKCandidateScannerLibraryMembers(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	cSource := scannerCBenchmarkSource(t)
	cRelease, err := cdriver.BuildWindowsMSVCProbeWithMap(toolchain, cdriver.OptimizedProfile, cSource)
	if err != nil {
		t.Fatal(err)
	}
	cVerbose, err := cdriver.BuildWindowsMSVCProbeWithLibraryLog(toolchain, cdriver.OptimizedProfile, cSource)
	if err != nil {
		t.Fatal(err)
	}
	cMembers := checkScannerLibraryPair(t, "independent C", cRelease, cVerbose)
	support, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "support.c"))
	if err != nil {
		t.Fatal(err)
	}
	extraSupport := "\n" + string(support) + "\nvoid fc_budget_flush(void) { fflush(stdout); }\n"
	variants := realScannerVectorVariants(t)
	var indexedMembers []string
	for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		program, entry, _ := scannerBudgetFairyProgram(t, variant)
		release, releaseUnit := buildRealScannerVectorNativeWithStorageAndMap(t, program, entry,
			cdriver.OptimizedProfile, extraSupport, "", true)
		verbose, verboseUnit := buildRealScannerVectorNativeWithLibraryLog(t, program, entry,
			cdriver.OptimizedProfile, extraSupport)
		if releaseUnit.Source != verboseUnit.Source {
			t.Fatalf("%s generated C changed between optimized builds", variant.name)
		}
		members := checkScannerLibraryPair(t, variant.name, release, verbose)
		if indexedMembers == nil {
			indexedMembers = members
		} else if strings.Join(indexedMembers, "\n") != strings.Join(members, "\n") {
			t.Fatalf("%s selected different libraries from indexed Fairy", variant.name)
		}
	}
	cOnly, fairyOnly := scannerMemberDifference(cMembers, indexedMembers)
	t.Logf("selected library members C=%d Fairy=%d C-only=%d Fairy-only=%d", len(cMembers), len(indexedMembers), len(cOnly), len(fairyOnly))
	if len(cOnly) > 40 {
		cOnly = cOnly[:40]
	}
	if len(fairyOnly) > 40 {
		fairyOnly = fairyOnly[:40]
	}
	t.Logf("first C-only=%v first Fairy-only=%v", cOnly, fairyOnly)
}

func checkScannerLibraryPair(t *testing.T, name string, release, verbose cdriver.WindowsHostArtifact) []string {
	t.Helper()
	var selected []string
	t.Run(name, func(t *testing.T) {
		if len(verbose.LinkerVerbose) == 0 || len(verbose.LinkMap) == 0 {
			t.Fatal("library-verbose build omitted log or map")
		}
		execute := func(artifact cdriver.WindowsHostArtifact) string {
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, strings.NewReader("12345 1000\n"))
			if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("library-verbose scanner execution: %+v %v", outcome, err)
			}
			fields := strings.Fields(outcome.Stdout)
			if len(fields) < 5 || fields[len(fields)-4] != "1000" {
				t.Fatalf("library-verbose timing report is incomplete: %q", outcome.Stdout)
			}
			return strings.Join(fields[:len(fields)-4], " ") + " " + fields[len(fields)-4] + " " + fields[len(fields)-1]
		}
		if execute(release) != execute(verbose) {
			t.Fatal("library-verbose build changed scanner output")
		}
		left, err := pecoff.InspectAMD64Unwind(release.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		right, err := pecoff.InspectAMD64Unwind(verbose.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if len(left.Functions) != len(right.Functions) || len(release.Bytes) != len(verbose.Bytes) {
			t.Fatalf("library logging changed image size or unwind count: %d/%d bytes %d/%d functions",
				len(release.Bytes), len(verbose.Bytes), len(left.Functions), len(right.Functions))
		}
		for i, function := range left.Functions {
			if function.BeginRVA != right.Functions[i].BeginRVA || function.EndRVA != right.Functions[i].EndRVA {
				t.Fatalf("library logging moved unwind interval %d", i)
			}
		}
		selected = scannerLoadedLibraryMembers(verbose.LinkerVerbose)
		if len(selected) == 0 {
			t.Fatal("MSVC log contains no loaded archive members")
		}
		t.Logf("EXE=%d unwind intervals=%d library log bytes=%d selected members=%d",
			len(verbose.Bytes), len(left.Functions), len(verbose.LinkerVerbose), len(selected))
	})
	return selected
}

func scannerLoadedLibraryMembers(log []byte) []string {
	seen := make(map[string]struct{})
	for _, line := range strings.Split(strings.ReplaceAll(string(log), "\r\n", "\n"), "\n") {
		member, ok := strings.CutPrefix(strings.TrimSpace(line), "Loaded ")
		if !ok {
			continue
		}
		member = strings.ToLower(filepath.Base(strings.ReplaceAll(strings.TrimSpace(member), "\\", "/")))
		if strings.Contains(member, ".lib(") && strings.HasSuffix(member, ")") {
			seen[member] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for member := range seen {
		result = append(result, member)
	}
	sort.Strings(result)
	return result
}

func scannerMemberDifference(left, right []string) ([]string, []string) {
	leftSet, rightSet := make(map[string]struct{}, len(left)), make(map[string]struct{}, len(right))
	for _, name := range left {
		leftSet[name] = struct{}{}
	}
	for _, name := range right {
		rightSet[name] = struct{}{}
	}
	var leftOnly, rightOnly []string
	for _, name := range left {
		if _, shared := rightSet[name]; !shared {
			leftOnly = append(leftOnly, name)
		}
	}
	for _, name := range right {
		if _, shared := leftSet[name]; !shared {
			rightOnly = append(rightOnly, name)
		}
	}
	return leftOnly, rightOnly
}
