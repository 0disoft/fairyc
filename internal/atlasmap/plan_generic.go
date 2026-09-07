package atlasmap

import (
	"slices"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

// Specializations returns detached concrete summaries owned by this package.
// IDs are compiler-provenance keys in the exact build, not source SymbolIds.
func (partition Partition) Specializations() []semanticsnapshot.SpecializationSummary {
	result := slices.Clone(partition.data.specializations)
	for i := range result {
		result[i].TypeArguments = slices.Clone(result[i].TypeArguments)
		for j := range result[i].TypeArguments {
			result[i].TypeArguments[j] = cloneTypeRef(result[i].TypeArguments[j])
		}
		result[i].Function = cloneFunctionSummaries([]semanticsnapshot.FunctionSummary{result[i].Function})[0]
	}
	return result
}
