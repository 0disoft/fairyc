package compiler

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestRestrictGenericSignaturesPreservesReportExhaustion(t *testing.T) {
	checked := semantic.Result{
		FailedPhase: semantic.PhaseType,
		Diagnostics: []semantic.Diagnostic{{Code: semantic.SemanticReportExhaustedCode,
			Primary: hir.Span{Source: "main.fy", StartByte: 10, EndByte: 11}}},
	}
	module := &ast.Module{Functions: []ast.Function{{
		Name: "generated",
		ResultType: ast.TypeReference{Kind: ast.TypeKindNamed, Path: "Result", Arguments: []ast.TypeReference{
			{Kind: ast.TypeKindNamed, Path: "u64"},
			{Kind: ast.TypeKindNamed, Path: "i32"},
		}},
	}}}
	expansion := genericsource.PackageExpansion{Origins: []genericsource.SpecializationOrigin{{GeneratedName: "generated"}}}
	got := restrictGenericSignatures(checked, []semantic.File{{Path: "main.fy", Module: module}}, expansion)
	if !reflect.DeepEqual(got, checked) {
		t.Fatalf("generic admission replaced bounded diagnostics: %+v", got)
	}
}

func TestRestrictGenericSignaturesBoundsTransportUnsupportedAtCollection(t *testing.T) {
	// ADR 0555 admits Result<u64,i32>; an i64 error remains unsupported.
	const generated = semantic.DefaultSemanticReportLimit + 64
	functions := make([]ast.Function, 0, generated)
	origins := make([]genericsource.SpecializationOrigin, 0, generated)
	for index := range generated {
		name := fmt.Sprintf("generated%d", index)
		functions = append(functions, ast.Function{
			Name: name,
			ResultType: ast.TypeReference{Kind: ast.TypeKindNamed, Path: "Result", Text: "Result<u64,i64>",
				Span: diagnosticSpan(index), Arguments: []ast.TypeReference{{
					Kind: ast.TypeKindNamed, Path: "u64",
				}, {Kind: ast.TypeKindNamed, Path: "i64"}}},
		})
		origins = append(origins, genericsource.SpecializationOrigin{GeneratedName: name})
	}

	got := restrictGenericSignatures(
		semantic.Result{CompletedThrough: semantic.PhaseFlow},
		[]semantic.File{{Path: "main.fy", Module: &ast.Module{Functions: functions}}},
		genericsource.PackageExpansion{Origins: origins},
	)
	if len(got.Unsupported) != semantic.DefaultSemanticReportLimit || cap(got.Unsupported) != semantic.DefaultSemanticReportLimit {
		t.Fatalf("unsupported len=%d cap=%d", len(got.Unsupported), cap(got.Unsupported))
	}
	if len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != semantic.SemanticReportExhaustedCode || got.FailedPhase != semantic.PhaseType {
		t.Fatalf("terminal marker = %+v", got.Diagnostics)
	}
	if got.Diagnostics[0].Primary.StartByte != semantic.DefaultSemanticReportLimit {
		t.Fatalf("marker primary = %+v", got.Diagnostics[0].Primary)
	}
}

func diagnosticSpan(start int) diagnostic.Span {
	return diagnostic.Span{StartByte: start, EndByte: start + 1}
}

