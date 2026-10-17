package semantic_test

import (
	"encoding/json"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func parseTestDeclarationFile(t *testing.T, text string) semantic.File {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/check.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("parse: %v %+v", err, parsed.Tree.Diagnostics)
	}
	return semantic.File{Path: "tests/check.fy", Module: parsed.AST, Test: true}
}

func TestTestAttributeRequiresExplicitSourceAdmission(t *testing.T) {
	file := parseTestDeclarationFile(t, `module tests::check; @test @effects(console) fn void works() { assert(true); }`)
	before, _ := json.Marshal(file.Module)
	for attempt := 0; attempt < 2; attempt++ {
		inspection := semantic.InspectTestDeclarations(file.Module)
		if len(inspection.Functions) != 1 || len(inspection.Diagnostics) != 0 {
			t.Fatalf("inspection: %+v", inspection)
		}
		result := semantic.Analyze([]semantic.File{file})
		if result.CompletedThrough != semantic.PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
			t.Fatalf("test analysis: %+v", result)
		}
	}
	after, _ := json.Marshal(file.Module)
	if string(before) != string(after) {
		t.Fatal("test declaration analysis mutated source AST")
	}
	file.Test = false
	production := semantic.Analyze([]semantic.File{file})
	if len(production.Unsupported) != 1 || production.Unsupported[0].Feature != "function attribute @test" {
		t.Fatalf("path inferred test admission: %+v", production)
	}
}

func TestTestAttributeDeclarationRejectionsAreSemantic(t *testing.T) {
	for name, text := range map[string]string{
		"arguments":  `@test("x") fn void invalid() {}`,
		"duplicates": `@test @test fn void invalid() {}`,
		"public":     `@test pub fn void invalid() {}`,
		"unsafe":     `@test unsafe fn void invalid() {}`,
		"parameter":  `@test fn void invalid(i32 x) {}`,
		"result":     `@test fn i32 invalid() { return 0; }`,
		"template":   `@test fn void invalid<T>() {}`,
		"structure":  `@test struct Invalid { i32 value; }`,
		"enum":       `@test enum Invalid { empty; }`,
		"resource":   `@test resource Invalid { i32 value; cleanup {} }`,
	} {
		t.Run(name, func(t *testing.T) {
			file := parseTestDeclarationFile(t, "module tests::check; "+text)
			before, _ := json.Marshal(file.Module)
			for attempt := 0; attempt < 2; attempt++ {
				inspection := semantic.InspectTestDeclarations(file.Module)
				if len(inspection.Functions) != 0 || len(inspection.Diagnostics) != 1 || inspection.Diagnostics[0].Code != "E9001" {
					t.Fatalf("inspection: %+v", inspection)
				}
				result := semantic.Analyze([]semantic.File{file})
				if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E9001" || len(result.Unsupported) != 0 {
					t.Fatalf("semantic rejection: %+v", result)
				}
			}
			after, _ := json.Marshal(file.Module)
			if string(before) != string(after) {
				t.Fatal("rejected declaration mutated source AST")
			}
		})
	}
}
