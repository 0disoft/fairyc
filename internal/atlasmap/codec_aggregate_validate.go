package atlasmap

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func validateAggregateSpecialization(item semanticsnapshot.SpecializationSummary, source semanticsnapshot.Declaration, instance string, files map[string]struct{}, keys map[semanticsnapshot.DeclarationKey]struct{}, instances map[string]string) error {
	if item.Aggregate == nil || !reflect.DeepEqual(item.Function, semanticsnapshot.FunctionSummary{}) ||
		(item.Declaration.Kind != semanticsnapshot.DeclarationStruct && item.Declaration.Kind != semanticsnapshot.DeclarationResource) {
		return fmt.Errorf("aggregate specialization has incompatible summary kind")
	}
	actual := *item.Aggregate
	if actual.Key != source.Key || actual.Provenance != semanticsnapshot.ProvenanceCompilerObserved || len(actual.TypeParameters) != 0 {
		return fmt.Errorf("aggregate specialization has invalid ownership or provenance")
	}
	expected := cloneDeclarations([]semanticsnapshot.Declaration{source})[0]
	expected.TypeParameters = nil
	expected.Provenance = semanticsnapshot.ProvenanceCompilerObserved
	var err error
	expected.Type, err = bindAggregateType(expected.Type, item.TypeArguments)
	if err != nil {
		return err
	}
	if len(actual.Fields) != len(expected.Fields) {
		return fmt.Errorf("aggregate field arity mismatch")
	}
	for i := range expected.Fields {
		expected.Fields[i].Type, err = bindAggregateType(expected.Fields[i].Type, item.TypeArguments)
		if err != nil {
			return err
		}
		if !aggregateSpanContains(actual.Fields[i].Span, expected.Fields[i].Span) || !aggregateSpanContains(source.Span, actual.Fields[i].Span) {
			return fmt.Errorf("aggregate field lost its source span")
		}
		expected.Fields[i].Span = actual.Fields[i].Span
	}
	if item.Declaration.Kind == semanticsnapshot.DeclarationStruct {
		if item.Cleanup != nil || actual.DropKind != "" || len(actual.DropEffects) != 0 {
			return fmt.Errorf("struct carries resource cleanup metadata")
		}
	} else {
		if actual.DropKind != "empty" && actual.DropKind != "non-empty" {
			return fmt.Errorf("resource has invalid drop kind")
		}
		if (actual.DropKind == "non-empty") != (item.Cleanup != nil) {
			return fmt.Errorf("resource drop kind contradicts its cleanup summary")
		}
		if source.DropKind == "" && item.Cleanup != nil {
			return fmt.Errorf("cleanup summary has no source declaration")
		}
		if err := validateSortedText("aggregate drop effects", actual.DropEffects, false); err != nil {
			return err
		}
		expected.DropKind, expected.DropEffects = actual.DropKind, actual.DropEffects
	}
	if !reflect.DeepEqual(declarationToTransport(expected), declarationToTransport(actual)) {
		return fmt.Errorf("aggregate specialization contradicts its source declaration")
	}
	if item.Cleanup != nil {
		if item.Cleanup.Function != item.Declaration {
			return fmt.Errorf("cleanup belongs to a different resource")
		}
		if err := validateFunction(*item.Cleanup, instance, files, keys, instances, true); err != nil {
			return err
		}
		spans := append([]semanticsnapshot.Span(nil), item.Cleanup.UnsafeSites...)
		spans = append(spans, item.Cleanup.MoveSites...)
		spans = append(spans, item.Cleanup.DropSites...)
		spans = append(spans, item.Cleanup.AllocationSites...)
		spans = append(spans, item.Cleanup.ReturnErrorSites...)
		for _, call := range item.Cleanup.Calls {
			spans = append(spans, call.Span)
		}
		for _, borrow := range item.Cleanup.Borrows {
			spans = append(spans, borrow.Span)
		}
		for _, span := range spans {
			if !aggregateSpanContains(source.Span, span) {
				return fmt.Errorf("cleanup observation is outside its resource")
			}
		}
	}
	return nil
}

func bindAggregateType(value semanticsnapshot.TypeRef, arguments []semanticsnapshot.TypeRef) (semanticsnapshot.TypeRef, error) {
	if value.Kind == semanticsnapshot.TypeParameter {
		index, err := strconv.Atoi(value.Name)
		if err != nil || index < 0 || index >= len(arguments) {
			return value, fmt.Errorf("unbound aggregate type parameter")
		}
		return cloneTypeRef(arguments[index]), nil
	}
	value = cloneTypeRef(value)
	for i := range value.Arguments {
		child, err := bindAggregateType(value.Arguments[i], arguments)
		if err != nil {
			return value, err
		}
		value.Arguments[i] = child
	}
	return value, nil
}

func aggregateSpanContains(outer, inner semanticsnapshot.Span) bool {
	return outer.PackageInstance == inner.PackageInstance && outer.Source == inner.Source && outer.StartByte <= inner.StartByte && inner.EndByte <= outer.EndByte
}
