package compiler_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// The indexed scanner uses Vec<T>, but never spells the private vector-for
// source form. Its executable output must be independent of candidate mode.
func TestPrivateVectorSDKCandidateUnusedSourceProducesSameMIRAndC(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	for _, name := range realScannerSupportFiles() {
		data, err := os.ReadFile(filepath.Join(example, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		writeVectorCacheFile(t, root, name, string(data))
	}
	writeVectorCacheFile(t, root, "src/scanner.fy", realScannerVectorVariants(t)[0].scanner)
	writeVectorCacheFile(t, root, "src/main.fy", realScannerExecutionMain)
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	input, inputErr := workspace.SemanticInput()
	coordinates, coordinateErr := workspace.PackageCoordinates()
	owners, ownerErr := workspace.VectorCandidateStorageOwners()
	rootID, rootErr := workspace.RootPackageID()
	if inputErr != nil || coordinateErr != nil || ownerErr != nil || rootErr != nil || len(owners) != 1 {
		t.Fatalf("indexed scanner authority: %v %v %v %v owners=%v", inputErr, coordinateErr, ownerErr, rootErr, owners)
	}

	type artifact struct {
		mir             []byte
		c               string
		functions       int
		foreignSymbols  []string
		specializations int
	}
	build := func(candidate bool) artifact {
		t.Helper()
		var analysis compiler.GenericAnalysis
		var err error
		if candidate {
			analysis, err = compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
		} else {
			analysis, err = compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
		}
		if err != nil || analysis.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow ||
			len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
			t.Fatalf("candidate=%v Flow: %v diagnostics=%+v unsupported=%+v", candidate, err,
				analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
		}
		checked, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
		if err != nil || !checked.Verified {
			t.Fatalf("candidate=%v MIR: %v diagnostics=%+v", candidate, err, checked.Semantic.Diagnostics)
		}
		encoded, err := mircodec.EncodeStorageTransfers(checked.MIR, 16<<20)
		if err != nil {
			t.Fatalf("candidate=%v MIR codec: %v", candidate, err)
		}
		unit, err := cbackend.EmitStorageTransfers(checked.MIR,
			hir.QualifiedModulePath(rootID, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatalf("candidate=%v C emission: %v", candidate, err)
		}
		return artifact{mir: encoded, c: unit.Source, functions: len(unit.Functions),
			foreignSymbols: unit.ForeignSymbols, specializations: len(analysis.Plan.Specializations)}
	}
	stable := build(false)
	candidate := build(true)
	if !bytes.Equal(stable.mir, candidate.mir) {
		t.Fatalf("unused vector-for candidate changed verified MIR: stable=%d candidate=%d bytes", len(stable.mir), len(candidate.mir))
	}
	if stable.c != candidate.c || stable.functions != candidate.functions ||
		!reflect.DeepEqual(stable.foreignSymbols, candidate.foreignSymbols) ||
		stable.specializations != candidate.specializations {
		t.Fatalf("unused vector-for candidate changed C transport: C=%d/%d functions=%d/%d foreign=%d/%d specializations=%d/%d",
			len(stable.c), len(candidate.c), stable.functions, candidate.functions,
			len(stable.foreignSymbols), len(candidate.foreignSymbols), stable.specializations, candidate.specializations)
	}
}
