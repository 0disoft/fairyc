package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// Production helpers are shared by type. Trace helpers also retain the original
// drop span so source-linked trace IDs remain exact at every nested invocation.
func (e *emitter) dropHelperKey(typ hir.Type, span hir.Span) dropSiteKey {
	if e.analysis.instrumentation == "" {
		span = hir.Span{}
	}
	return dropSiteKey{resourceType: typ, span: span}
}

func (e *emitter) prepareDropHelpers() error {
	e.dropHelperIDs = make(map[dropSiteKey]int)
	for functionIndex, function := range e.analysis.functions {
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if operation.Kind != mir.OperationDrop {
					continue
				}
				if err := e.contextError(); err != nil {
					return err
				}
				resource, found := e.analysis.resourceLocals[functionIndex][operation.Symbol]
				if !found {
					return fmt.Errorf("drop helper names undeclared local %d", operation.Symbol)
				}
				for _, typ := range e.analysis.dropLeafTypes(resource.Type) {
					if !e.analysis.resourceHasNestedFields(typ) {
						continue
					}
					key := e.dropHelperKey(typ, operation.Span)
					if _, found := e.dropHelperIDs[key]; !found {
						e.dropHelperIDs[key] = len(e.dropHelpers)
						e.dropHelpers = append(e.dropHelpers, key)
					}
				}
			}
		}
	}
	return nil
}

func (e *emitter) emitDropHelperPrototypes(output *strings.Builder) {
	for index, key := range e.dropHelpers {
		resource := e.analysis.resourceByType[key.resourceType]
		fmt.Fprintf(output, "%svoid fy_drop_%d(fy_resource_%d *fy_value);\n", e.functionLinkage(), index, resource)
	}
}

func (e *emitter) emitDropHelper(output *strings.Builder, index int) error {
	if err := e.contextError(); err != nil {
		return err
	}
	key := e.dropHelpers[index]
	resource := e.analysis.resourceByType[key.resourceType]
	fmt.Fprintf(output, "%svoid fy_drop_%d(fy_resource_%d *fy_value) {\n    (void)fy_value;\n", e.functionLinkage(), index, resource)
	if err := e.emitDirectDropBody(output, key.resourceType, "(*fy_value)", "", key.span); err != nil {
		return err
	}
	output.WriteString("}\n\n")
	return nil
}
