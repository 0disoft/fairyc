package cbackend

import (
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
)

// This is a closed private FFI contract, not permission for the fy_private_
// namespace or stable SDK linkage. Descriptor validity remains the unsafe
// caller's responsibility; the SDK candidate owns that caller.
func validVectorRuntimeFunction(function mir.ForeignFunction, targetID target.ID) bool {
	if targetID != target.X8664WindowsMSVC {
		return false
	}
	pointer := hir.RawPointer(hir.TypeVoid, true)
	result := hir.TypeU32
	var parameters []hir.Type
	switch function.LinkName {
	case "fy_private_vector_initialize_v0":
		parameters = []hir.Type{pointer, hir.TypeUSize, hir.TypeUSize}
	case "fy_private_vector_release_empty_v0":
		parameters = []hir.Type{pointer}
	case "fy_private_vector_reserve_v0":
		parameters = []hir.Type{pointer, hir.TypeUSize}
	case "fy_private_vector_prepare_push_v0":
		parameters = []hir.Type{pointer, hir.TypeUSize, pointer}
	case "fy_private_vector_commit_push_v0":
		parameters = []hir.Type{pointer, pointer, hir.TypeUSize}
	case "fy_private_vector_pop_slot_v0":
		result, parameters = pointer, []hir.Type{pointer, hir.TypeUSize}
	case "fy_private_vector_element_slot_v0":
		shared := hir.RawPointer(hir.TypeVoid, false)
		result, parameters = shared, []hir.Type{shared, hir.TypeUSize, hir.TypeUSize}
	default:
		return false
	}
	return function.ResultType == result && slices.Equal(function.ParameterTypes, parameters)
}
