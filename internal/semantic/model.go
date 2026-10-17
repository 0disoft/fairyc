// Package semantic resolves names and creates typed HIR for the implemented Stage 0 subset.
package semantic

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/source"
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
	// Test is loader/harness-owned admission, never inferred from a source path.
	Test    bool `json:"-"`
	Package hir.PackageID
	Path    string
	Module  *ast.Module
}

// PackageImport binds one source import path from a resolved package instance to its exact direct
// provider instance. It is build input, not source syntax or a public package manifest field.
type PackageImport struct {
	FromPackage hir.PackageID
	ImportPath  string
	ToPackage   hir.PackageID
}

// PackageInput is one already-resolved, source-acquired semantic compilation input.
type PackageInput struct {
	Files   []File
	Imports []PackageImport
}

type packageImportKey struct {
	packageID hir.PackageID
	path      string
}

// Result contains observable semantic facts and build-local typed HIR.
type Result struct {
	// Set by the compiler coordinator when source templates were expanded or
	// removed. Consumers cannot recover those declarations from ordinary HIR.
	RequiresGenericProvenance bool              `json:"-"`
	NominalTypes              []NominalTypeFact `json:"-"`
	HIR                       hir.Package
	CompletedThrough          Phase
	FailedPhase               Phase
	Diagnostics               []Diagnostic
	Resolutions               []Resolution
	Types                     []TypeFact
	Borrows                   []BorrowFact
	RegionOrigins             []RegionOriginFact
	ResultHandling            []ResultHandlingFact
	Effects                   []EffectFact
	ResourceStates            []ResourceStateFact
	Constants                 []ConstantFact
	Unsupported               []Unsupported
}

// ConstantFact retains one private module constant without adding an executable HIR declaration.
type ConstantFact struct {
	Package     hir.PackageID
	Module      string
	Name        string
	Declaration hir.Span
	Type        hir.Type
	Value       string
	Span        hir.Span
}

