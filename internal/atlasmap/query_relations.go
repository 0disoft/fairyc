package atlasmap

import (
	"slices"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func (index *QueryIndex) buildRelations(functions []semanticsnapshot.FunctionSummary) []QueryRelation {
	result := make([]QueryRelation, 0, countRelations(functions))
	for _, function := range functions {
		sourcePosition := index.byKey[function.Function]
		source := index.nodes[sourcePosition].symbol.Visible
		for _, call := range function.Calls {
			kind := RelationCalls
			if call.TargetKind == semanticsnapshot.CallTargetExternal {
				kind = RelationCallsForeign
			}
			target := &QueryTarget{
				PackageInstance: call.Target.PackageInstance, Module: call.Target.Module,
				Kind: call.Target.Kind, Name: call.Target.Name, Specialization: call.Specialization,
			}
			if targetPosition, found := index.byKey[call.Target]; found {
				target.Symbol = index.nodes[targetPosition].symbol.Visible
			}
			span := call.Span
			result = append(result, QueryRelation{
				Source: source, Kind: kind, Target: target, Span: &span, Certainty: call.Certainty,
				Provenance: function.Provenance, ABI: call.ABI, LinkName: call.LinkName,
			})
		}
		for _, category := range function.TrapCategories {
			result = append(result, QueryRelation{Source: source, Kind: RelationMayTrap, Provenance: function.Provenance, Detail: category})
		}
		result = appendSpanRelations(result, source, RelationUsesUnsafe, function.Provenance, function.UnsafeSites)
		result = appendSpanRelations(result, source, RelationMoves, function.Provenance, function.MoveSites)
		result = appendSpanRelations(result, source, RelationDrops, function.Provenance, function.DropSites)
		result = appendSpanRelations(result, source, RelationAllocates, function.Provenance, function.AllocationSites)
		result = appendSpanRelations(result, source, RelationReturnsError, function.Provenance, function.ReturnErrorSites)
		for _, borrow := range function.Borrows {
			kind := RelationBorrowsShared
			if borrow.Kind == "mutable" {
				kind = RelationBorrowsMutable
			}
			span := borrow.Span
			result = append(result, QueryRelation{
				Source: source, Kind: kind, Span: &span, Provenance: function.Provenance, Escapes: borrow.Escapes,
			})
		}
	}
	slices.SortFunc(result, compareQueryRelation)
	return result
}

func appendSpanRelations(
	result []QueryRelation,
	source string,
	kind RelationKind,
	provenance semanticsnapshot.Provenance,
	spans []semanticsnapshot.Span,
) []QueryRelation {
	for _, value := range spans {
		span := value
		result = append(result, QueryRelation{Source: source, Kind: kind, Span: &span, Provenance: provenance})
	}
	return result
}
