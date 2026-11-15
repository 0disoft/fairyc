package compiler

import (
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

type genericUnsupportedKey struct {
	span    hir.Span
	feature string
}

// Only this collector owns mutable report storage; source and analysis inputs
// remain read-only. The shared semantic report limit bounds both collections.
type genericSignatureRestrictionCollector struct {
	seen   map[genericUnsupportedKey]struct{}
	values []semantic.Unsupported
}

func collectGenericSignatureRestrictions(files []semantic.File, origins []genericsource.SpecializationOrigin) ([]semantic.Unsupported, *hir.Span) {
	functions := make(map[string]bool, len(origins))
	for _, origin := range origins {
		functions[origin.GeneratedName] = true
	}
	collector := genericSignatureRestrictionCollector{
		seen:   make(map[genericUnsupportedKey]struct{}, semantic.DefaultSemanticReportLimit),
		values: make([]semantic.Unsupported, 0, semantic.DefaultSemanticReportLimit),
	}
	for _, file := range files {
		for _, function := range file.Module.Functions {
			if !functions[function.Name] {
				continue
			}
			if exhaustedAt := collector.addFunction(file, function); exhaustedAt != nil {
				return collector.values, exhaustedAt
			}
		}
	}
	return collector.values, nil
}

func (collector *genericSignatureRestrictionCollector) addFunction(file semantic.File, function ast.Function) *hir.Span {
	// Preserve source visitation order even though the final report is sorted.
	if exhaustedAt := collector.addType(file, function.ResultType); exhaustedAt != nil {
		return exhaustedAt
	}
	for _, parameter := range function.Parameters {
		if exhaustedAt := collector.addType(file, parameter.Type); exhaustedAt != nil {
			return exhaustedAt
		}
	}
	return nil
}

func (collector *genericSignatureRestrictionCollector) addType(file semantic.File, reference ast.TypeReference) *hir.Span {
	value, unsupported := genericSignatureRestriction(file, reference)
	if !unsupported {
		return nil
	}
	key := genericUnsupportedKey{span: value.Span, feature: value.Feature}
	if _, duplicate := collector.seen[key]; duplicate {
		return nil
	}
	// A duplicate at capacity must not consume the terminal diagnostic.
	if len(collector.values) == semantic.DefaultSemanticReportLimit {
		exhaustedAt := value.Span
		return &exhaustedAt
	}
	collector.seen[key] = struct{}{}
	collector.values = append(collector.values, value)
	return nil
}

func genericSignatureRestriction(file semantic.File, reference ast.TypeReference) (semantic.Unsupported, bool) {
	if !unadmittedGenericScalar(reference) {
		return semantic.Unsupported{}, false
	}
	return semantic.Unsupported{
		Feature: "type " + strings.TrimSpace(reference.Text),
		Span: hir.Span{Package: file.Package, Source: file.Path,
			StartByte: reference.Span.StartByte, EndByte: reference.Span.EndByte},
	}, true
}
