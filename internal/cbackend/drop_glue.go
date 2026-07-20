package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func (e *emitter) emitDropGlue(output *strings.Builder, functionIndex int, operation mir.Operation) error {
	resource, found := e.resourceLocals[functionIndex][operation.Symbol]
	if !found {
		return fmt.Errorf("drop glue names undeclared resource local %d", operation.Symbol)
	}
	expression := fmt.Sprintf("fy_l_%d", operation.Symbol)
	if _, direct := e.resourceByType[resource.Type]; direct {
		return e.emitDirectDropGlue(output, resource.Type, expression, "")
	}
	tagged, found := e.taggedType(resource.Type)
	if !found {
		return fmt.Errorf("drop glue has unsupported resource type %q", resource.Type)
	}
	switch tagged.kind {
	case taggedOption:
		if len(tagged.payloads) != 1 {
			return fmt.Errorf("drop glue Option %q has %d payloads", resource.Type, len(tagged.payloads))
		}
		condition := expression + ".tag == FY_OPTION_SOME"
		return e.emitDirectDropGlue(output, tagged.payloads[0], expression+".p_0", condition)
	case taggedResult:
		if len(tagged.payloads) != 2 {
			return fmt.Errorf("drop glue Result %q has %d payloads", resource.Type, len(tagged.payloads))
		}
		if err := e.emitDirectDropGlue(output, tagged.payloads[0], expression+".p_0", expression+".tag == FY_RESULT_OK"); err != nil {
			return err
		}
		return e.emitDirectDropGlue(output, tagged.payloads[1], expression+".p_1", expression+".tag == FY_RESULT_ERR")
	default:
		return fmt.Errorf("drop glue has unsupported tagged kind %q", tagged.kind)
	}
}

func (e *emitter) emitDirectDropGlue(output *strings.Builder, resourceType hir.Type, expression, condition string) error {
	resourceIndex, found := e.resourceByType[resourceType]
	if !found {
		return nil
	}
	definition := e.resourceTypes[resourceIndex].definition
	if definition.DropKind == hir.ResourceDropEmpty {
		return nil
	}
	functionIndex, found := e.bySymbol[definition.DropSymbol]
	if !found {
		return fmt.Errorf("resource %q names missing drop symbol %d", resourceType, definition.DropSymbol)
	}
	if condition == "" {
		fmt.Fprintf(output, "    fy_fn_%d(%s);\n", functionIndex, expression)
		return nil
	}
	fmt.Fprintf(output, "    if (%s) { fy_fn_%d(%s); }\n", condition, functionIndex, expression)
	return nil
}
