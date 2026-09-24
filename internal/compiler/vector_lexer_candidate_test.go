package compiler_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type realScannerVectorVariant struct {
	name      string
	candidate bool
	scanner   string
}

// Keep the checked-in scanner edits in one place for parity and cost probes.
func realScannerVectorVariants(t *testing.T) []realScannerVectorVariant {
	t.Helper()
	const before = `    var usize index = 0;
    var usize position = 0;
    while (index < v::len<Token>(ref tokens)) {
        let ref Token token = v::at<Token>(ref tokens, index);`
	const after = `    var usize position = 0;
    for (let ref Token token in ref tokens) {`
	const endBefore = `        position = token.end;
        index += 1;
    }
    assert(position == input.len);`
	const endAfter = `        position = token.end;
    }
    assert(position == input.len);`
	const cookedBefore = `    while (index < v::len<bytes::Buffer>(ref stream.cooked)) {
        cooked += bytes::view(v::at<bytes::Buffer>(ref stream.cooked, index)).len; index += 1;
    }`
	const cookedAfter = `    for (let ref bytes::Buffer item in ref stream.cooked) {
        cooked += bytes::view(item).len;
    }`
	const diagnosticBefore = `    while (index < v::len<diagnostics::Entry>(ref stream.errors)) {
        diagnostic += diagnostics::token_view(v::at<diagnostics::Entry>(ref stream.errors, index)).len;
        index += 1;
    }`
	const diagnosticAfter = `    for (let ref diagnostics::Entry item in ref stream.errors) {
        diagnostic += diagnostics::token_view(item).len;
    }`
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	scannerBytes, err := os.ReadFile(filepath.Join(example, "src", "scanner.fy"))
	if err != nil {
		t.Fatal(err)
	}
	scanner := string(scannerBytes)
	if strings.Count(scanner, before) != 1 || strings.Count(scanner, endBefore) != 1 ||
		strings.Count(scanner, cookedBefore) != 1 || strings.Count(scanner, diagnosticBefore) != 1 {
		t.Fatal("real scanner loop changed; refresh the private candidate comparison")
	}
	tokenScanner := strings.Replace(strings.Replace(scanner, before, after, 1), endBefore, endAfter, 1)
	cookedScanner := strings.Replace(scanner, cookedBefore, cookedAfter, 1)
	diagnosticScanner := strings.Replace(scanner, diagnosticBefore, diagnosticAfter, 1)
	allScanner := strings.Replace(strings.Replace(tokenScanner, cookedBefore, cookedAfter, 1),
		diagnosticBefore, diagnosticAfter, 1)
	return []realScannerVectorVariant{
		{name: "explicit index", scanner: scanner},
		{name: "token vector for", candidate: true, scanner: tokenScanner},
		{name: "cooked vector for", candidate: true, scanner: cookedScanner},
		{name: "diagnostic vector for", candidate: true, scanner: diagnosticScanner},
		{name: "all vector for", candidate: true, scanner: allScanner},
	}
}

