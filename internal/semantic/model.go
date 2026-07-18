// Package semantic resolves names and creates typed HIR for the implemented Stage 0 subset.
package semantic

import (
	"cmp"
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

// Phase is one ordered semantic analysis boundary.
type Phase string

const (
	PhaseNone    Phase = ""
	PhaseResolve Phase = "resolve"
	PhaseType    Phase = "type"
)

// File is one syntax-clean AST with its source-root-relative identity.
type File struct {
	Path   string
	Module *ast.Module
}

// Result contains observable semantic facts and build-local typed HIR.
type Result struct {
	HIR              hir.Package
	CompletedThrough Phase
	FailedPhase      Phase
	Diagnostics      []Diagnostic
	Resolutions      []Resolution
	Types            []TypeFact
	Unsupported      []Unsupported
}

// Diagnostic is one semantic rejection with role-labelled related source spans.
type Diagnostic struct {
	Code    string
	Primary hir.Span
	Related []RelatedSpan
}

// RelatedSpan explains another source location involved in a diagnostic.
type RelatedSpan struct {
	Role string
	Span hir.Span
}

// Resolution records a path use and its resolved declaration coordinate.
type Resolution struct {
	Use    hir.Span
	Target SymbolCoordinate
}

// SymbolCoordinate is stable fixture-facing declaration identity.
type SymbolCoordinate struct {
	Module      string
	Name        string
	Kind        string
	Declaration hir.Span
}

// TypeFact records one canonical type at a source expression or declaration span.
type TypeFact struct {
	Expression    hir.Span
	CanonicalType hir.Type
}

// Unsupported identifies syntax outside the deliberately narrow implemented semantic subset.
type Unsupported struct {
	Feature string
	Span    hir.Span
}

// Analyze resolves every file before creating type facts or HIR.
func Analyze(files []File) Result {
	ordered := append([]File(nil), files...)
	slices.SortFunc(ordered, func(a, b File) int { return cmp.Compare(a.Path, b.Path) })
	analyzer := newAnalyzer(ordered)
	analyzer.resolve()
	if len(analyzer.diagnostics) > 0 {
		analyzer.finish()
		return Result{
			FailedPhase: PhaseResolve,
			Diagnostics: analyzer.diagnostics,
			Resolutions: analyzer.resolutions,
			Unsupported: analyzer.unsupported,
		}
	}
	result := analyzer.typeCheck()
	analyzer.finish()
	result.Diagnostics = analyzer.diagnostics
	result.Resolutions = analyzer.resolutions
	result.Types = analyzer.types
	result.Unsupported = analyzer.unsupported
	if len(result.Diagnostics) > 0 {
		result.CompletedThrough = PhaseResolve
		result.FailedPhase = PhaseType
	} else if len(result.Unsupported) == 0 {
		result.CompletedThrough = PhaseType
	} else {
		result.CompletedThrough = PhaseResolve
	}
	return result
}

func (a *analyzer) finish() {
	for index := range a.diagnostics {
		slices.SortFunc(a.diagnostics[index].Related, compareRelated)
	}
	slices.SortFunc(a.diagnostics, compareDiagnostic)
	a.diagnostics = slices.CompactFunc(a.diagnostics, func(left, right Diagnostic) bool {
		return compareDiagnostic(left, right) == 0 && slices.EqualFunc(left.Related, right.Related, func(a, b RelatedSpan) bool {
			return compareRelated(a, b) == 0
		})
	})
	slices.SortFunc(a.resolutions, func(left, right Resolution) int {
		return compareSpan(left.Use, right.Use)
	})
	slices.SortFunc(a.types, func(left, right TypeFact) int {
		if n := compareSpan(left.Expression, right.Expression); n != 0 {
			return n
		}
		return cmp.Compare(left.CanonicalType, right.CanonicalType)
	})
	a.types = slices.CompactFunc(a.types, func(left, right TypeFact) bool {
		return compareSpan(left.Expression, right.Expression) == 0 && left.CanonicalType == right.CanonicalType
	})
	slices.SortFunc(a.unsupported, func(left, right Unsupported) int {
		if n := compareSpan(left.Span, right.Span); n != 0 {
			return n
		}
		return cmp.Compare(left.Feature, right.Feature)
	})
	a.unsupported = slices.CompactFunc(a.unsupported, func(left, right Unsupported) bool {
		return compareSpan(left.Span, right.Span) == 0 && left.Feature == right.Feature
	})
}

func compareDiagnostic(left, right Diagnostic) int {
	if n := compareSpan(left.Primary, right.Primary); n != 0 {
		return n
	}
	return cmp.Compare(left.Code, right.Code)
}

func compareRelated(left, right RelatedSpan) int {
	if n := cmp.Compare(left.Role, right.Role); n != 0 {
		return n
	}
	return compareSpan(left.Span, right.Span)
}

func compareSpan(left, right hir.Span) int {
	if n := cmp.Compare(left.Source, right.Source); n != 0 {
		return n
	}
	if n := cmp.Compare(left.StartByte, right.StartByte); n != 0 {
		return n
	}
	return cmp.Compare(left.EndByte, right.EndByte)
}
