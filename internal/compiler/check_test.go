package compiler_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
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
