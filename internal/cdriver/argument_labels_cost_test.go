package cdriver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Debug probes use the existing private CodeView/PDB builder, not a new CLI flag.
func TestArgumentLabelDebugCost(t *testing.T) {
	root := os.Getenv("FAIRYC_LABEL_COST_OUTPUT")
	if root == "" {
		t.Skip("opt-in cost measurement")
	}
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{}
	for repetition := 0; repetition < 2; repetition++ {
		variants := []string{"baseline-plain", "current-plain", "current-labels"}
		if repetition == 1 {
			variants = []string{"current-labels", "current-plain", "baseline-plain"}
		}
		for _, variant := range variants {
			unit, err := os.ReadFile(filepath.Join(root, variant+".c"))
			if err != nil {
				t.Fatal(err)
			}
			support, err := os.ReadFile(filepath.Join(root, variant+"-runtime.c"))
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []Profile{LowProfile, OptimizedProfile} {
				artifact, err := buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode, []windowsMSVCSource{{Name: "unit.c", Contents: string(unit)}, {Name: "runtime.c", Contents: string(support)}}, WindowsMSVCBuildOptions{CaptureProgramDatabase: true, EmbeddedCodeView: true})
				if err != nil {
					t.Fatal(err)
				}
				if len(artifact.ProgramDatabase) == 0 {
					t.Fatal("missing requested PDB")
				}
				outcome, err := ExecuteWindowsMSVCTarget(artifact)
				if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("debug execute: %+v %v", outcome, err)
				}
				objects := map[string]int{}
				for _, o := range artifact.Objects {
					objects[o.Name] = len(o.Bytes)
				}
				rows = append(rows, map[string]any{"variant": variant, "repetition": repetition, "profile": profile.Name, "exe_bytes": len(artifact.Bytes), "pdb_bytes": len(artifact.ProgramDatabase), "object_bytes": objects, "msvc": artifact.Toolchain.Version, "exit_code": outcome.ExitCode})
			}
		}
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("LABEL_DEBUG_REPORT " + string(encoded))
}
