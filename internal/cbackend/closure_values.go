package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
)

// This is a private transport description, not a source layout or FFI promise.
func closureValueLayout() layout.Type {
	i32 := layout.Type{Kind: layout.KindLanguageScalar, Name: "i32"}
	return layout.Type{Kind: layout.KindStruct, Fields: []layout.Field{
		{Name: "invoke", Type: layout.Type{Kind: layout.KindFunctionPtr}},
		{Name: "captures", Type: layout.Type{Kind: layout.KindArray, Element: &i32, Length: 4}},
	}}
}

func structHasClosureField(definition mir.StructType) bool {
	for _, field := range definition.Fields {
		if hir.IsClosureI32(field.Type) {
			return true
		}
	}
	return false
}

// Closure-bearing records must not inherit the existing tagged aggregate lane.
// Return the original index without allocation when no new leaf is present.
func closureFreeStructIndex(structures []structCType, byType map[hir.Type]int) map[hir.Type]int {
	for _, structure := range structures {
		if !structHasClosureField(structure.definition) {
			continue
		}
		filtered := make(map[hir.Type]int, len(byType))
		for typ, index := range byType {
			if !structHasClosureField(structures[index].definition) {
				filtered[typ] = index
			}
		}
		return filtered
	}
	return byType
}

func (e *analysisBuilder) inspectClosureCreate(fi int, op mir.Operation) error {
	index, found := e.bySymbol[op.Symbol]
	if !found || op.Type != hir.TypeClosureI32 || op.DestinationLocal < 0 || len(op.Operands) > 4 {
		return fmt.Errorf("invalid closure construction in %s", e.functions[fi].Name)
	}
	target := e.functions[index]
	if !target.ClosureTarget || int(target.ClosureCaptures) != len(op.Operands) || target.ResultType != hir.TypeI32 || len(target.ParameterTypes) != len(op.Operands)+1 {
		return fmt.Errorf("invalid closure target %s", target.Name)
	}
	for _, typ := range target.ParameterTypes {
		if typ != hir.TypeI32 {
			return fmt.Errorf("invalid closure target parameter in %s", target.Name)
		}
	}
	for _, operand := range op.Operands {
		typ, err := e.operandType(fi, operand)
		if err != nil || typ != hir.TypeI32 {
			return fmt.Errorf("invalid closure capture in %s", e.functions[fi].Name)
		}
	}
	e.features["copied-closure-value"] = struct{}{}
	return nil
}

func (e *analysisBuilder) inspectClosureCall(fi int, term mir.Terminator) error {
	if len(term.Arguments) != 2 || term.Callee != 0 || term.ForeignABI != "" || term.DestinationLocal < 0 || len(term.Successors) != 1 {
		return fmt.Errorf("invalid closure call in %s", e.functions[fi].Name)
	}
	for i, want := range []hir.Type{hir.TypeClosureI32, hir.TypeI32} {
		got, err := e.operandType(fi, term.Arguments[i])
		if err != nil || got != want {
			return fmt.Errorf("invalid closure call operand %d in %s", i, e.functions[fi].Name)
		}
	}
	e.features["copied-closure-value"] = struct{}{}
	return nil
}

func (e *emitter) emitClosureValueType(output *strings.Builder) {
	if _, needed := e.analysis.features["copied-closure-value"]; !needed {
		return
	}
	output.WriteString("typedef struct fy_closure_i32 {\n    int32_t (*invoke)(const int32_t *, int32_t);\n    int32_t captures[4];\n} fy_closure_i32;\n\n")
}

func closureAdapterPrototype(index int) string {
	return fmt.Sprintf("int32_t fy_closure_%d(const int32_t *fy_captures, int32_t fy_argument)", index)
}

func (e *emitter) closureTargets() []int {
	// Avoid introducing map/slice allocations in no-closure emission.
	if _, needed := e.analysis.features["copied-closure-value"]; !needed {
		return nil
	}
	targets := make(map[int]bool)
	for fi, function := range e.analysis.functions {
		for bi, block := range function.Blocks {
			if !e.analysis.reachableBlocks[fi][bi] {
				continue
			}
			for _, op := range block.Operations {
				if op.Kind == mir.OperationClosureCreate {
					targets[e.analysis.bySymbol[op.Symbol]] = true
				}
			}
		}
	}
	var result []int
	for index := range e.analysis.functions {
		if targets[index] {
			result = append(result, index)
		}
	}
	return result
}

func (e *emitter) emitClosureAdapters(output *strings.Builder) {
	for _, index := range e.closureTargets() {
		fn := e.analysis.functions[index]
		fmt.Fprintf(output, "%s%s {\n", e.functionLinkage(), closureAdapterPrototype(index))
		if fn.ClosureCaptures == 0 {
			output.WriteString("    (void)fy_captures;\n")
		}
		fmt.Fprintf(output, "    return fy_fn_%d(", index)
		for i := uint8(0); i < fn.ClosureCaptures; i++ {
			fmt.Fprintf(output, "fy_captures[%d], ", i)
		}
		output.WriteString("fy_argument);\n}\n")
	}
}

func (e *emitter) emitClosureCreate(output *strings.Builder, op mir.Operation) error {
	index, found := e.analysis.bySymbol[op.Symbol]
	if !found {
		return fmt.Errorf("missing closure target %d", op.Symbol)
	}
	captures := [4]string{"INT32_C(0)", "INT32_C(0)", "INT32_C(0)", "INT32_C(0)"}
	if len(op.Operands) > len(captures) {
		return fmt.Errorf("invalid closure capture count")
	}
	for i, operand := range op.Operands {
		value, err := cOperand(operand)
		if err != nil {
			return err
		}
		captures[i] = value
	}
	fmt.Fprintf(output, "    fy_l_%d = (fy_closure_i32){ &fy_closure_%d, { %s } };\n", op.DestinationLocal, index, strings.Join(captures[:], ", "))
	return nil
}

func (e *emitter) emitClosureCall(output *strings.Builder, term mir.Terminator) error {
	if len(term.Arguments) != 2 {
		return fmt.Errorf("invalid closure call operands")
	}
	next, err := oneSuccessor(term)
	if err != nil {
		return err
	}
	callable, err := cOperand(term.Arguments[0])
	if err != nil {
		return err
	}
	argument, err := cOperand(term.Arguments[1])
	if err != nil {
		return err
	}
	// MIR operands are evaluated values. These block-local snapshots also keep
	// callable-before-argument ordering explicit in C and read the callable once.
	output.WriteString("    {\n")
	fmt.Fprintf(output, "        fy_closure_i32 fy_callable = %s;\n        int32_t fy_argument = %s;\n", callable, argument)
	fmt.Fprintf(output, "        fy_l_%d = fy_callable.invoke(fy_callable.captures, fy_argument);\n    }\n    goto fy_b_%d;\n", term.DestinationLocal, next)
	return nil
}
