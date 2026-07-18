package mir

import "slices"

// Clone makes an independent MIR copy for verifier mutation and backend isolation.
func Clone(input Program) Program {
	result := Program{Functions: make([]Function, len(input.Functions))}
	for functionIndex, function := range input.Functions {
		cloned := function
		cloned.Resources = slices.Clone(function.Resources)
		cloned.DropFlags = slices.Clone(function.DropFlags)
		cloned.Loans = slices.Clone(function.Loans)
		cloned.ReturnOrigins = slices.Clone(function.ReturnOrigins)
		cloned.RegionValues = slices.Clone(function.RegionValues)
		cloned.UnsafeOperations = slices.Clone(function.UnsafeOperations)
		cloned.Entities = slices.Clone(function.Entities)
		cloned.Blocks = make([]Block, len(function.Blocks))
		for blockIndex, block := range function.Blocks {
			clonedBlock := block
			clonedBlock.Operations = slices.Clone(block.Operations)
			for operationIndex := range clonedBlock.Operations {
				clonedBlock.Operations[operationIndex].TrapCategories = slices.Clone(block.Operations[operationIndex].TrapCategories)
			}
			clonedBlock.Terminator.Successors = slices.Clone(block.Terminator.Successors)
			clonedBlock.Terminator.TrapCategories = slices.Clone(block.Terminator.TrapCategories)
			cloned.Blocks[blockIndex] = clonedBlock
		}
		cloned.OrderedEvaluations = slices.Clone(function.OrderedEvaluations)
		for index := range cloned.OrderedEvaluations {
			cloned.OrderedEvaluations[index].Sequence = slices.Clone(function.OrderedEvaluations[index].Sequence)
		}
		cloned.Calls = slices.Clone(function.Calls)
		for index := range cloned.Calls {
			cloned.Calls[index].ArgumentOrder = slices.Clone(function.Calls[index].ArgumentOrder)
		}
		cloned.Moves = slices.Clone(function.Moves)
		cloned.CleanupPaths = slices.Clone(function.CleanupPaths)
		for index := range cloned.CleanupPaths {
			cloned.CleanupPaths[index].Actions = slices.Clone(function.CleanupPaths[index].Actions)
		}
		cloned.ResultBranches = slices.Clone(function.ResultBranches)
		for index := range cloned.ResultBranches {
			cloned.ResultBranches[index].ResultTemporaryDeadOn = slices.Clone(function.ResultBranches[index].ResultTemporaryDeadOn)
			cloned.ResultBranches[index].ErrorCleanupActions = slices.Clone(function.ResultBranches[index].ErrorCleanupActions)
		}
		cloned.TrapPaths = slices.Clone(function.TrapPaths)
		for index := range cloned.TrapPaths {
			cloned.TrapPaths[index].Categories = slices.Clone(function.TrapPaths[index].Categories)
			cloned.TrapPaths[index].CleanupActions = slices.Clone(function.TrapPaths[index].CleanupActions)
		}
		cloned.EliminatedForms = slices.Clone(function.EliminatedForms)
		result.Functions[functionIndex] = cloned
	}
	return result
}
