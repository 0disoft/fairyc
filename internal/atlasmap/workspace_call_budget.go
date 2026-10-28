package atlasmap

import "github.com/0disoft/fairyc/internal/semanticsnapshot"

// workspaceCallBudgetExceeded counts the same call bodies as buildRelations and
// addSpecializations, without copying summaries or constructing query indexes.
// This is rejection-only: successful joins still validate every partition.
func workspaceCallBudgetExceeded(partitions []Partition, remaining int) bool {
	exceeds := func(calls []semanticsnapshot.Call) bool {
		for i := range calls {
			if calls[i].TargetKind == semanticsnapshot.CallTargetExternal {
				continue
			}
			if remaining == 0 {
				return true
			}
			remaining--
		}
		return false
	}
	for _, partition := range partitions {
		for _, function := range partition.data.functions {
			if exceeds(function.Calls) {
				return true
			}
		}
		for _, value := range partition.data.specializations {
			if value.Aggregate == nil {
				if exceeds(value.Function.Calls) {
					return true
				}
			} else if value.Cleanup != nil && exceeds(value.Cleanup.Calls) {
				return true
			}
		}
	}
	return false
}
