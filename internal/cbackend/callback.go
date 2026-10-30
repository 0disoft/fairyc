package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
)

func (e *emitter) emitCallbackType(output *strings.Builder) error {
	if _, callbacks := e.analysis.features["bounded-c-callback"]; !callbacks {
		return nil
	}
	spec, found := target.Lookup(e.analysis.targetID)
	if !found {
		return fmt.Errorf("missing callback target layout %q", e.analysis.targetID)
	}
	output.WriteString("typedef int32_t (*fy_c_callback_i32_userdata)(int32_t, void *);\n")
	fmt.Fprintf(output, "_Static_assert(sizeof(fy_c_callback_i32_userdata) == UINT64_C(%d), \"fairyc callback pointer size\");\n", spec.FunctionPointer.SizeBytes)
	fmt.Fprintf(output, "_Static_assert(_Alignof(fy_c_callback_i32_userdata) == UINT64_C(%d), \"fairyc callback pointer alignment\");\n\n", spec.FunctionPointer.AlignmentBytes)
	return nil
}

// Adapters keep the C callback contract separate from private Fairy functions.
func (e *emitter) emitCallbackAdapters(output *strings.Builder) {
	for _, index := range e.callbackTargets() {
		function := e.analysis.functions[index]
		if e.analysis.nonReturning[function.Symbol] {
			fmt.Fprintf(output, "%s_Noreturn int32_t fy_cb_%d(int32_t value, void *data) { fy_fn_%d(value, data); }\n", e.functionLinkage(), index, index)
		} else {
			fmt.Fprintf(output, "%sint32_t fy_cb_%d(int32_t value, void *data) { return fy_fn_%d(value, data); }\n", e.functionLinkage(), index, index)
		}
	}
}

func (e *emitter) callbackTargets() []int {
	targets := make(map[int]bool)
	for fi, function := range e.analysis.functions {
		for bi, block := range function.Blocks {
			if !e.analysis.reachableBlocks[fi][bi] {
				continue
			}
			for _, operation := range block.Operations {
				if operation.Kind == mir.OperationCCallback {
					targets[e.analysis.bySymbol[operation.Symbol]] = true
				}
			}
		}
	}
	var result []int
	for index := range e.analysis.functions {
		if !targets[index] {
			continue
		}
		result = append(result, index)
	}
	return result
}
