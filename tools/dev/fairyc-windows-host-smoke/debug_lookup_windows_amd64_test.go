//go:build windows && amd64

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/pecoff"
)

func retainedDebugMapRVA(t *testing.T, files map[string][]byte, symbol, object string) uint32 {
	t.Helper()
	image, err := pecoff.InspectExecutable(files["program.exe"])
	if err != nil {
		t.Fatal(err)
	}
	var rva uint32
	for _, line := range strings.Split(string(files["program.map"]), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[1] != symbol || !strings.Contains(fields[0], ":") {
			continue
		}
		address, err := strconv.ParseUint(fields[2], 16, 64)
		if err != nil || address <= image.ImageBase || address-image.ImageBase > 0xffffffff || rva != 0 || fields[len(fields)-1] != object {
			t.Fatalf("invalid or duplicate owned map row: %s", line)
		}
		rva = uint32(address - image.ImageBase)
	}
	if rva == 0 {
		t.Fatalf("no owned map row for %s", symbol)
	}
	return rva
}

func retainedDebugSourceLine(t *testing.T, data []byte, declaration string) uint32 {
	t.Helper()
	var result uint32
	for i, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, declaration) {
			if result != 0 {
				t.Fatalf("ambiguous source declaration %q", declaration)
			}
			result = uint32(i + 1)
		}
	}
	if result == 0 {
		t.Fatalf("missing source declaration %q", declaration)
	}
	return result
}

func TestWindowsRetainedDebugLookup(t *testing.T) {
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(t.TempDir(), "debug-lookup.exe")
	build := exec.Command("go", "build", "-mod=readonly", "-o", tool, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build independent lookup tool: %v\n%s", err, output)
	}
	store := os.Getenv("FAIRYC_RETAINED_DEBUG_STORE")
	if store == "" {
		store = t.TempDir()
	}
	store, err = filepath.Abs(store)
	if err != nil {
		t.Fatal(err)
	}
	type retained struct {
		key   string
		files map[string][]byte
	}
	var bundles []retained
	var lookupRows []retainedDebugLookup
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, manifest, files := buildRetainedDebugFixture(t, tc, profile)
			outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
			if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("debug native execution: %+v %v", outcome, err)
			}
			for _, target := range []struct{ symbol, object string }{{"main", "unit.obj"}, {"fairyc_debug_boundary", "runtime.obj"}} {
				codeview, err := pecoff.InspectCodeViewRelocations(files[target.object])
				if err != nil || !hasCodeViewRelocationPair(codeview, target.symbol) {
					object, _ := pecoff.InspectObject(files[target.object])
					for _, symbol := range object.SymbolRecords {
						if strings.HasPrefix(symbol.Name, "$LN") {
							t.Logf("debug label: %+v", symbol)
						}
					}
					t.Fatalf("CodeView %s: %+v %v", target.symbol, codeview, err)
				}
				endLabels := 0
				for _, pair := range codeview.Pairs {
					if pair.SectionEndLabel {
						endLabels++
					}
				}
				t.Logf("%s %s CodeView relocation pairs=%d exact code-end labels=%d", profile.Name, target.object, len(codeview.Pairs), endLabels)
			}
			key, err := retainWindowsDebugBundle(store, manifest, files)
			if err != nil {
				t.Fatal(err)
			}
			bundles = append(bundles, retained{key, files})
			for _, target := range []struct{ symbol, source, object, declaration string }{
				{"main", "unit.c", "unit.obj", "int main(void) {"},
				{"fairyc_debug_boundary", "runtime.c", "runtime.obj", "__declspec(noinline) int32_t fairyc_debug_boundary(int32_t value) {"},
			} {
				rva := retainedDebugMapRVA(t, files, target.symbol, target.object)
				line := retainedDebugSourceLine(t, files[target.source], target.declaration)
				// Builders already removed their source/PDB directories. Only the retained
				// store, image hash and RVA reach a fresh process with an unrelated cwd.
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				command := exec.CommandContext(ctx, tool, "--debug-lookup", "--store", store, "--image-sha", key, "--rva", fmt.Sprintf("0x%x", rva))
				command.Dir = t.TempDir()
				command.Env = append(os.Environ(), "_NT_SYMBOL_PATH="+command.Dir, "_NT_ALTERNATE_SYMBOL_PATH="+command.Dir)
				output, err := command.CombinedOutput()
				cancel()
				if err != nil {
					t.Fatalf("fresh-process retained lookup: %v\n%s", err, output)
				}
				var row retainedDebugLookup
				if err := json.Unmarshal(output, &row); err != nil {
					t.Fatal(err)
				}
				if row.Schema != "fairyc.windows-debug-lookup/v1" || row.ImageSHA256 != key || row.PDBSHA256 != debugDigest(files["program.pdb"]) || row.SourceRevision != manifest.SourceRevision || row.Profile != profile.Name || row.RVA != rva || row.Symbol != target.symbol || row.SourceFile != target.source || row.SourceSHA256 != debugDigest(files[target.source]) || row.Line != line || row.LineText != strings.Split(string(files[target.source]), "\n")[line-1] {
					t.Fatalf("retained lookup drift: %+v; want %s %s:%d", row, target.symbol, target.source, line)
				}
				lookupRows = append(lookupRows, row)
				t.Logf("retained %s/%s: RVA=%#x exact %s:%d, image=%s PDB=%s", profile.Name, target.symbol, rva, row.SourceFile, row.Line, key, row.PDBSHA256)
				if _, err := lookupRetainedWindowsDebug(store, key, rva+1); err == nil {
					t.Fatal("interior PC accepted as exact function start")
				}
			}
		})
	}
	if len(bundles) != 2 || len(lookupRows) != 4 {
		t.Fatal("incomplete retained debug matrix")
	}
	for i, bundle := range bundles {
		image, pdb := bundle.files["program.exe"], bundle.files["program.pdb"]
		if _, err := symbolicateWindowsProgram(image, bundles[1-i].files["program.pdb"], 0x1000); err == nil {
			t.Fatal("cross-profile PDB accepted")
		}
		for _, offset := range []int{4, 20} {
			bad := append([]byte(nil), image...)
			start := bytes.Index(bad, []byte("RSDS"))
			if start < 0 {
				t.Fatal("no RSDS")
			}
			bad[start+offset] ^= 1
			if _, err := symbolicateWindowsProgram(bad, pdb, 0x1000); err == nil {
				t.Fatal("mutated RSDS GUID/age accepted")
			}
		}
		if _, err := symbolicateWindowsProgram(image, nil, 0x1000); err == nil {
			t.Fatal("missing PDB accepted")
		}
		if _, err := lookupRetainedWindowsDebug(store, bundle.key, 1); err == nil {
			t.Fatal("header RVA accepted")
		}
		if _, err := lookupRetainedWindowsDebug(store, bundle.key, ^uint32(0)); err == nil {
			t.Fatal("outside RVA accepted")
		}
	}
	if os.Getenv("FAIRYC_RETAINED_DEBUG_STORE") != "" {
		data, err := json.MarshalIndent(lookupRows, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(store, "lookup-results.json"), append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
		t.Logf("retained two PE/PDB bundles and four lookup receipts at %s", store)
	}
}

func TestWindowsRetainedDebugExistingCanary(t *testing.T) {
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	count, err := runWindowsRelocationProbe(tc)
	if err != nil || count != 2 {
		t.Fatalf("existing relocation/symbolication canary: executions=%d %v", count, err)
	}
	t.Log("existing caller.c function/line and relocation canary: two native executions passed")
}