// Diagnostic is one semantic rejection with role-labelled related source spans.
type Diagnostic struct {
	Code    string
	Primary hir.Span
	Related []RelatedSpan
	// ExpectedType and ActualType describe direct-call and named-struct field-store mismatches.
	// Canonical package qualifiers are build-local, not compatibility identities.
	ExpectedType hir.Type
	ActualType   hir.Type
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

// SymbolCoordinate is fixture-facing declaration identity within one compilation. Package is a
// build-local discriminator and must not be serialized as a compatibility fact.
type SymbolCoordinate struct {
	Package     hir.PackageID
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

// EffectFact records one callable's canonical declared source-effect set.
type EffectFact struct {
	Declaration hir.Span
	Effects     []hir.Effect
}

// BorrowFact records the hidden origin and local lifetime of one safe borrow.
type BorrowFact struct {
	// Function disambiguates specializations sharing original source coordinates.
	// It is build-local and never belongs to a serialized semantic contract.
	Function hir.SymbolID `json:"-"`
	Borrow   hir.Span
	Kind     string
	Origin   string
	LastUse  *hir.Span
	Escapes  bool
}

// RegionOriginFact records one compiler-known allocation and its hidden lexical origin.
type RegionOriginFact struct {
	Operation  string
	Allocation hir.Span
	Origin     string
	LastUse    *hir.Span
	Escapes    bool
}

// ResultHandlingFact records one source-visible handling decision for a must-use Result.
type ResultHandlingFact struct {
	Expression      hir.Span
	Mode            string
	ResultType      string
	SuccessType     string
	ErrorType       string
	EvaluationCount int
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
	return analyze(files, nil)
}

// AnalyzePackages resolves an already-resolved multi-package input without choosing a public
// manifest, lockfile, solver, source acquisition, or PackageInstanceId encoding.
func AnalyzePackages(input PackageInput) (Result, error) {
	bindings, err := validatePackageInput(input)
	if err != nil {
		return Result{}, err
	}
	return analyze(input.Files, bindings), nil
}

func analyze(files []File, bindings map[packageImportKey]hir.PackageID) Result {
	return analyzeBoundNominals(files, bindings, nil)
}

func analyzeBoundNominals(files []File, bindings map[packageImportKey]hir.PackageID, nominals map[string]NominalTypeBinding) Result {
	return analyzeBoundNominalsWithVectorCandidate(files, bindings, nominals, nil)
}

func analyzeBoundNominalsWithVectorCandidate(files []File, bindings map[packageImportKey]hir.PackageID, nominals map[string]NominalTypeBinding, vectorTypes map[vectorTypeKey]bool) Result {
	ordered := cloneResourceAbilityInputs(files)
	slices.SortFunc(ordered, func(a, b File) int {
		if byPackage := cmp.Compare(a.Package, b.Package); byPackage != 0 {
			return byPackage
		}
		return cmp.Compare(a.Path, b.Path)
	})
	testDiagnostics := TestDeclarationDiagnostics(ordered)
	for index := range ordered {
		ordered[index].Path = hir.QualifiedSourcePath(ordered[index].Package, ordered[index].Path)
	}
	analyzer := newAnalyzer(ordered, bindings)
	analyzer.nominalBindings = nominals
	analyzer.vectorForTypes = vectorTypes
	analyzer.diagnostics = testDiagnostics
	if len(analyzer.diagnostics) == 0 {
		analyzer.resolve()
	}
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
	result.ResultHandling = analyzer.resultHandling
	result.Effects = analyzer.effectFacts
	result.ResourceStates = analyzer.resourceStates
	result.Constants = analyzer.constantFacts
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
		result.NominalTypes = analyzer.visibleNominalTypes()
	} else if analyzer.flowStarted {
		result.CompletedThrough = PhaseType
	} else {
		result.CompletedThrough = PhaseResolve
	}
	return result
}

func validatePackageInput(input PackageInput) (map[packageImportKey]hir.PackageID, error) {
	packages := make(map[hir.PackageID]struct{})
	usedImports := make(map[packageImportKey]struct{})
	sourcePaths := make(map[packageImportKey]struct{})
	for _, file := range input.Files {
		packages[file.Package] = struct{}{}
		sourceKey := packageImportKey{packageID: file.Package, path: file.Path}
		if _, duplicate := sourcePaths[sourceKey]; duplicate {
			return nil, fmt.Errorf("package %d source path %q is present more than once", file.Package, file.Path)
		}
		sourcePaths[sourceKey] = struct{}{}
		if file.Module == nil {
			return nil, fmt.Errorf("package %d source path %q has no syntax-clean module", file.Package, file.Path)
		}
		for _, imported := range file.Module.Imports {
			usedImports[packageImportKey{packageID: file.Package, path: imported.Path}] = struct{}{}
		}
	}
	bindings := make(map[packageImportKey]hir.PackageID, len(input.Imports))
	for _, binding := range input.Imports {
		if _, found := packages[binding.FromPackage]; !found {
			return nil, fmt.Errorf("package import source %d is not present", binding.FromPackage)
		}
		if _, found := packages[binding.ToPackage]; !found {
			return nil, fmt.Errorf("package import target %d is not present", binding.ToPackage)
		}
		if !validModulePath(binding.ImportPath) {
			return nil, fmt.Errorf("package import path %q is invalid", binding.ImportPath)
		}
		key := packageImportKey{packageID: binding.FromPackage, path: binding.ImportPath}
		if _, duplicate := bindings[key]; duplicate {
			return nil, fmt.Errorf("package import from %d for %q is bound more than once", binding.FromPackage, binding.ImportPath)
		}
		if _, used := usedImports[key]; !used {
			return nil, fmt.Errorf("package import from %d for %q has no source use", binding.FromPackage, binding.ImportPath)
		}
		bindings[key] = binding.ToPackage
	}
	for key := range usedImports {
		if _, found := bindings[key]; !found {
			return nil, fmt.Errorf("package import from %d for %q has no resolved binding", key.packageID, key.path)
		}
	}
	return bindings, nil
}

func validModulePath(value string) bool {
	segments := strings.Split(value, "::")
	if len(segments) == 0 {
		return false
	}
	for _, segment := range segments {
		if !source.ValidIdentifier(segment) {
			return false
		}
	}
	return true
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
		return strings.Compare(left.CanonicalType.String(), right.CanonicalType.String())
	})
	slices.SortFunc(a.borrows, func(left, right BorrowFact) int {
		if n := compareSpan(left.Borrow, right.Borrow); n != 0 {
			return n
		}
		return cmp.Compare(left.Origin, right.Origin)
	})
	slices.SortFunc(a.effectFacts, func(left, right EffectFact) int {
		return compareSpan(left.Declaration, right.Declaration)
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
	slices.SortFunc(a.resultHandling, func(left, right ResultHandlingFact) int {
		if n := compareSpan(left.Expression, right.Expression); n != 0 {
			return n
		}
		return cmp.Compare(left.Mode, right.Mode)
	})
	a.resultHandling = slices.CompactFunc(a.resultHandling, func(left, right ResultHandlingFact) bool {
		return compareSpan(left.Expression, right.Expression) == 0 && left.Mode == right.Mode && left.ResultType == right.ResultType && left.SuccessType == right.SuccessType && left.ErrorType == right.ErrorType && left.EvaluationCount == right.EvaluationCount
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
	slices.SortFunc(a.constantFacts, func(left, right ConstantFact) int {
		return compareSpan(left.Declaration, right.Declaration)
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
	if n := cmp.Compare(left.Package, right.Package); n != 0 {
		return n
	}
	if n := cmp.Compare(left.Source, right.Source); n != 0 {
		return n
	}
	if n := cmp.Compare(left.StartByte, right.StartByte); n != 0 {
		return n
	}
	return cmp.Compare(left.EndByte, right.EndByte)
}
