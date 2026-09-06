package compiler

import (
	"errors"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// genericSourceDiagnostic publishes only source-owned, uniquely mapped failures.
// Unknown adapter/invariant failures remain errors; message prose is not parsed.
func genericSourceDiagnostic(err error, input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID) (semantic.Diagnostic, bool) {
	var located *genericsource.SourceError
	if !errors.As(err, &located) || located == nil {
		return semantic.Diagnostic{}, false
	}
	code := ""
	switch located.Kind {
	case genericsource.FailureUnknown:
		code = "E0101"
	case genericsource.FailurePrivate:
		code = "E0103"
	case genericsource.FailureDeclaration:
		code = "E4501"
	case genericsource.FailureRequest:
		code = "E4502"
	case genericsource.FailureExpansion:
		switch genericplan.ErrorCode(err) {
		case genericplan.CodeSpecializationCycle, genericplan.CodePlanningBudgetExceeded:
			code = "E4503"
		default:
			return semantic.Diagnostic{}, false
		}
	default:
		return semantic.Diagnostic{}, false
	}
	if genericplan.ErrorCode(err) == genericplan.CodeCatalogBudgetExceeded {
		code = "E4503"
	}
	var primary hir.Span
	matched := false
	for _, file := range input.Files {
		coordinate, exists := coordinates[file.Package]
		if !exists || coordinate == (packagegraph.CoordinateID{}) || genericPackageOwner(coordinate) != located.Owner || file.Module == nil || file.Module.Path != located.Module {
			continue
		}
		if matched || file.Path == "" || !located.Span.Valid(file.Module.SourceSize) {
			return semantic.Diagnostic{}, false
		}
		matched = true
		primary = hir.Span{Package: file.Package, Source: file.Path, StartByte: located.Span.StartByte, EndByte: located.Span.EndByte}
	}
	if !matched {
		return semantic.Diagnostic{}, false
	}
	return semantic.Diagnostic{Code: code, Primary: primary}, true
}
