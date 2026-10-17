package mir

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestLowerPreservesBuildLocalPackageIdentity(t *testing.T) {
	t.Parallel()
	input := semantic.PackageInput{
		Files: []semantic.File{
			parseMIRPackageFile(t, 1, "app_a.fy", "module app_a;\nimport util as u;\nfn i32 main() { return u::value(); }\n"),
			parseMIRPackageFile(t, 2, "app_b.fy", "module app_b;\nimport util as u;\nfn i32 main() { return u::value(); }\n"),
			parseMIRPackageFile(t, 3, "util.fy", "module util;\npub struct Token { i32 value; }\npub fn i32 value() { return 1; }\n"),
			parseMIRPackageFile(t, 4, "util.fy", "module util;\npub struct Token { i32 value; }\npub fn i32 value() { return 2; }\n"),
		},
		Imports: []semantic.PackageImport{
			{FromPackage: 1, ImportPath: "util", ToPackage: 3},
			{FromPackage: 2, ImportPath: "util", ToPackage: 4},
		},
	}
	semanticResult, err := semantic.AnalyzePackages(input)
	if err != nil || semanticResult.CompletedThrough != semantic.PhaseFlow || len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		t.Fatalf("AnalyzePackages() result = %+v, error = %v", semanticResult, err)
	}

	program, err := Lower(semanticResult.HIR)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	names := make([]string, 0, len(program.Functions))
	for _, function := range program.Functions {
		names = append(names, function.Name)
		if function.Span.Package == hir.DefaultPackage || strings.Contains(function.Span.Source, "$p") {
			t.Fatalf("Lower() function span leaked or lost package-aware source identity: %+v", function.Span)
		}
	}
	want := []string{"$p1::app_a::main", "$p2::app_b::main", "$p3::util::value", "$p4::util::value"}
	if !slices.Equal(names, want) {
		t.Fatalf("Lower() function names = %v, want %v", names, want)
	}
	structTypes := make([]hir.Type, 0, len(program.StructTypes))
	for _, structure := range program.StructTypes {
		structTypes = append(structTypes, structure.Type)
	}
	wantStructTypes := []hir.Type{hir.ParseType("$p3::util::Token"), hir.ParseType("$p4::util::Token")}
	if !slices.Equal(structTypes, wantStructTypes) {
		t.Fatalf("Lower() struct types = %v, want %v", structTypes, wantStructTypes)
	}
}

func parseMIRPackageFile(t *testing.T, packageID hir.PackageID, path, text string) semantic.File {
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
