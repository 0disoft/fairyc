package compiler_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestCheckProducesVerifiedMIR(t *testing.T) {
	t.Parallel()
	result, err := compiler.Check([]semantic.File{
		parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;\nfn i32 main() { return 42; }\n"),
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !result.Verified || result.Semantic.CompletedThrough != semantic.PhaseFlow || len(result.MIR.Functions) != 1 || result.MIR.Functions[0].Name != "main::main" {
		t.Fatalf("Check() result = %+v", result)
	}
}

func TestCheckProducesVerifiedMIRForOwnedTextExample(t *testing.T) {
	t.Parallel()
	result, err := compiler.Check([]semantic.File{
		parseCompilerFileFromPath(t, filepath.Join("..", "..", "examples", "owned_text", "main.fy")),
		parseCompilerFileFromPath(t, filepath.Join("..", "..", "examples", "owned_text", "owned_text.fy")),
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !result.Verified || result.Semantic.CompletedThrough != semantic.PhaseFlow {
		t.Fatalf("Check() result = %+v", result)
	}
}

func TestCheckProducesVerifiedMIRForFallibleRegionPlacement(t *testing.T) {
	t.Parallel()
	stdlib, err := sdkstdlib.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	files := make([]semantic.File, 0, len(stdlib.Sources)+1)
	for _, value := range stdlib.Sources {
		files = append(files, semantic.File{Package: hir.DefaultPackage, Path: value.Path, Module: value.Module})
	}
	regionSource, err := os.ReadFile(filepath.Join("..", "..", "examples", "regions", "src", "main.fy"))
	if err != nil {
		t.Fatalf("ReadFile(region example) error = %v", err)
	}
	files = append(files, parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(regionSource)))
	result, err := compiler.Check(files)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !result.Verified || result.Semantic.CompletedThrough != semantic.PhaseFlow {
		t.Fatalf("Check() result = %+v", result)
	}
}

func TestCheckReturnsSourceDiagnosticsWithoutMIR(t *testing.T) {
	t.Parallel()
	result, err := compiler.Check([]semantic.File{
		parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;\nfn i32 main() { return missing; }\n"),
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Verified || result.Semantic.FailedPhase != semantic.PhaseResolve || len(result.Semantic.Diagnostics) != 1 || len(result.MIR.Functions) != 0 {
		t.Fatalf("Check() result = %+v", result)
	}
}

func TestCheckPackagesClassifiesInputFailure(t *testing.T) {
	t.Parallel()
	_, err := compiler.CheckPackages(semantic.PackageInput{
		Files: []semantic.File{parseCompilerFile(t, 1, "main.fy", "module main;\nimport util as u;\nfn i32 main() { return u::value(); }\n")},
	})
	var pipelineError *compiler.Error
	if !errors.As(err, &pipelineError) || pipelineError.Stage != compiler.StagePackageInput {
		t.Fatalf("CheckPackages() error = %v", err)
	}
}

func TestAnalyzePackagesSeparatesSemanticAnalysisFromVerifiedMIRLowering(t *testing.T) {
	t.Parallel()
	input := semantic.PackageInput{Files: []semantic.File{
		parseCompilerFile(t, 1, "main.fy", "module main;\nfn i32 main() { return 42; }\n"),
	}}
	analysis, err := compiler.AnalyzePackages(input)
	if err != nil {
		t.Fatalf("AnalyzePackages() error = %v", err)
	}
	if analysis.Semantic.CompletedThrough != semantic.PhaseFlow || analysis.Semantic.FailedPhase != semantic.PhaseNone ||
		len(analysis.Semantic.Diagnostics) != 0 || len(analysis.Semantic.Unsupported) != 0 {
		t.Fatalf("AnalyzePackages() analysis = %+v", analysis)
	}
	result, err := compiler.LowerAnalyzed(analysis)
	if err != nil {
		t.Fatalf("LowerAnalyzed() error = %v", err)
	}
	if !result.Verified || len(result.MIR.Functions) != 1 || result.MIR.Functions[0].Name != "$p1::main::main" {
		t.Fatalf("LowerAnalyzed() result = %+v", result)
	}
}

func TestLowerAnalyzedPublishesVerifiedStateTransactionsAtomically(t *testing.T) {
	t.Parallel()
	analysis, encoded := statefulCompilerAnalysis(t)
	result, err := compiler.LowerAnalyzed(analysis)
	if err != nil {
		t.Fatalf("LowerAnalyzed() error = %v", err)
	}
	if !result.Verified || len(result.MIR.Functions) != 1 || len(result.StateTransactions) != 1 {
		t.Fatalf("LowerAnalyzed() result = %+v", result)
	}
	transaction := result.StateTransactions[0]
	if transaction.Package != 1 || transaction.Function != result.MIR.Functions[0].Symbol {
		t.Fatalf("state transaction owner = %d/%d, MIR symbol = %d", transaction.Package, transaction.Function, result.MIR.Functions[0].Symbol)
	}
	store, err := contractstatemir.NewMemoryStore(transaction.Program.Layout)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := contractstatemir.Execute(store, transaction.Program, contractstate.Limits{
		MaxWrites: 8, MaxWriteBytes: 1024, MaxValueBytes: 64, MaxMapKeyBytes: 64,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Exit != contractstatemir.ExitCommit || len(outcome.Reads) != 1 || !outcome.Reads[0].Found ||
		outcome.Reads[0].Value.TypeID != contractstate.U64.TypeID() || !bytes.Equal(outcome.Reads[0].Value.Encoded, encoded) {
		t.Fatalf("state execution = %+v, want committed read 42", outcome)
	}
}

func TestLowerAnalyzedRejectsStateMIRBeforePublishingExecutableIR(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		effects []hir.Effect
	}{
		{name: "missing storage authority", effects: nil},
		{name: "unrepresented external call authority", effects: []hir.Effect{hir.EffectExternalCall, hir.EffectStorageRead, hir.EffectStorageWrite}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			analysis, _ := statefulCompilerAnalysis(t)
			analysis.Semantic.HIR.Modules[0].Functions[0].Effects = test.effects
			result, err := compiler.LowerAnalyzed(analysis)
			var pipelineError *compiler.Error
			if !errors.As(err, &pipelineError) || pipelineError.Stage != compiler.StageStateMIRLower {
				t.Fatalf("LowerAnalyzed() error = %v", err)
			}
			if result.Verified || len(result.MIR.Functions) != 0 || len(result.StateTransactions) != 0 {
				t.Fatalf("LowerAnalyzed() published partial executable IR = %+v", result)
			}
		})
	}
}

func TestLowerAnalyzedRejectsStateOwnerCallBeforePublishingExecutableIR(t *testing.T) {
	t.Parallel()
	analysis := statefulCallingCompilerAnalysis(t)
	result, err := compiler.LowerAnalyzed(analysis)
	var pipelineError *compiler.Error
	if !errors.As(err, &pipelineError) || pipelineError.Stage != compiler.StageStateMIRLower ||
		!strings.Contains(err.Error(), "contains a call; calls are not admitted") {
		t.Fatalf("LowerAnalyzed() error = %v", err)
	}
	if result.Verified || len(result.MIR.Functions) != 0 || len(result.StateTransactions) != 0 {
		t.Fatalf("LowerAnalyzed() published partial executable IR = %+v", result)
	}
}

func TestLowerAnalyzedRejectsStateOwnerControlFlowBeforePublishingExecutableIR(t *testing.T) {
	t.Parallel()
	analysis := statefulControlFlowCompilerAnalysis(t)
	result, err := compiler.LowerAnalyzed(analysis)
	var pipelineError *compiler.Error
	if !errors.As(err, &pipelineError) || pipelineError.Stage != compiler.StageStateMIRLower ||
		!strings.Contains(err.Error(), "contains dynamic control flow; dynamic control flow is not admitted") {
		t.Fatalf("LowerAnalyzed() error = %v", err)
	}
	if result.Verified || len(result.MIR.Functions) != 0 || len(result.StateTransactions) != 0 {
		t.Fatalf("LowerAnalyzed() published partial executable IR = %+v", result)
	}
}

func TestLowerAnalyzedRejectsUnsafeStateOwnerBeforePublishingExecutableIR(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		analysis func(*testing.T) compiler.Analysis
		contains string
	}{
		{name: "unsafe body", analysis: statefulUnsafeCompilerAnalysis, contains: "unsafe code is not admitted"},
		{name: "raw-pointer parameter", analysis: statefulRawPointerCompilerAnalysis, contains: "raw-pointer types are not admitted"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := compiler.LowerAnalyzed(test.analysis(t))
			var pipelineError *compiler.Error
			if !errors.As(err, &pipelineError) || pipelineError.Stage != compiler.StageStateMIRLower ||
				!strings.Contains(err.Error(), test.contains) {
				t.Fatalf("LowerAnalyzed() error = %v", err)
			}
			if result.Verified || len(result.MIR.Functions) != 0 || len(result.StateTransactions) != 0 {
				t.Fatalf("LowerAnalyzed() published partial executable IR = %+v", result)
			}
		})
	}
}

func TestLowerAnalyzedRejectsImplicitStateOwnersBeforePublishingExecutableIR(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		analysis func(*testing.T) compiler.Analysis
		contains string
	}{
		{name: "receiver-bound method", analysis: statefulReceiverMethodCompilerAnalysis, contains: "receiver-bound owners are not admitted"},
		{name: "drop glue", analysis: statefulDropGlueCompilerAnalysis, contains: "drop-glue owners are not admitted"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := compiler.LowerAnalyzed(test.analysis(t))
			var pipelineError *compiler.Error
			if !errors.As(err, &pipelineError) || pipelineError.Stage != compiler.StageStateMIRLower ||
				!strings.Contains(err.Error(), test.contains) {
				t.Fatalf("LowerAnalyzed() error = %v", err)
			}
			if result.Verified || len(result.MIR.Functions) != 0 || len(result.StateTransactions) != 0 {
				t.Fatalf("LowerAnalyzed() published partial executable IR = %+v", result)
			}
		})
	}
}

func TestLowerAnalyzedPreservesSourceDiagnosticsWithoutMIR(t *testing.T) {
	t.Parallel()
	analysis, err := compiler.AnalyzePackages(semantic.PackageInput{Files: []semantic.File{
		parseCompilerFile(t, 1, "main.fy", "module main;\nfn i32 main() { return missing; }\n"),
	}})
	if err != nil {
		t.Fatalf("AnalyzePackages() error = %v", err)
	}
	result, err := compiler.LowerAnalyzed(analysis)
	if err != nil {
		t.Fatalf("LowerAnalyzed() error = %v", err)
	}
	if result.Verified || result.Semantic.FailedPhase != semantic.PhaseResolve || len(result.Semantic.Diagnostics) != 1 || len(result.MIR.Functions) != 0 {
		t.Fatalf("LowerAnalyzed() result = %+v", result)
	}
}

func statefulCompilerAnalysis(t *testing.T) (compiler.Analysis, []byte) {
	t.Helper()
	analysis, err := compiler.AnalyzePackages(semantic.PackageInput{Files: []semantic.File{
		parseCompilerFile(t, 1, "main.fy", "module main;\nfn i32 main() { return 42; }\n"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	return attachStatefulCompilerAnalysis(t, analysis, "test.compiler-result")
}

func statefulCallingCompilerAnalysis(t *testing.T) compiler.Analysis {
	t.Helper()
	analysis, err := compiler.AnalyzePackages(semantic.PackageInput{Files: []semantic.File{
		parseCompilerFile(t, 1, "main.fy", "module main;\nfn i32 helper() { return 42; }\nfn i32 main() { return helper(); }\n"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	analysis, _ = attachStatefulCompilerAnalysis(t, analysis, "test.compiler-result-call")
	return analysis
}

func statefulControlFlowCompilerAnalysis(t *testing.T) compiler.Analysis {
	t.Helper()
	analysis, err := compiler.AnalyzePackages(semantic.PackageInput{Files: []semantic.File{
		parseCompilerFile(t, 1, "main.fy", "module main;\nfn i32 main() { return if (true) { 42 } else { 0 }; }\n"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	analysis, _ = attachStatefulCompilerAnalysis(t, analysis, "test.compiler-result-control-flow")
	return analysis
}

func statefulUnsafeCompilerAnalysis(t *testing.T) compiler.Analysis {
	t.Helper()
	analysis, err := compiler.AnalyzePackages(semantic.PackageInput{Files: []semantic.File{
		parseCompilerFile(t, 1, "main.fy", "module main;\nfn i32 main() { let ptr u8 pointer = unsafe { c\"fairy\" as ptr u8 }; return 42; }\n"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	analysis, _ = attachStatefulCompilerAnalysis(t, analysis, "test.compiler-result-unsafe")
	return analysis
}

func statefulRawPointerCompilerAnalysis(t *testing.T) compiler.Analysis {
	t.Helper()
	analysis, err := compiler.AnalyzePackages(semantic.PackageInput{Files: []semantic.File{
		parseCompilerFile(t, 1, "main.fy", "module main;\nfn i32 main(ptr u8 pointer) { return 42; }\n"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	analysis, _ = attachStatefulCompilerAnalysis(t, analysis, "test.compiler-result-raw-pointer")
	return analysis
}

func statefulReceiverMethodCompilerAnalysis(t *testing.T) compiler.Analysis {
	t.Helper()
	analysis, err := compiler.AnalyzePackages(semantic.PackageInput{Files: []semantic.File{
		parseCompilerFile(t, 1, "main.fy", "module main;\nstruct Counter { i32 value; }\nfn i32 Counter::read(ref Counter this) { return 42; }\nfn i32 main() { return 42; }\n"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	analysis, _ = attachStatefulCompilerFunction(t, analysis, "test.compiler-result-receiver", func(function *hir.Function) bool {
		return function.Name == "Counter::read"
	})
	return analysis
}

func statefulDropGlueCompilerAnalysis(t *testing.T) compiler.Analysis {
	t.Helper()
	analysis, err := compiler.AnalyzePackages(semantic.PackageInput{Files: []semantic.File{
		parseCompilerFile(t, 1, "main.fy", "module main;\nresource Handle { i32 value; cleanup { let i32 marker = this.value; } }\nfn i32 main() { return 42; }\n"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	analysis, _ = attachStatefulCompilerFunction(t, analysis, "test.compiler-result-drop-glue", func(function *hir.Function) bool {
		return function.DropGlueFor != ""
	})
	analysis.Semantic.HIR.Modules[0].Resources[0].DropEffects = hir.CanonicalEffects([]hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite})
	return analysis
}

func attachStatefulCompilerAnalysis(t *testing.T, analysis compiler.Analysis, contractUID string) (compiler.Analysis, []byte) {
	return attachStatefulCompilerFunction(t, analysis, contractUID, func(function *hir.Function) bool {
		return function.Name == "main"
	})
}

func attachStatefulCompilerFunction(t *testing.T, analysis compiler.Analysis, contractUID string, selectFunction func(*hir.Function) bool) (compiler.Analysis, []byte) {
	t.Helper()
	var function *hir.Function
	for moduleIndex := range analysis.Semantic.HIR.Modules {
		for functionIndex := range analysis.Semantic.HIR.Modules[moduleIndex].Functions {
			candidate := &analysis.Semantic.HIR.Modules[moduleIndex].Functions[functionIndex]
			if selectFunction(candidate) {
				function = candidate
				break
			}
		}
	}
	if function == nil {
		t.Fatal("analyzed package has no selected state function")
	}
	function.Effects = hir.CanonicalEffects([]hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite})
	encoded, err := contractstate.U64.Encode(42)
	if err != nil {
		t.Fatal(err)
	}
	analysis.Semantic.HIR.StateLayouts = []hir.StateLayout{{
		Package: 1, ContractUID: contractUID,
		Fields: []hir.StateField{{Name: "value", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID()}},
	}}
	analysis.Semantic.HIR.StateTransactions = []hir.StateTransaction{{
		Package: 1, Function: function.Symbol, Exit: hir.StateTransactionCommit,
		Accesses: []hir.StateAccess{
			{Kind: hir.StateAccessCellWrite, Field: "value", Value: hir.StateValue{TypeID: contractstate.U64.TypeID(), Encoded: encoded}},
			{Kind: hir.StateAccessCellRead, Field: "value"},
		},
	}}
	return analysis, encoded
}

func TestCheckPackagesVerifiesIsolatedModulePaths(t *testing.T) {
	t.Parallel()
	result, err := compiler.CheckPackages(semantic.PackageInput{
		Files: []semantic.File{
			parseCompilerFile(t, 1, "app_a.fy", "module app_a;\nimport util as u;\nfn i32 main() { return u::value(); }\n"),
			parseCompilerFile(t, 2, "app_b.fy", "module app_b;\nimport util as u;\nfn i32 main() { return u::value(); }\n"),
			parseCompilerFile(t, 3, "util.fy", "module util;\npub fn i32 value() { return 1; }\n"),
			parseCompilerFile(t, 4, "util.fy", "module util;\npub fn i32 value() { return 2; }\n"),
		},
		Imports: []semantic.PackageImport{
			{FromPackage: 1, ImportPath: "util", ToPackage: 3},
			{FromPackage: 2, ImportPath: "util", ToPackage: 4},
		},
	})
	if err != nil {
		t.Fatalf("CheckPackages() error = %v", err)
	}
	if !result.Verified || result.Semantic.CompletedThrough != semantic.PhaseFlow {
		t.Fatalf("CheckPackages() result = %+v", result)
	}
	names := make([]string, 0, len(result.MIR.Functions))
	for _, function := range result.MIR.Functions {
		names = append(names, function.Name)
	}
	want := []string{"$p1::app_a::main", "$p2::app_b::main", "$p3::util::value", "$p4::util::value"}
	if !slices.Equal(names, want) {
		t.Fatalf("CheckPackages() MIR function names = %v, want %v", names, want)
	}
}

func parseCompilerFile(t *testing.T, packageID hir.PackageID, path, text string) semantic.File {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(text)), source.DefaultLimits())
	if err != nil {
		t.Fatalf("AnalyzeSyntax(%s) error = %v", path, err)
	}
	if len(parsed.Tree.Diagnostics) != 0 || parsed.AST == nil {
		t.Fatalf("AnalyzeSyntax(%s) diagnostics = %+v, AST = %+v", path, parsed.Tree.Diagnostics, parsed.AST)
	}
	return semantic.File{Package: packageID, Path: path, Module: parsed.AST}
}

func parseCompilerFileFromPath(t *testing.T, path string) semantic.File {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	logicalPath := filepath.ToSlash(filepath.Join("examples", "owned_text", filepath.Base(path)))
	return parseCompilerFile(t, hir.DefaultPackage, logicalPath, string(text))
}
