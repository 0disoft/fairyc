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

// Transport-only C record scalars do not expand the executable generic subset.
// Use authenticated expansion origins, never a generated-name prefix. Ordinary
// functions keep their transport types, and unused type arguments remain valid.
func restrictGenericSignatures(checked semantic.Result, files []semantic.File, expansion genericsource.PackageExpansion) semantic.Result {
	if checked.FailedPhase == semantic.PhaseResolve || len(expansion.Origins) == 0 {
		return checked
	}
	functions := make(map[string]bool, len(expansion.Origins))
	for _, origin := range expansion.Origins {
		functions[origin.GeneratedName] = true
	}
	var unsupported []semantic.Unsupported
	for _, file := range files {
		for _, function := range file.Module.Functions {
			if !functions[function.Name] {
				continue
			}
			check := func(reference ast.TypeReference) {
				if unadmittedGenericScalar(reference) {
					unsupported = append(unsupported, semantic.Unsupported{
						Feature: "type " + strings.TrimSpace(reference.Text),
						Span: hir.Span{Package: file.Package, Source: file.Path,
							StartByte: reference.Span.StartByte, EndByte: reference.Span.EndByte},
					})
				}
			}
			check(function.ResultType)
			for _, parameter := range function.Parameters {
				check(parameter.Type)
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
	return semantic.Result{CompletedThrough: semantic.PhaseResolve, Unsupported: slices.Compact(unsupported)}
}

func unadmittedGenericScalar(reference ast.TypeReference) bool {
	if reference.Kind == ast.TypeKindNamed {
		switch reference.Path {
		case "i8", "i16", "isize", "u16", "u64", "f32", "f64":
			return true
		}
	}
	if reference.Element != nil && unadmittedGenericScalar(*reference.Element) ||
		reference.Result != nil && unadmittedGenericScalar(*reference.Result) {
		return true
	}
	for _, argument := range reference.Arguments {
		if unadmittedGenericScalar(argument) {
			return true
		}
	}
	for _, parameter := range reference.Parameters {
		if unadmittedGenericScalar(parameter) {
			return true
		}
	}
	return false
}