func TestPrivateVectorSDKCandidateRealScannerNativeParity(t *testing.T) {
	const scannerInput = "let n=12345; /*open"
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	variants := realScannerVectorVariants(t)

	lexed := lexer.Lex(source.NewFile("input.fy", []byte(scannerInput)),
		source.NewBudget(source.DefaultLimits(), len(scannerInput)))
	cookedBytes, diagnosticBytes := 0, 0
	for _, token := range lexed.Tokens {
		cookedBytes += len(token.Cooked)
	}
	for _, diagnostic := range lexed.Diagnostics {
		diagnosticBytes += len(diagnostic.Token)
	}
	if cookedBytes == 0 || diagnosticBytes == 0 {
		t.Fatal("real scanner sample no longer covers cooked and diagnostic storage")
	}
	wantStorage := fmt.Sprintf("%d %d %d %d %d ", len(scannerInput), cookedBytes,
		diagnosticBytes, len(lexed.Tokens), len(lexed.Diagnostics))
	outputs := make(map[string]map[string]realScannerVectorEvidence)
	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			root := t.TempDir()
			for _, name := range []string{"fairy.toml", "src/numbers.fy", "src/diagnostics.fy", "src/utf8.fy", "src/strings.fy", "src/budget.fy"} {
				data, err := os.ReadFile(filepath.Join(example, filepath.FromSlash(name)))
				if err != nil {
					t.Fatal(err)
				}
				writeVectorCacheFile(t, root, name, string(data))
			}
			writeVectorCacheFile(t, root, "src/scanner.fy", variant.scanner)
			writeVectorCacheFile(t, root, "src/main.fy", `module main;
import scanner as scanner;
import std::bytes as bytes;
import std::io as io;
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
@effects(console) fn void number(usize value) {
    assert(switch (io::write_usize(value)) { case .ok(_) : { true } case .err(_) : { false } });
    assert(switch (io::write(" ")) { case .ok(_) : { true } case .err(_) : { false } });
}
@effects(console) fn i32 exercise() {
    let i32 empty = scanner::report_work(b""[..], false, 32768, 1048576, 256, 16777216);
    if (empty != 0) { return empty; }
    let i32 sample = scanner::report_work(b"let n=12345; /*open"[..], false, 32768, 1048576, 256, 16777216);
    if (sample != 0) { return sample; }
    let bytes::Buffer owned = switch (bytes::copy(b"let n=12345; /*open")) {
        case .ok(value) : { value }
        case .err(_) : { assert(false); bytes::empty() }
    };
    return switch (scanner::collect(owned, false, 32768, 1048576, 256, 16777216)) {
        case .ok(stream) : {
            let scanner::Storage usage = scanner::storage(ref stream);
            number(usage.source_bytes); number(usage.cooked_bytes); number(usage.diagnostic_bytes);
            number(usage.tokens); number(usage.diagnostics);
            0
        }
        case .err(status) : { status }
    };
}
@effects(console) fn i32 main() {
    let i32 status = exercise();
    assert(unsafe { clean() } == 1);
    return status;
}`)
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			input, err := workspace.SemanticInput()
			if err != nil {
				t.Fatal(err)
			}
			coordinates, err := workspace.PackageCoordinates()
			if err != nil {
				t.Fatal(err)
			}
			owners, err := workspace.VectorCandidateStorageOwners()
			if err != nil || len(owners) != 1 {
				t.Fatalf("candidate SDK authority: %v owners=%v", err, owners)
			}
			if variant.candidate {
				stable, err := compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
				if err != nil || len(stable.Analysis.Semantic.Unsupported) == 0 {
					t.Fatalf("stable source gate changed: %v %+v", err, stable.Analysis.Semantic)
				}
			}
			var analysis compiler.GenericAnalysis
			if variant.candidate {
				analysis, err = compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
			} else {
				analysis, err = compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
			}
			if err != nil || analysis.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow ||
				len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
				t.Fatalf("real scanner source and Flow: %v diagnostics=%+v unsupported=%+v", err,
					analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
			}
			result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
			if err != nil || !result.Verified {
				t.Fatalf("real scanner verified MIR: %v %+v", err, result.Semantic.Diagnostics)
			}
			if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
				return
			}
			rootID, err := workspace.RootPackageID()
			if err != nil {
				t.Fatal(err)
			}
			outputs[variant.name] = make(map[string]realScannerVectorEvidence)
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					measured := runRealScannerVectorNative(t, result.MIR,
						hir.QualifiedModulePath(rootID, "main")+"::main", profile)
					measured.Specializations = len(analysis.Plan.Specializations)
					outputs[variant.name][profile.Name] = measured
				})
			}
		})
	}
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			baseline := outputs["explicit index"][profile.Name]
			if baseline.Stdout == "" || !strings.HasSuffix(baseline.Stdout, wantStorage) {
				t.Fatalf("%s scanner storage oracle differs: got=%q want suffix=%q", profile.Name, baseline.Stdout, wantStorage)
			}
			for _, name := range []string{"token vector for", "cooked vector for", "diagnostic vector for", "all vector for"} {
				if candidate := outputs[name][profile.Name]; candidate.Stdout != baseline.Stdout {
					t.Fatalf("%s %s output differs: explicit=%q candidate=%q", profile.Name, name, baseline.Stdout, candidate.Stdout)
				}
			}
			for _, name := range []string{"explicit index", "token vector for", "cooked vector for", "diagnostic vector for", "all vector for"} {
				measured := outputs[name][profile.Name]
				report, err := json.Marshal(struct {
					Variant string `json:"variant"`
					Profile string `json:"profile"`
					realScannerVectorEvidence
				}{name, profile.Name, measured})
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("VECTOR_LEXER_SIZE %s", report)
			}
		}
	}
}

