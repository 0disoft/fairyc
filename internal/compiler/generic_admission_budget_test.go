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
	const generated = semantic.DefaultSemanticReportLimit + 64
	functions := make([]ast.Function, 0, generated)
	origins := make([]genericsource.SpecializationOrigin, 0, generated)
	for index := range generated {
		name := fmt.Sprintf("generated%d", index)
		functions = append(functions, ast.Function{
			Name: name,
			ResultType: ast.TypeReference{Kind: ast.TypeKindNamed, Path: "Result",
				Span: diagnosticSpan(index), Arguments: []ast.TypeReference{{
					Kind: ast.TypeKindNamed, Path: "u64",
				}, {Kind: ast.TypeKindNamed, Path: "i32"}}},
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
