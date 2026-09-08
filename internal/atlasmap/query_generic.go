package atlasmap

import (
	"slices"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

// QuerySpecializationFacts reports observations for one concrete implementation,
// never universal executable facts about the original template.
type QuerySpecializationFacts struct {
	ID            string                        `json:"id"`
	TypeArguments []semanticsnapshot.TypeRef    `json:"type_arguments"`
	Function      *QueryFunctionFacts           `json:"function,omitempty"`
	Aggregate     *semanticsnapshot.Declaration `json:"aggregate,omitempty"`
	Cleanup       *QueryFunctionFacts           `json:"cleanup,omitempty"`
}

// NewQueryIndexV2 admits generic-aware exact-current maps without changing the v1 entry point.
func NewQueryIndexV2(partition Partition, expectation LoadExpectation) (*QueryIndex, error) {
	return newQueryIndex(partition, expectation, LogicalSchemaV2)
}

// NewQueryIndexV3 exposes aggregate and cleanup facts from exact-current v3 maps.
func NewQueryIndexV3(partition Partition, expectation LoadExpectation) (*QueryIndex, error) {
	return newQueryIndex(partition, expectation, LogicalSchemaV3)
}

func (index *QueryIndex) addSpecializations(values []semanticsnapshot.SpecializationSummary) {
	for _, value := range values {
		facts := QuerySpecializationFacts{ID: value.ID, TypeArguments: value.TypeArguments, Aggregate: value.Aggregate}
		var relations []QueryRelation
		if value.Aggregate == nil {
			facts.Function, relations = index.specializationBodyFacts(value.Function, value.ID, "")
		} else if value.Cleanup != nil {
			facts.Cleanup, relations = index.specializationBodyFacts(*value.Cleanup, value.ID, "cleanup")
		}
		position := index.byKey[value.Declaration]
		node := &index.nodes[position]
		node.specializations = append(node.specializations, facts)
		index.relations = append(index.relations, relations...)
	}
}

func (index *QueryIndex) specializationBodyFacts(summary semanticsnapshot.FunctionSummary, id, body string) (*QueryFunctionFacts, []QueryRelation) {
	relations := index.buildRelations([]semanticsnapshot.FunctionSummary{summary})
	counts := map[RelationKind]int{}
	for i := range relations {
		relations[i].SourceSpecialization = id
		relations[i].SourceBody = body
		counts[relations[i].Kind]++
	}
	facts := &QueryFunctionFacts{Provenance: summary.Provenance}
	for _, kind := range supportedRelationKinds {
		if count := counts[kind]; count != 0 {
			facts.Relations = append(facts.Relations, RelationKindCount{Kind: kind, Count: count})
		}
	}
	return facts, relations
}

func cloneQuerySpecializations(values []QuerySpecializationFacts) []QuerySpecializationFacts {
	result := slices.Clone(values)
	for i := range result {
		result[i].TypeArguments = slices.Clone(result[i].TypeArguments)
		for j := range result[i].TypeArguments {
			result[i].TypeArguments[j] = cloneTypeRef(result[i].TypeArguments[j])
		}
		if result[i].Function != nil {
			value := *result[i].Function
			value.Relations = slices.Clone(value.Relations)
			result[i].Function = &value
		}
		if result[i].Aggregate != nil {
			value := cloneDeclarations([]semanticsnapshot.Declaration{*result[i].Aggregate})[0]
			result[i].Aggregate = &value
		}
		if result[i].Cleanup != nil {
			value := *result[i].Cleanup
			value.Relations = slices.Clone(value.Relations)
			result[i].Cleanup = &value
		}
	}
	return result
}
