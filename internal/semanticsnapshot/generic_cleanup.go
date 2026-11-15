package semanticsnapshot

import (
	"fmt"
	"maps"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func buildGenericCleanupSummaries(result compiler.Result, implementations map[DeclarationKey]SpecializationSummary, metadata map[hir.PackageID]packageMetadata, symbols map[hir.SymbolID]DeclarationKey, foreign map[hir.SymbolID]Declaration, private map[hir.SymbolID]struct{}) (map[DeclarationKey]FunctionSummary, []semantic.BorrowFact, int, int, error) {
	view, err := selectGenericCleanupFunctions(result.MIR.Functions, genericCleanupOwners(implementations), metadata, private)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	// With no selected cleanup, the summary builder only reads the original symbols.
	// A selected view owns its copy before installing any synthetic bindings.
	cleanupSymbols := symbols
	if len(view.bindings) != 0 {
		cleanupSymbols = maps.Clone(symbols)
		maps.Copy(cleanupSymbols, view.bindings)
	}
	ordinary, cleanup := splitGenericCleanupBorrows(result.Semantic.Borrows, view.bindings)
	program := result.MIR
	program.Functions = view.functions
	values, resolved, external, err := buildFunctionSummaries(program, cleanup, metadata, cleanupSymbols, foreign, private, view.bounds)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	summaries, err := bindGenericCleanupSummaries(values, view.owners, implementations)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	return summaries, ordinary, resolved, external, nil
}

func genericCleanupOwners(implementations map[DeclarationKey]SpecializationSummary) map[string]DeclarationKey {
	owners := map[string]DeclarationKey{}
	for key, implementation := range implementations {
		if implementation.Aggregate == nil || key.Kind != DeclarationResource {
			continue
		}
		// joinGenericImplementations verified this nominal type, including its
		// owning coordinate, against the original declaration's signature.
		coordinate := implementation.Aggregate.Type.PackageCoordinate
		owners[genericNominalKey(TypeRef{PackageCoordinate: coordinate, Name: key.Module + "::" + key.Name})] = key
	}
	return owners
}

type genericCleanupView struct {
	functions []mir.Function
	owners    map[DeclarationKey]DeclarationKey
	bounds    map[DeclarationKey]Span
	bindings  map[hir.SymbolID]DeclarationKey
}

// selectGenericCleanupFunctions changes only its local summary view, never executable MIR.
func selectGenericCleanupFunctions(functions []mir.Function, owners map[string]DeclarationKey, metadata map[hir.PackageID]packageMetadata, private map[hir.SymbolID]struct{}) (genericCleanupView, error) {
	view := genericCleanupView{
		owners:   map[DeclarationKey]DeclarationKey{},
		bounds:   map[DeclarationKey]Span{},
		bindings: map[hir.SymbolID]DeclarationKey{},
	}
	for _, function := range functions {
		if function.DropGlueFor == hir.ParseType("") {
			continue
		}
		typ, err := normalizeType(function.DropGlueFor, metadata)
		if err != nil {
			return genericCleanupView{}, err
		}
		owner, found := owners[genericNominalKey(typ)]
		if !found {
			continue
		}
		if _, known := private[function.Symbol]; !known {
			return genericCleanupView{}, fmt.Errorf("generic cleanup has no private compiler owner")
		}
		key := owner
		key.Kind = DeclarationFunction
		key.Name += "_cleanup"
		if _, duplicate := view.owners[key]; duplicate {
			return genericCleanupView{}, fmt.Errorf("generic resource has duplicate cleanup implementations")
		}
		span, err := normalizeSpan(function.Span, metadata)
		if err != nil {
			return genericCleanupView{}, err
		}
		view.owners[key], view.bounds[key], view.bindings[function.Symbol] = owner, span, key
		function.DropGlueFor = hir.ParseType("")
		view.functions = append(view.functions, function)
	}
	return view, nil
}

func splitGenericCleanupBorrows(facts []semantic.BorrowFact, selected map[hir.SymbolID]DeclarationKey) ([]semantic.BorrowFact, []semantic.BorrowFact) {
	var ordinary, cleanup []semantic.BorrowFact
	for _, fact := range facts {
		if _, found := selected[fact.Function]; !found {
			ordinary = append(ordinary, fact)
			continue
		}
		cleanup = append(cleanup, fact)
	}
	return ordinary, cleanup
}

func bindGenericCleanupSummaries(values []FunctionSummary, owners map[DeclarationKey]DeclarationKey, implementations map[DeclarationKey]SpecializationSummary) (map[DeclarationKey]FunctionSummary, error) {
	summaries := map[DeclarationKey]FunctionSummary{}
	for _, value := range values {
		owner := owners[value.Function]
		value.Function = implementations[owner].Declaration
		summaries[owner] = value
	}
	for key, implementation := range implementations {
		if implementation.Aggregate == nil || implementation.Aggregate.DropKind != string(hir.ResourceDropNonEmpty) {
			continue
		}
		if _, found := summaries[key]; !found {
			return nil, fmt.Errorf("non-empty aggregate cleanup has no executable summary")
		}
	}
	return summaries, nil
}
