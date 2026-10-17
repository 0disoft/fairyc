package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"strings"
)

func (e *emissionAnalysis) structCFieldName(structIndex, fieldIndex int) string {
	definition := e.structTypes[structIndex].definition
	if definition.ReprC {
		return definition.Fields[fieldIndex].Name
	}
	return fmt.Sprintf("f_%d", fieldIndex)
}

func (e *emitter) emitPrototype(output *strings.Builder, functionIndex int) error {
	function := &e.analysis.functions[functionIndex]
	resultType, err := e.analysis.emittedFunctionResultType(function)
	if err != nil {
		return fmt.Errorf("prototype for %s: %w", function.Name, err)
	}
	output.WriteString("static ")
	if e.analysis.nonReturning[function.Symbol] {
		output.WriteString("_Noreturn ")
	}
	fmt.Fprintf(output, "%s fy_fn_%d(", resultType, functionIndex)
	if err := e.emitParameters(output, function.Parameters, function.ParameterTypes); err != nil {
		return fmt.Errorf("prototype for %s: %w", function.Name, err)
	}
	output.WriteString(");\n")
	return nil
}

func (e *emitter) emitForeignPrototype(output *strings.Builder, functionIndex int) error {
	function := &e.analysis.foreignFunctions[functionIndex]
	resultType, err := e.analysis.foreignCType(function.ResultType, true)
	if err != nil {
		return fmt.Errorf("foreign prototype for %s: %w", function.Name, err)
	}
	fmt.Fprintf(output, "extern %s %s(", resultType, function.LinkName)
	if len(function.ParameterTypes) == 0 {
		output.WriteString("void")
	} else {
		for index, parameterType := range function.ParameterTypes {
			if index != 0 {
				output.WriteString(", ")
			}
			cType, typeErr := e.analysis.foreignCType(parameterType, false)
			if typeErr != nil {
				return fmt.Errorf("foreign prototype for %s: %w", function.Name, typeErr)
			}
			fmt.Fprintf(output, "%s fy_a_%d", cType, index)
		}
	}
	output.WriteString(");\n")
	return nil
}

func (e *emitter) emitForeignMacroGuard(output *strings.Builder, functionIndex int) {
	linkName := e.analysis.foreignFunctions[functionIndex].LinkName
	fmt.Fprintf(output, "#ifdef %s\n#undef %s\n#endif\n", linkName, linkName)
}

func (e *emitter) emitParameters(output *strings.Builder, parameters []mir.LocalID, parameterTypes []hir.Type) error {
	if len(parameters) != len(parameterTypes) {
		return fmt.Errorf("parameter type count %d does not match parameter count %d", len(parameterTypes), len(parameters))
	}
	if len(parameters) == 0 {
		output.WriteString("void")
		return nil
	}
	for index, parameter := range parameters {
		if index != 0 {
			output.WriteString(", ")
		}
		cType, err := e.analysis.cValueType(parameterTypes[index])
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "%s fy_l_%d", cType, parameter)
	}
	return nil
}
