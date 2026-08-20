package semantic_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestAnalyzePackagesResolvesEqualModulePathsByPackage(t *testing.T) {
	t.Parallel()
	input := semantic.PackageInput{
		Files: []semantic.File{
			parsePackageFile(t, 1, "app_a.fy", "module app_a;\nimport util as u;\nfn i32 main() { return u::value(); }\n"),
			parsePackageFile(t, 2, "app_b.fy", "module app_b;\nimport util as u;\nfn i32 main() { return u::value(); }\n"),
			parsePackageFile(t, 3, "util.fy", "module util;\npub struct Token { i32 value; }\npub fn i32 value() { return 1; }\n"),
			parsePackageFile(t, 4, "util.fy", "module util;\npub struct Token { i32 value; }\npub fn i32 value() { return 2; }\n"),
		},
		Imports: []semantic.PackageImport{
			{FromPackage: 1, ImportPath: "util", ToPackage: 3},
			{FromPackage: 2, ImportPath: "util", ToPackage: 4},
		},
	}

	result, err := semantic.AnalyzePackages(input)
	if err != nil {
		t.Fatalf("AnalyzePackages() error = %v", err)
	}
	if result.CompletedThrough != semantic.PhaseFlow || result.FailedPhase != semantic.PhaseNone || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("AnalyzePackages() result = %+v", result)
	}
	if len(result.HIR.Modules) != 4 {
		t.Fatalf("AnalyzePackages() HIR modules = %d, want 4", len(result.HIR.Modules))
	}

	resolvedPackages := make(map[hir.PackageID]int)
	for _, resolution := range result.Resolutions {
		if resolution.Target.Module == "util" && resolution.Target.Name == "value" {
			resolvedPackages[resolution.Target.Package]++
			if resolution.Use.Package == hir.DefaultPackage || resolution.Target.Declaration.Package != resolution.Target.Package ||
				strings.Contains(resolution.Use.Source, "$p") || strings.Contains(resolution.Target.Declaration.Source, "$p") {
				t.Fatalf("AnalyzePackages() leaked or lost package-aware source identity: %+v", resolution)
			}
		}
	}
	if resolvedPackages[3] != 1 || resolvedPackages[4] != 1 || len(resolvedPackages) != 2 {
		t.Fatalf("AnalyzePackages() util resolutions = %v", resolvedPackages)
	}

	sourcePaths := make(map[hir.PackageID]string)
	for _, module := range result.HIR.Modules {
		sourcePaths[module.Package] = module.SourcePath
	}
	if sourcePaths[3] != "util.fy" || sourcePaths[4] != "util.fy" {
		t.Fatalf("AnalyzePackages() source paths = %v", sourcePaths)
	}
	types := make(map[hir.Type]struct{})
	for _, module := range result.HIR.Modules {
		for _, structure := range module.Structs {
			types[structure.Type] = struct{}{}
		}
	}
	if _, found := types[hir.Type("$p3::util::Token")]; !found {
		t.Fatalf("AnalyzePackages() struct types = %v", types)
	}
	if _, found := types[hir.Type("$p4::util::Token")]; !found {
		t.Fatalf("AnalyzePackages() struct types = %v", types)
	}
}

func TestAnalyzePackagesKeepsDiagnosticPathsUnqualified(t *testing.T) {
	t.Parallel()
	result, err := semantic.AnalyzePackages(semantic.PackageInput{
		Files: []semantic.File{
			parsePackageFile(t, 1, "app.fy", "module app;\nimport util as u;\nfn i32 main() { return u::private_value(); }\n"),
			parsePackageFile(t, 2, "util.fy", "module util;\nfn i32 private_value() { return 1; }\n"),
		},
		Imports: []semantic.PackageImport{{FromPackage: 1, ImportPath: "util", ToPackage: 2}},
	})
	if err != nil {
		t.Fatalf("AnalyzePackages() error = %v", err)
	}
	if result.FailedPhase != semantic.PhaseResolve || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E0103" || len(result.Diagnostics[0].Related) != 1 {
		t.Fatalf("AnalyzePackages() result = %+v", result)
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Primary.Package != 1 || diagnostic.Primary.Source != "app.fy" ||
		diagnostic.Related[0].Span.Package != 2 || diagnostic.Related[0].Span.Source != "util.fy" ||
		strings.Contains(diagnostic.Primary.Source, "$p") || strings.Contains(diagnostic.Related[0].Span.Source, "$p") {
		t.Fatalf("AnalyzePackages() diagnostic paths = %+v", diagnostic)
	}
}

func TestAnalyzePackagesKeepsResourceAbilitiesBehindLanguageCutover(t *testing.T) {
	t.Parallel()
	result, err := semantic.AnalyzePackages(semantic.PackageInput{Files: []semantic.File{
		parsePackageFile(t, 1, "asset.fy", `module asset;
@abilities(store, key)
resource Coin { i32 amount; cleanup {} }
fn void keep(Coin coin) {}`),
	}})
	if err != nil {
		t.Fatalf("AnalyzePackages() error = %v", err)
	}
	if result.CompletedThrough != semantic.PhaseResolve || result.FailedPhase != semantic.PhaseNone ||
		len(result.Diagnostics) != 0 || len(result.Unsupported) != 1 ||
		result.Unsupported[0].Feature != "resource attribute @abilities" {
		t.Fatalf("AnalyzePackages() result = %+v", result)
	}
}

func TestAnalyzePackagesRejectsInvalidImportBindings(t *testing.T) {
	t.Parallel()
	files := []semantic.File{
		parsePackageFile(t, 1, "app.fy", "module app;\nimport util as u;\nfn i32 main() { return u::value(); }\n"),
		parsePackageFile(t, 2, "util.fy", "module util;\npub fn i32 value() { return 1; }\n"),
	}
	tests := []struct {
		name     string
		imports  []semantic.PackageImport
		contains string
	}{
		{
			name: "duplicate",
			imports: []semantic.PackageImport{
				{FromPackage: 1, ImportPath: "util", ToPackage: 2},
				{FromPackage: 1, ImportPath: "util", ToPackage: 2},
			},
			contains: "bound more than once",
		},
		{
			name:     "missing target",
			imports:  []semantic.PackageImport{{FromPackage: 1, ImportPath: "util", ToPackage: 9}},
			contains: "target 9 is not present",
		},
		{
			name:     "unused",
			imports:  []semantic.PackageImport{{FromPackage: 1, ImportPath: "other", ToPackage: 2}},
			contains: "has no source use",
		},
		{
			name:     "missing binding",
			imports:  nil,
			contains: "has no resolved binding",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := semantic.AnalyzePackages(semantic.PackageInput{Files: files, Imports: test.imports})
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("AnalyzePackages() error = %v, want containing %q", err, test.contains)
			}
		})
	}
}

func parsePackageFile(t *testing.T, packageID hir.PackageID, path, text string) semantic.File {
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
