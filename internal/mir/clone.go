package mir

import "slices"

// Clone makes an independent MIR copy for verifier mutation and backend isolation.
func Clone(input Program) Program {
	result := Program{
		StructTypes:      slices.Clone(input.StructTypes),
		EnumTypes:        slices.Clone(input.EnumTypes),
		ResourceTypes:    slices.Clone(input.ResourceTypes),
		ForeignFunctions: slices.Clone(input.ForeignFunctions),
		Functions:        make([]Function, len(input.Functions)),
	}
	for index := range result.StructTypes {
		result.StructTypes[index].Fields = slices.Clone(input.StructTypes[index].Fields)
	}
	for index := range result.EnumTypes {
		result.EnumTypes[index].Variants = slices.Clone(input.EnumTypes[index].Variants)
		for variantIndex := range result.EnumTypes[index].Variants {
			result.EnumTypes[index].Variants[variantIndex].Fields = slices.Clone(input.EnumTypes[index].Variants[variantIndex].Fields)
		}
	}
	for index := range result.ForeignFunctions {
		result.ForeignFunctions[index].ParameterTypes = slices.Clone(input.ForeignFunctions[index].ParameterTypes)
	}
	for index := range result.ResourceTypes {
		result.ResourceTypes[index].Fields = slices.Clone(input.ResourceTypes[index].Fields)
	}
	for functionIndex, function := range input.Functions {
		cloned := function
		cloned.Parameters = slices.Clone(function.Parameters)
		cloned.ParameterTypes = slices.Clone(function.ParameterTypes)
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
				clonedBlock.Operations[operationIndex].Operands = slices.Clone(block.Operations[operationIndex].Operands)
				clonedBlock.Operations[operationIndex].Fields = slices.Clone(block.Operations[operationIndex].Fields)
			}
			clonedBlock.Terminator.Successors = slices.Clone(block.Terminator.Successors)
			clonedBlock.Terminator.TrapCategories = slices.Clone(block.Terminator.TrapCategories)
			clonedBlock.Terminator.Arguments = slices.Clone(block.Terminator.Arguments)
			clonedBlock.Terminator.SwitchOperands = slices.Clone(block.Terminator.SwitchOperands)
			clonedBlock.Terminator.SwitchVariants = slices.Clone(block.Terminator.SwitchVariants)
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
		cloned.Reassignments = slices.Clone(function.Reassignments)
		cloned.RegionScopes = slices.Clone(function.RegionScopes)
		for index := range cloned.RegionScopes {
			cloned.RegionScopes[index].Placements = slices.Clone(function.RegionScopes[index].Placements)
			cloned.RegionScopes[index].CleanupBeforeReset = slices.Clone(function.RegionScopes[index].CleanupBeforeReset)
			cloned.RegionScopes[index].EarlyResets = slices.Clone(function.RegionScopes[index].EarlyResets)
			for resetIndex := range cloned.RegionScopes[index].EarlyResets {
				cloned.RegionScopes[index].EarlyResets[resetIndex].CleanupBeforeReset = slices.Clone(function.RegionScopes[index].EarlyResets[resetIndex].CleanupBeforeReset)
			}
		}
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