func TestRestrictGenericSignaturesReportBoundaryOrderAndOwnership(t *testing.T) {
	limit := semantic.DefaultSemanticReportLimit
	for _, test := range []struct {
		name       string
		count      int
		duplicates bool
		overflow   bool
	}{
		{name: "below limit", count: limit - 1},
		{name: "exact limit", count: limit},
		{name: "first excess parameter", count: limit + 1, overflow: true},
		{name: "duplicates at limit", count: limit, duplicates: true},
		{name: "duplicates before excess parameter", count: limit + 1, duplicates: true, overflow: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			files, expansion := genericRestrictionBoundaryInput(test.count, test.duplicates)
			beforeFiles, beforeExpansion := genericRestrictionBoundaryInput(test.count, test.duplicates)
			checked := semantic.Result{CompletedThrough: semantic.PhaseFlow,
				Diagnostics: []semantic.Diagnostic{{Code: "existing"}},
				Unsupported: []semantic.Unsupported{{Feature: "existing"}}}
			beforeChecked := semantic.Result{CompletedThrough: semantic.PhaseFlow,
				Diagnostics: []semantic.Diagnostic{{Code: "existing"}},
				Unsupported: []semantic.Unsupported{{Feature: "existing"}}}
			got := restrictGenericSignatures(checked, files, expansion)
			if !reflect.DeepEqual(files, beforeFiles) || !reflect.DeepEqual(expansion, beforeExpansion) || !reflect.DeepEqual(checked, beforeChecked) {
				t.Fatal("admission mutated source, provenance, or prior analysis")
			}
			wantCount := min(test.count, limit)
			if len(got.Unsupported) != wantCount || cap(got.Unsupported) != limit || got.CompletedThrough != semantic.PhaseResolve {
				t.Fatalf("unexpected bounded result: len=%d cap=%d phase=%s", len(got.Unsupported), cap(got.Unsupported), got.CompletedThrough)
			}
			for i, value := range got.Unsupported {
				wantStart := i
				if i == wantCount-1 {
					wantStart = limit + 10 // The return type is visited first, but sorted last.
				}
				if value.Feature != "type ref u64" || value.Span != (hir.Span{Package: 7, Source: "main.fy", StartByte: wantStart, EndByte: wantStart + 1}) {
					t.Fatalf("unsupported[%d] = %+v, want start %d", i, value, wantStart)
				}
			}
			if !test.overflow {
				if got.FailedPhase != semantic.PhaseNone || len(got.Diagnostics) != 0 {
					t.Fatalf("premature exhaustion: %+v", got.Diagnostics)
				}
				return
			}
			wantDiagnostic := semantic.Diagnostic{Code: semantic.SemanticReportExhaustedCode,
				Primary: hir.Span{Package: 7, Source: "main.fy", StartByte: limit - 1, EndByte: limit}}
			if got.FailedPhase != semantic.PhaseType || !reflect.DeepEqual(got.Diagnostics, []semantic.Diagnostic{wantDiagnostic}) {
				t.Fatalf("changed first excess diagnostic: %+v", got.Diagnostics)
			}
		})
	}
}

func genericRestrictionBoundaryInput(count int, duplicates bool) ([]semantic.File, genericsource.PackageExpansion) {
	badType := func(start int) ast.TypeReference {
		return ast.TypeReference{Kind: ast.TypeKindReference, Text: " ref u64 ", Span: diagnosticSpan(start),
			Element: &ast.TypeReference{Kind: ast.TypeKindNamed, Path: "u64"}}
	}
	function := ast.Function{Name: "generated", ResultType: badType(semantic.DefaultSemanticReportLimit + 10)}
	for i := range count - 1 {
		if duplicates {
			function.Parameters = append(function.Parameters, ast.Parameter{Type: function.ResultType})
		}
		function.Parameters = append(function.Parameters, ast.Parameter{Type: badType(i)})
	}
	if duplicates {
		function.Parameters = append(function.Parameters, ast.Parameter{Type: function.ResultType})
	}
	functions := []ast.Function{{Name: "ordinary", ResultType: badType(-1)}, function}
	files := []semantic.File{{Package: 7, Path: "main.fy", Module: &ast.Module{Functions: functions}}}
	if count > semantic.DefaultSemanticReportLimit {
		// Exhaustion must stop before visiting another function or file.
		files[0].Module.Functions = append(files[0].Module.Functions, ast.Function{Name: "generated", ResultType: badType(-2)})
		files = append(files, semantic.File{Path: "must-not-be-visited.fy"})
	}
	return files, genericsource.PackageExpansion{Origins: []genericsource.SpecializationOrigin{{GeneratedName: "generated"}}}
}

func TestRestrictGenericSignaturesReturnsPriorAnalysisBeforeSourceTraversal(t *testing.T) {
	for _, test := range []struct {
		name    string
		checked semantic.Result
		origins []genericsource.SpecializationOrigin
	}{
		{name: "resolution failed", checked: semantic.Result{FailedPhase: semantic.PhaseResolve}, origins: []genericsource.SpecializationOrigin{{GeneratedName: "generated"}}},
		{name: "no provenance", checked: semantic.Result{CompletedThrough: semantic.PhaseFlow}},
		{name: "report exhausted", checked: semantic.Result{Diagnostics: []semantic.Diagnostic{{Code: semantic.SemanticReportExhaustedCode}}}, origins: []genericsource.SpecializationOrigin{{GeneratedName: "generated"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := restrictGenericSignatures(test.checked, []semantic.File{{}}, genericsource.PackageExpansion{Origins: test.origins})
			if !reflect.DeepEqual(got, test.checked) {
				t.Fatalf("replaced prior analysis: %+v", got)
			}
		})
	}
}