type realScannerVectorEvidence struct {
	Stdout          string `json:"-"`
	CBytes          int    `json:"c_bytes"`
	Functions       int    `json:"functions"`
	ForeignSymbols  int    `json:"foreign_symbols"`
	Specializations int    `json:"specializations"`
	UnitObjectBytes int    `json:"unit_object_bytes"`
	EXEBytes        int    `json:"exe_bytes"`
	MSVC            string `json:"msvc"`
}

func runRealScannerVectorNative(t *testing.T, program mir.Program, entry string, profile cdriver.Profile) realScannerVectorEvidence {
	t.Helper()
	artifact, unit := buildRealScannerVectorNative(t, program, entry, profile, "")
	outcome, err := cdriver.ExecuteWindowsHost(artifact)
	if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
		t.Fatalf("native scanner: %+v %v", outcome, err)
	}
	unitBytes := 0
	for _, object := range artifact.Objects {
		if object.Name == "unit.obj" {
			unitBytes = len(object.Bytes)
		}
	}
	if unitBytes == 0 {
		t.Fatal("native scanner is missing its generated unit object")
	}
	return realScannerVectorEvidence{
		Stdout: outcome.Stdout, CBytes: len(unit.Source), Functions: len(unit.Functions),
		ForeignSymbols: len(unit.ForeignSymbols), UnitObjectBytes: unitBytes,
		EXEBytes: len(artifact.Bytes), MSVC: artifact.Toolchain.Version,
	}
}

func buildRealScannerVectorNative(t *testing.T, program mir.Program, entry string, profile cdriver.Profile, extraSupport string) (cdriver.WindowsHostArtifact, cbackend.Unit) {
	return buildRealScannerVectorNativeWithStorage(t, program, entry, profile, extraSupport, "")
}

func buildRealScannerVectorNativeWithStorage(t *testing.T, program mir.Program, entry string, profile cdriver.Profile, extraSupport, storageOverride string) (cdriver.WindowsHostArtifact, cbackend.Unit) {
	t.Helper()
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.EmitStorageTransfers(program, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	storage, err := cstdlib.WindowsMSVCVectorStorageSource()
	if err != nil {
		t.Fatal(err)
	}
	if storageOverride != "" {
		storage = storageOverride
	}
	symbols := []string{"fy_std_system_heap_allocate_v0"}
	for _, symbol := range unit.ForeignSymbols {
		if strings.HasPrefix(symbol, "fy_std_") {
			symbols = append(symbols, symbol)
		}
	}
	sdk, err := cstdlib.WindowsMSVCSourceForSymbols(symbols)
	if err != nil {
		t.Fatal(err)
	}
	heap, err := cstdlib.WindowsMSVCSourceForSymbols(symbols[:1])
	if err != nil || !strings.HasPrefix(storage, heap) {
		t.Fatalf("private scanner storage heap boundary: %v", err)
	}
	artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source,
		scalar+vectorStorageAllocator+sdk+strings.TrimPrefix(storage, heap)+vectorForNativeProbe+extraSupport)
	if err != nil {
		t.Fatal(err)
	}
	return artifact, unit
}
