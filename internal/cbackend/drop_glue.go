package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func (e *emitter) emitDropGlue(output *strings.Builder, functionIndex int, operation mir.Operation) error {
	resource, found := e.analysis.resourceLocals[functionIndex][operation.Symbol]
	if !found {
		return fmt.Errorf("drop glue names undeclared resource local %d", operation.Symbol)
	}
	expression := fmt.Sprintf("fy_l_%d", operation.Symbol)
	return e.emitOwnedDropGlue(output, resource.Type, expression, "", operation.Span)
}

func (e *emitter) emitDirectDropGlue(output *strings.Builder, resourceType hir.Type, expression, condition string, span hir.Span) error {
	resourceIndex, found := e.analysis.resourceByType[resourceType]
	if !found {
		return nil
	}
	definition := e.analysis.resourceTypes[resourceIndex].definition
	if condition != "" && e.analysis.resourceHasNestedFields(resourceType) {
		fmt.Fprintf(output, "    if (%s) {\n", condition)
		if err := e.emitDirectDropGlue(output, resourceType, expression, "", span); err != nil {
			return err
		}
		output.WriteString("    }\n")
		return nil
	}
	if definition.DropKind != hir.ResourceDropEmpty {
		functionIndex, found := e.analysis.bySymbol[definition.DropSymbol]
		if !found {
			return fmt.Errorf("resource %q names missing drop symbol %d", resourceType, definition.DropSymbol)
		}
		if e.analysis.resourceHasNestedFields(resourceType) {
			fmt.Fprintf(output, "    %s = fy_fn_%d(%s);\n", expression, functionIndex, expression)
		} else if condition == "" {
			fmt.Fprintf(output, "    fy_fn_%d(%s);\n", functionIndex, expression)
		} else {
			fmt.Fprintf(output, "    if (%s) { fy_fn_%d(%s); }\n", condition, functionIndex, expression)
		}
	}
	for index := len(definition.Fields) - 1; index >= 0; index-- {
		field := definition.Fields[index]
		if _, nested := e.analysis.resourceByType[field.Type]; !nested {
			continue
		}
		if e.analysis.instrumentation == InstrumentationDropTraceV0 {
			if err := e.emitDropTraceCall(output, field.Type, span, ""); err != nil {
				return err
			}
		}
		if err := e.emitDirectDropGlue(output, field.Type, fmt.Sprintf("(%s).f_%d", expression, index), "", span); err != nil {
			return err
		}
	}
	return nil
}
