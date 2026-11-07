package compiler

import (
	"cmp"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

// Direct ordinary scalar transport now shares the generic executable subset.
// Only flat Option<S> and Result<S,u32> transport share that extension.
// Use authenticated expansion origins, never a generated-name prefix. Ordinary
// functions keep their transport types, and unused type arguments remain valid.
func restrictGenericSignatures(checked semantic.Result, files []semantic.File, expansion genericsource.PackageExpansion) semantic.Result {
	if checked.FailedPhase == semantic.PhaseResolve || len(expansion.Origins) == 0 ||
		slices.ContainsFunc(checked.Diagnostics, func(value semantic.Diagnostic) bool {
			return value.Code == semantic.SemanticReportExhaustedCode
		}) {
		return checked
	}
	functions := make(map[string]bool, len(expansion.Origins))
	for _, origin := range expansion.Origins {
		functions[origin.GeneratedName] = true
	}
	seen := make(map[genericUnsupportedKey]struct{}, semantic.DefaultSemanticReportLimit)
	unsupported := make([]semantic.Unsupported, 0, semantic.DefaultSemanticReportLimit)
	var exhausted bool
	var exhaustedAt hir.Span
	for _, file := range files {
		if exhausted {
			break
		}
		for _, function := range file.Module.Functions {
			if exhausted {
				break
			}
			if !functions[function.Name] {
				continue
			}
			check := func(reference ast.TypeReference) bool {
				if unadmittedGenericScalar(reference) {
					value := semantic.Unsupported{
						Feature: "type " + strings.TrimSpace(reference.Text),
						Span: hir.Span{Package: file.Package, Source: file.Path,
							StartByte: reference.Span.StartByte, EndByte: reference.Span.EndByte},
					}
					key := genericUnsupportedKey{span: value.Span, feature: value.Feature}
					if _, duplicate := seen[key]; duplicate {
						return true
					}
					if len(unsupported) == semantic.DefaultSemanticReportLimit {
						exhausted = true
						exhaustedAt = value.Span
						return false
					}
					seen[key] = struct{}{}
					unsupported = append(unsupported, value)
				}
				return true
			}
			if !check(function.ResultType) {
				break
			}
			for _, parameter := range function.Parameters {
				if !check(parameter.Type) {
					break
				}
			}
		}
	}
	if len(unsupported) == 0 {
		return checked
	}
	slices.SortFunc(unsupported, func(a, b semantic.Unsupported) int {
		return cmp.Or(cmp.Compare(a.Span.Package, b.Span.Package), cmp.Compare(a.Span.Source, b.Span.Source),
			cmp.Compare(a.Span.StartByte, b.Span.StartByte),
			cmp.Compare(a.Span.EndByte, b.Span.EndByte), cmp.Compare(a.Feature, b.Feature))
	})
	result := semantic.Result{CompletedThrough: semantic.PhaseResolve, Unsupported: slices.Compact(unsupported)}
	if exhausted {
		result.FailedPhase = semantic.PhaseType
		result.Diagnostics = []semantic.Diagnostic{{Code: semantic.SemanticReportExhaustedCode, Primary: exhaustedAt}}
	}
	return result
}

type genericUnsupportedKey struct {
	span    hir.Span
	feature string
}

func unadmittedGenericScalar(reference ast.TypeReference) bool {
	if reference.Kind == ast.TypeKindNamed && len(reference.Arguments) == 0 {
		return false
	}
	if reference.Kind == ast.TypeKindNamed && reference.Path == "Option" && len(reference.Arguments) == 1 {
		payload := reference.Arguments[0]
		if payload.Kind == ast.TypeKindNamed && len(payload.Arguments) == 0 && containsTransportScalar(payload) {
			return false
		}
	}
	if reference.Kind == ast.TypeKindNamed && reference.Path == "Result" && len(reference.Arguments) == 2 {
		success, failure := reference.Arguments[0], reference.Arguments[1]
		if success.Kind == ast.TypeKindNamed && len(success.Arguments) == 0 && containsTransportScalar(success) &&
			failure.Kind == ast.TypeKindNamed && failure.Path == "u32" && len(failure.Arguments) == 0 {
			return false
		}
	}
	return containsTransportScalar(reference)
}

func containsTransportScalar(reference ast.TypeReference) bool {
	if reference.Kind == ast.TypeKindNamed {
		switch reference.Path {
		case "i8", "i16", "isize", "u16", "u64", "f32", "f64":
			return true
		}
	}
	if reference.Element != nil && containsTransportScalar(*reference.Element) ||
		reference.Result != nil && containsTransportScalar(*reference.Result) {
		return true
	}
	for _, argument := range reference.Arguments {
		if containsTransportScalar(argument) {
			return true
		}
	}
	for _, parameter := range reference.Parameters {
		if containsTransportScalar(parameter) {
			return true
		}
	}
	return false
}
