package cbackend

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
)

func callbackTypeName(typ hir.Type) string {
	if typ == hir.TypeCCallbackI32UserData {
		return "fy_c_callback_i32_userdata"
	}
	return fmt.Sprintf("fy_c_callback_%x", sha256.Sum256([]byte(typ.String())))
}

func (e *emissionAnalysis) callbackCSignature(typ hir.Type, named bool) (string, []string, error) {
	result, parameters, ok := hir.CallbackSignature(typ)
	if !ok {
		return "", nil, fmt.Errorf("invalid callback signature %q", typ)
	}
	resultC := "void"
	var err error
	if result != hir.TypeVoid {
		resultC, err = e.cValueType(result)
		if err != nil {
			return "", nil, err
		}
	}
	args := make([]string, len(parameters))
	for i, parameter := range parameters {
		args[i], err = e.cValueType(parameter)
		if err != nil {
			return "", nil, err
		}
		if named {
			args[i] += fmt.Sprintf(" fy_a_%d", i)
		}
	}
	return resultC, args, nil
}

func (e *emitter) emitCallbackType(output *strings.Builder) error {
	if _, callbacks := e.analysis.features["bounded-c-callback"]; !callbacks {
		return nil
	}
	spec, found := target.Lookup(e.analysis.targetID)
	if !found {
		return fmt.Errorf("missing callback target layout %q", e.analysis.targetID)
	}
	seen := map[hir.Type]bool{}
	var types []hir.Type
	for _, function := range e.analysis.foreignFunctions {
		for _, typ := range function.ParameterTypes {
			if hir.IsCCallback(typ) && !seen[typ] {
				seen[typ] = true
				types = append(types, typ)
			}
		}
	}
	slices.SortFunc(types, func(a, b hir.Type) int { return strings.Compare(a.String(), b.String()) })
	for _, typ := range types {
		result, args, err := e.analysis.callbackCSignature(typ, false)
		if err != nil {
			return err
		}
		name := callbackTypeName(typ)
		fmt.Fprintf(output, "typedef %s (*%s)(%s);\n", result, name, strings.Join(args, ", "))
		fmt.Fprintf(output, "_Static_assert(sizeof(%s) == UINT64_C(%d), \"fairyc callback pointer size\");\n", name, spec.FunctionPointer.SizeBytes)
		fmt.Fprintf(output, "_Static_assert(_Alignof(%s) == UINT64_C(%d), \"fairyc callback pointer alignment\");\n\n", name, spec.FunctionPointer.AlignmentBytes)
	}
	return nil
}

func (e *emitter) callbackPrototype(index int) (string, error) {
	function := e.analysis.functions[index]
	typ := hir.CCallback(function.ResultType, function.ParameterTypes)
	result, args, err := e.analysis.callbackCSignature(typ, true)
	if err != nil {
		return "", err
	}
	prefix := ""
	if e.analysis.nonReturning[function.Symbol] {
		prefix = "_Noreturn "
	}
	return fmt.Sprintf("%s%s fy_cb_%d(%s)", prefix, result, index, strings.Join(args, ", ")), nil
}

// Adapters keep the exact C callback contract separate from private Fairy functions.
func (e *emitter) emitCallbackAdapters(output *strings.Builder) error {
	for _, index := range e.callbackTargets() {
		function := e.analysis.functions[index]
		prototype, err := e.callbackPrototype(index)
		if err != nil {
			return err
		}
		args := make([]string, len(function.ParameterTypes))
		for i := range args {
			args[i] = fmt.Sprintf("fy_a_%d", i)
		}
		result := ""
		if function.ResultType != hir.TypeVoid && !e.analysis.nonReturning[function.Symbol] {
			result = "return "
		}
		fmt.Fprintf(output, "%s%s { %sfy_fn_%d(%s); }\n", e.functionLinkage(), prototype, result, index, strings.Join(args, ", "))
	}
	return nil
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
