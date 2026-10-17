package semanticsnapshot

import (
	"fmt"
	"maps"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func buildGenericCleanupSummaries(result compiler.Result, implementations map[DeclarationKey]SpecializationSummary, metadata map[hir.PackageID]packageMetadata, symbols map[hir.SymbolID]DeclarationKey, foreign map[hir.SymbolID]Declaration, private map[hir.SymbolID]struct{}) (map[DeclarationKey]FunctionSummary, []semantic.BorrowFact, int, int, error) {
	owners := map[string]DeclarationKey{}
	for key, implementation := range implementations {
		if implementation.Aggregate == nil || key.Kind != DeclarationResource {
			continue
		}
		for _, info := range metadata {
			if info.instance == key.PackageInstance {
				owners[genericNominalKey(TypeRef{PackageCoordinate: info.coord, Name: key.Module + "::" + key.Name})] = key
			}
		}
	}
	program := result.MIR
	program.Functions = nil
	symbols = maps.Clone(symbols)
	bounds := map[DeclarationKey]Span{}
	keys := map[DeclarationKey]DeclarationKey{}
	selected := map[hir.SymbolID]bool{}
	for _, function := range result.MIR.Functions {
		if function.DropGlueFor == hir.ParseType("") {
			continue
		}
		typ, err := normalizeType(function.DropGlueFor, metadata)
		if err != nil {
			return nil, nil, 0, 0, err
		}
		owner, found := owners[genericNominalKey(typ)]
		if !found {
			continue
		}
		if _, known := private[function.Symbol]; !known {
			return nil, nil, 0, 0, fmt.Errorf("generic cleanup has no private compiler owner")
		}
		key := owner
		key.Kind = DeclarationFunction
		key.Name += "_cleanup"
		if _, duplicate := keys[key]; duplicate {
			return nil, nil, 0, 0, fmt.Errorf("generic resource has duplicate cleanup implementations")
		}
		span, err := normalizeSpan(function.Span, metadata)
		if err != nil {
			return nil, nil, 0, 0, err
		}
		keys[key], bounds[key], symbols[function.Symbol], selected[function.Symbol] = owner, span, key, true
		// Only the summary view treats this verified private body as a function.
		// Neither source declarations nor the executable MIR are changed.
		function.DropGlueFor = hir.ParseType("")
		program.Functions = append(program.Functions, function)
	}
	var ordinary, cleanup []semantic.BorrowFact
	for _, fact := range result.Semantic.Borrows {
		if selected[fact.Function] {
			cleanup = append(cleanup, fact)
		} else {
			ordinary = append(ordinary, fact)
		}
	}
	values, resolved, external, err := buildFunctionSummaries(program, cleanup, metadata, symbols, foreign, private, bounds)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	summaries := map[DeclarationKey]FunctionSummary{}
	for _, value := range values {
		owner := keys[value.Function]
		value.Function = implementations[owner].Declaration
		summaries[owner] = value
	}
	for key, implementation := range implementations {
		if implementation.Aggregate != nil && implementation.Aggregate.DropKind == "non-empty" {
			if _, found := summaries[key]; !found {
				return nil, nil, 0, 0, fmt.Errorf("non-empty aggregate cleanup has no executable summary")
			}
		}
	}
	return summaries, ordinary, resolved, external, nil
}
