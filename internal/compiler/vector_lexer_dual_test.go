package compiler_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// The two source spellings share one package graph but retain distinct
// scanner modules. The stable source gate must still reject the candidate.
func dualScannerCheckedProgram(t *testing.T, mainSource string) (mir.Program, string) {
	t.Helper()
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	return dualScannerCheckedProgramWithSources(t, mainSource, variants[0].scanner, variants[4].scanner)
}

func dualScannerCheckedProgramWithSources(t *testing.T, mainSource, indexedSource, candidateSource string) (mir.Program, string) {
	t.Helper()
	module := func(src, name string) string {
		const declaration = "module scanner;"
		if !strings.HasPrefix(src, declaration) || strings.Count(src, declaration) != 1 {
			t.Fatal("checked-in scanner module declaration changed")
		}
		return strings.Replace(src, declaration, "module "+name+";", 1)
	}
	root := t.TempDir()
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	for _, name := range []string{"fairy.toml", "src/numbers.fy", "src/diagnostics.fy", "src/utf8.fy", "src/strings.fy", "src/budget.fy"} {
		data, err := os.ReadFile(filepath.Join(example, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		writeVectorCacheFile(t, root, name, string(data))
	}
	writeVectorCacheFile(t, root, "src/indexed.fy", module(indexedSource, "indexed"))
	writeVectorCacheFile(t, root, "src/candidate.fy", module(candidateSource, "candidate"))
	writeVectorCacheFile(t, root, "src/main.fy", mainSource)
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	input, inputErr := workspace.SemanticInput()
	coordinates, coordinateErr := workspace.PackageCoordinates()
	owners, ownerErr := workspace.VectorCandidateStorageOwners()
	if inputErr != nil || coordinateErr != nil || ownerErr != nil || len(owners) != 1 {
		t.Fatalf("combined scanner authority: %v %v %v owners=%v", inputErr, coordinateErr, ownerErr, owners)
	}
	stable, stableErr := compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
	if stableErr != nil || len(stable.Analysis.Semantic.Unsupported) == 0 {
		t.Fatalf("stable vector source gate changed: %v %+v", stableErr, stable.Analysis.Semantic)
	}
	analysis, err := compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
	if err != nil || analysis.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow || len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
		t.Fatalf("combined scanner Flow: %v diagnostics=%+v unsupported=%+v", err, analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
	}
	checked, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !checked.Verified {
		t.Fatalf("combined scanner MIR: %v diagnostics=%+v", err, checked.Semantic.Diagnostics)
	}
	rootID, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	return checked.MIR, hir.QualifiedModulePath(rootID, "main") + "::main"
}

// Prove that the indexed and private vector-loop scanner sources can coexist
// in one verified program before timing either form inside one process.
func TestPrivateVectorSDKCandidateDualScannerNativeParity(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	const sample = "let n=12345; /*open"
	program, entry := dualScannerCheckedProgram(t, dualScannerNativeMain)
	lexed := lexer.Lex(source.NewFile("input.fy", []byte(sample)), source.NewBudget(source.DefaultLimits(), len(sample)))
	cooked, diagnostic := 0, 0
	for _, token := range lexed.Tokens {
		cooked += len(token.Cooked)
	}
	for _, entry := range lexed.Diagnostics {
		diagnostic += len(entry.Token)
	}
	if cooked == 0 || diagnostic == 0 {
		t.Fatal("sample no longer covers cooked and diagnostic buffers")
	}
	wantStorage := fmt.Sprintf("%d %d %d %d %d ", len(sample), cooked, diagnostic, len(lexed.Tokens), len(lexed.Diagnostics))
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			native := runRealScannerVectorNative(t, program, entry, profile)
			parts := strings.Split(native.Stdout, "|")
			if len(parts) != 4 || parts[0] == "" || parts[0] != parts[1] || parts[2] != wantStorage || parts[3] != wantStorage {
				t.Fatalf("combined scanner output: parts=%q want storage=%q", parts, wantStorage)
			}
		})
	}
}

const dualScannerNativeMain = `module main;
import indexed as indexed;
import candidate as candidate;
import std::bytes as bytes;
import std::io as io;
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
@effects(console) fn void number(usize value) {
    assert(switch (io::write_usize(value)) { case .ok(_) : { true } case .err(_) : { false } });
    assert(switch (io::write(" ")) { case .ok(_) : { true } case .err(_) : { false } });
}
@effects(console) fn void separator() {
    assert(switch (io::write("|")) { case .ok(_) : { true } case .err(_) : { false } });
}
@effects(console) fn i32 main() {
    assert(indexed::report_work(b"let n=12345; /*open"[..], false, 32768, 1048576, 256, 16777216) == 0);
    separator();
    assert(candidate::report_work(b"let n=12345; /*open"[..], false, 32768, 1048576, 256, 16777216) == 0);
    separator();
    let bytes::Buffer indexed_input = switch (bytes::copy(b"let n=12345; /*open")) {
        case .ok(value) : { value }
        case .err(_) : { assert(false); }
    };
    let i32 indexed_status = switch (indexed::collect(indexed_input, false, 32768, 1048576, 256, 16777216)) {
        case .ok(stream) : {
            let indexed::Storage usage = indexed::storage(ref stream);
            number(usage.source_bytes); number(usage.cooked_bytes); number(usage.diagnostic_bytes);
            number(usage.tokens); number(usage.diagnostics);
            0
        }
        case .err(status) : { status }
    };
    if (indexed_status != 0) { return indexed_status; }
    separator();
    let bytes::Buffer candidate_input = switch (bytes::copy(b"let n=12345; /*open")) {
        case .ok(value) : { value }
        case .err(_) : { assert(false); }
    };
    let i32 candidate_status = switch (candidate::collect(candidate_input, false, 32768, 1048576, 256, 16777216)) {
        case .ok(stream) : {
            let candidate::Storage usage = candidate::storage(ref stream);
            number(usage.source_bytes); number(usage.cooked_bytes); number(usage.diagnostic_bytes);
            number(usage.tokens); number(usage.diagnostics);
            0
        }
        case .err(status) : { status }
    };
    if (candidate_status != 0) { return candidate_status; }
    assert(unsafe { clean() } == 1);
    return 0;
}`
