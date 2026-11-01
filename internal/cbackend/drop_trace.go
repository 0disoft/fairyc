package cbackend

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

type dropSiteKey struct {
	resourceType hir.Type
	span         hir.Span
}

func (e *analysisBuilder) prepareDropTraceSites() error {
	if e.instrumentation == "" {
		return nil
	}
	if e.instrumentation != InstrumentationDropTraceV0 {
		return fmt.Errorf("unsupported C instrumentation %q", e.instrumentation)
	}
	keys := make(map[dropSiteKey]struct{})
	for functionIndex := range e.functions {
		function := &e.functions[functionIndex]
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if operation.Kind != mir.OperationDrop {
					continue
				}
				resource, found := e.resourceLocals[functionIndex][operation.Symbol]
				if !found {
					return fmt.Errorf("drop trace in %s names undeclared resource local %d", function.Name, operation.Symbol)
				}
				for _, resourceType := range e.dropLeafTypes(resource.Type) {
					keys[dropSiteKey{resourceType: resourceType, span: operation.Span}] = struct{}{}
				}
			}
		}
	}
	ordered := make([]dropSiteKey, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	slices.SortFunc(ordered, compareDropSiteKey)
	if uint64(len(ordered)) > uint64(^uint32(0)) {
		return fmt.Errorf("drop trace has %d sites, exceeding uint32 identity space", len(ordered))
	}
	e.dropSites = make([]DropSite, len(ordered))
	e.dropSiteByKey = make(map[dropSiteKey]uint32, len(ordered))
	for index, key := range ordered {
		id := uint32(index)
		e.dropSiteByKey[key] = id
		e.dropSites[index] = DropSite{
			ID: id, Package: key.span.Package, ResourceType: key.resourceType.String(), Source: key.span.Source,
			StartByte: key.span.StartByte, EndByte: key.span.EndByte,
		}
	}
	return nil
}

func compareDropSiteKey(left, right dropSiteKey) int {
	if compared := strings.Compare(left.resourceType.String(), right.resourceType.String()); compared != 0 {
		return compared
	}
	if compared := cmp.Compare(left.span.Package, right.span.Package); compared != 0 {
		return compared
	}
	if compared := strings.Compare(left.span.Source, right.span.Source); compared != 0 {
		return compared
	}
	if left.span.StartByte != right.span.StartByte {
		return cmp.Compare(left.span.StartByte, right.span.StartByte)
	}
	return cmp.Compare(left.span.EndByte, right.span.EndByte)
}

func (e *emissionAnalysis) dropLeafTypes(value hir.Type) []hir.Type {
	if index, direct := e.resourceByType[value]; direct {
		result := []hir.Type{value}
		fields := e.resourceTypes[index].definition.Fields
		for index := len(fields) - 1; index >= 0; index-- {
			result = append(result, e.dropLeafTypes(fields[index].Type)...)
		}
		return result
	}
	tagged, found := e.taggedType(value)
	if !found {
		return nil
	}
	result := make([]hir.Type, 0, len(tagged.payloads))
	for _, payload := range tagged.payloads {
		result = append(result, e.dropLeafTypes(payload)...)
	}
	return result
}

func (e *emitter) emitDropTrace(output *strings.Builder, functionIndex int, operation mir.Operation) error {
	resource, found := e.analysis.resourceLocals[functionIndex][operation.Symbol]
	if !found {
		return fmt.Errorf("drop trace names undeclared resource local %d", operation.Symbol)
	}
	return e.emitOwnedDropTrace(output, resource.Type, fmt.Sprintf("fy_l_%d", operation.Symbol), "", operation.Span)
}

func (e *emitter) emitDropTraceCall(output *strings.Builder, resourceType hir.Type, span hir.Span, condition string) error {
	id, found := e.analysis.dropSiteByKey[dropSiteKey{resourceType: resourceType, span: span}]
	if !found {
		return fmt.Errorf("drop trace site is missing for %q at %s:%d-%d", resourceType, span.Source, span.StartByte, span.EndByte)
	}
	if condition == "" {
		fmt.Fprintf(output, "    fy_runtime_trace_drop(UINT32_C(%d));\n", id)
		return nil
	}
	fmt.Fprintf(output, "    if (%s) { fy_runtime_trace_drop(UINT32_C(%d)); }\n", condition, id)
	return nil
}
