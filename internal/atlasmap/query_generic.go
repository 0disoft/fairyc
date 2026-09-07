package atlasmap

import (
	"slices"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

// QuerySpecializationFacts reports observations for one concrete implementation,
// never universal executable facts about the original template.
type QuerySpecializationFacts struct {
	ID            string                     `json:"id"`
	TypeArguments []semanticsnapshot.TypeRef `json:"type_arguments"`
	Function      QueryFunctionFacts         `json:"function"`
}

// NewQueryIndexV2 admits generic-aware exact-current maps without changing the v1 entry point.
func NewQueryIndexV2(partition Partition, expectation LoadExpectation) (*QueryIndex, error) {
	return newQueryIndex(partition, expectation, LogicalSchemaV2)
}

func (index *QueryIndex) addSpecializations(values []semanticsnapshot.SpecializationSummary) {
	for _, value := range values {
		relations := index.buildRelations([]semanticsnapshot.FunctionSummary{value.Function})
		counts := map[RelationKind]int{}
		for i := range relations {
			relations[i].SourceSpecialization = value.ID
			counts[relations[i].Kind]++
		}
		facts := QueryFunctionFacts{Provenance: value.Function.Provenance}
		for _, kind := range supportedRelationKinds {
			if count := counts[kind]; count != 0 {
				facts.Relations = append(facts.Relations, RelationKindCount{Kind: kind, Count: count})
			}
		}
		position := index.byKey[value.Declaration]
		node := &index.nodes[position]
		node.specializations = append(node.specializations, QuerySpecializationFacts{ID: value.ID, TypeArguments: value.TypeArguments, Function: facts})
		index.relations = append(index.relations, relations...)
	}
}

func cloneQuerySpecializations(values []QuerySpecializationFacts) []QuerySpecializationFacts {
	result := slices.Clone(values)
	for i := range result {
		result[i].TypeArguments = slices.Clone(result[i].TypeArguments)
		for j := range result[i].TypeArguments {
			result[i].TypeArguments[j] = cloneTypeRef(result[i].TypeArguments[j])
		}
		result[i].Function.Relations = slices.Clone(result[i].Function.Relations)
	}
	return result
}
