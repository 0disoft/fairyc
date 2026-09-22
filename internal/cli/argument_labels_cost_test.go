package cli

import (
	"bytes"
	"debug/pe"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Opt-in measurement: ordinary test runs never build the comparison corpus.
func TestArgumentLabelCostMeasurement(t *testing.T) {
	lane := os.Getenv("FAIRYC_LABEL_COST_LANE")
	if lane == "" {
		t.Skip("opt-in cost measurement")
	}
	if lane != "baseline" && lane != "current" {
		t.Fatal("invalid lane")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("requires Windows AMD64")
	}
	output := os.Getenv("FAIRYC_LABEL_COST_OUTPUT")
	if output == "" {
		t.Fatal("missing output root")
	}
	variants := []string{"plain", "scanner"}
	if lane == "current" {
		variants = append(variants, "labels")
	}
	if os.Getenv("FAIRYC_LABEL_COST_REVERSE") == "1" {
		for i, j := 0, len(variants)-1; i < j; i, j = i+1, j-1 {
			variants[i], variants[j] = variants[j], variants[i]
		}
	}
	rows := []map[string]any{}
	for _, variant := range variants {
		root := t.TempDir()
		var hashes map[string]string
		if variant == "scanner" {
			root, hashes = tokenIndexWorkspace(t, false)
		} else {
			var text strings.Builder
			text.WriteString("module main; fn usize expression(usize minimum, usize depth, bool allow_record) { return if (allow_record) { minimum + depth } else { depth }; } fn i32 main() { var usize total = 0;\n")
			for i := 0; i < 64; i++ {
				if variant == "labels" {
					fmt.Fprintln(&text, "total += expression(.minimum = 1, .depth = 0, .allow_record = true);")
				} else {
					fmt.Fprintln(&text, "total += expression(1, 0, true);")
				}
			}
			text.WriteString("assert(total == 64); return 0; }")
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.label-cost\"\nname = \"label-cost\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
			writeTestSource(t, root, "src/main.fy", text.String())
			hashes = map[string]string{"src/main.fy": tokenIndexHash([]byte(text.String()))}
		}
		// Exclude setup and initial lock creation. Every measured iteration reloads
		// source and parses/checks/lowers verified MIR; no package cache is used.
		check := func() compiler.Result {
			w, err := workspaceloader.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			r, err := workspacecompiler.Check(w)
			if err != nil || !r.Verified {
				t.Fatalf("%s check: %v %+v", variant, err, r.Semantic.Diagnostics)
			}
			return r
		}
		checked := check()
		const iterations = 8
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		start := time.Now()
		for i := 0; i < iterations; i++ {
			checked = check()
		}
		elapsed := time.Since(start).Nanoseconds()
		runtime.ReadMemStats(&after)
		row := map[string]any{"variant": variant, "iterations": iterations, "check_ns": elapsed / iterations, "go_alloc_bytes": (after.TotalAlloc - before.TotalAlloc) / iterations, "go_mallocs": (after.Mallocs - before.Mallocs) / iterations, "gc_cycles": after.NumGC - before.NumGC, "source_sha256": hashes}
		// Separate artifact/cache observations from frontend timing.
		if variant != "scanner" {
			w, err := workspaceloader.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			s, err := workspacecompiler.NewSession(workspacecompiler.Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 4, MaxBytes: 16 << 20}, MaxArtifactBytes: 8 << 20})
			if err != nil {
				t.Fatal(err)
			}
			cold, err := s.Check(w)
			if err != nil || !cold.Compiler.Verified || cold.Cache.Stored != 1 {
				t.Fatalf("cold cache: %+v %v", cold.Cache, err)
			}
			warm, err := s.Check(w)
			if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != workspacecompiler.OutcomeAllHit {
				t.Fatalf("warm cache: %+v %v", warm.Cache, err)
			}
			row["cache_stats"] = s.Stats()
			entry := tokenScannerEntry(t, checked)
			unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
			if err != nil {
				t.Fatal(err)
			}
			support, err := windowsTargetSupportSource(unit.ForeignSymbols)
			if err != nil {
				t.Fatal(err)
			}
			prefix := filepath.Join(output, lane+"-"+variant)
			if err = os.WriteFile(prefix+".c", []byte(unit.Source), 0600); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(prefix+"-runtime.c", []byte(support), 0600); err != nil {
				t.Fatal(err)
			}
			row["c_bytes"] = len(unit.Source)
			row["c_sha256"] = tokenIndexHash([]byte(unit.Source))
			artifacts := []map[string]any{}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				var diagnostics bytes.Buffer
				start := time.Now()
				artifact, code := compileWindowsProgram(checked.MIR, entry, target.X8664WindowsMSVC, profile, &diagnostics)
				nativeNS := time.Since(start).Nanoseconds()
				if code != ExitSuccess {
					t.Fatal(diagnostics.String())
				}
				outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("native: %+v %v", outcome, err)
				}
				image, err := pe.NewFile(bytes.NewReader(artifact.Bytes))
				if err != nil {
					t.Fatal(err)
				}
				sections := map[string]uint32{}
				for _, s := range image.Sections {
					sections[s.Name] = s.VirtualSize
				}
				image.Close()
				objects := map[string]int{}
				for _, o := range artifact.Objects {
					objects[o.Name] = len(o.Bytes)
				}
				artifacts = append(artifacts, map[string]any{"profile": profile.Name, "native_ns": nativeNS, "exe_bytes": len(artifact.Bytes), "pdb_bytes": len(artifact.ProgramDatabase), "object_bytes": objects, "pe_virtual_section_bytes": sections, "msvc": artifact.Toolchain.Version, "exit_code": outcome.ExitCode})
			}
			row["artifacts"] = artifacts
		}
		rows = append(rows, row)
	}
	encoded, err := json.Marshal(map[string]any{"lane": lane, "go": runtime.Version(), "gomaxprocs": runtime.GOMAXPROCS(0), "rows": rows})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("LABEL_COST_REPORT " + string(encoded))
}
