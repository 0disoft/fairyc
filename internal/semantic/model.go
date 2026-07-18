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
	PhaseFlow    Phase = "flow"
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
	Borrows          []BorrowFact
	RegionOrigins    []RegionOriginFact
	ResourceStates   []ResourceStateFact
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

// BorrowFact records the hidden origin and local lifetime of one safe borrow.
type BorrowFact struct {
	Borrow  hir.Span
	Kind    string
	Origin  string
	LastUse *hir.Span
	Escapes bool
}

// RegionOriginFact records one compiler-known allocation and its hidden lexical origin.
type RegionOriginFact struct {
	Operation  string
	Allocation hir.Span
	Origin     string
	LastUse    *hir.Span
	Escapes    bool
}

// ResourceState is the source-visible ownership state of one complete resource root.
type ResourceState string

const (
	ResourceLive      ResourceState = "Live"
	ResourceDead      ResourceState = "Dead"
	ResourceMaybeLive ResourceState = "MaybeLive"
)

// ResourceStateFact samples one binding immediately before or after a source span.
type ResourceStateFact struct {
	Binding            hir.Span
	At                 hir.Span
	Observation        string
	State              ResourceState
	CleanupObligations *int
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
	if len(analyzer.diagnostics) == 0 && len(analyzer.unsupported) == 0 {
		analyzer.flowCheck()
	}
	analyzer.finish()
	result.Diagnostics = analyzer.diagnostics
	result.Resolutions = analyzer.resolutions
	result.Types = analyzer.types
	result.Borrows = analyzer.borrows
	result.RegionOrigins = analyzer.regionOrigins
	result.ResourceStates = analyzer.resourceStates
	result.Unsupported = analyzer.unsupported
	if len(result.Diagnostics) > 0 {
		if analyzer.flowStarted {
			result.CompletedThrough = PhaseType
			result.FailedPhase = PhaseFlow
		} else {
			result.CompletedThrough = PhaseResolve
			result.FailedPhase = PhaseType
		}
	} else if len(result.Unsupported) == 0 {
		result.CompletedThrough = PhaseFlow
	} else if analyzer.flowStarted {
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
	slices.SortFunc(a.borrows, func(left, right BorrowFact) int {
		if n := compareSpan(left.Borrow, right.Borrow); n != 0 {
			return n
		}
		return cmp.Compare(left.Origin, right.Origin)
	})
	slices.SortFunc(a.regionOrigins, func(left, right RegionOriginFact) int {
		if n := compareSpan(left.Allocation, right.Allocation); n != 0 {
			return n
		}
		return cmp.Compare(left.Origin, right.Origin)
	})
	a.regionOrigins = slices.CompactFunc(a.regionOrigins, func(left, right RegionOriginFact) bool {
		return compareSpan(left.Allocation, right.Allocation) == 0 && left.Operation == right.Operation && left.Origin == right.Origin && left.Escapes == right.Escapes && optionalSpanEqual(left.LastUse, right.LastUse)
	})
	slices.SortFunc(a.resourceStates, func(left, right ResourceStateFact) int {
		if n := compareSpan(left.Binding, right.Binding); n != 0 {
			return n
		}
		if n := compareSpan(left.At, right.At); n != 0 {
			return n
		}
		return cmp.Compare(left.Observation, right.Observation)
	})
	a.resourceStates = slices.CompactFunc(a.resourceStates, func(left, right ResourceStateFact) bool {
		return compareSpan(left.Binding, right.Binding) == 0 && compareSpan(left.At, right.At) == 0 &&
			left.Observation == right.Observation && left.State == right.State && obligationsEqual(left.CleanupObligations, right.CleanupObligations)
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

func optionalSpanEqual(left, right *hir.Span) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return compareSpan(*left, *right) == 0
}

func obligationsEqual(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
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
