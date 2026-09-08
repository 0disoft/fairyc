package atlasmap

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

type transportSpecialization struct {
	ID            string                    `json:"id"`
	Declaration   transportDeclarationKey   `json:"declaration"`
	TypeArguments []transportTypeRef        `json:"type_arguments"`
	Function      *transportFunctionSummary `json:"function,omitempty"`
	Aggregate     *transportDeclaration     `json:"aggregate,omitempty"`
	Cleanup       *transportFunctionSummary `json:"cleanup,omitempty"`
}

func specializationToTransport(value semanticsnapshot.SpecializationSummary) transportSpecialization {
	result := transportSpecialization{ID: value.ID, Declaration: keyToTransport(value.Declaration), TypeArguments: make([]transportTypeRef, 0, len(value.TypeArguments))}
	if value.Aggregate == nil {
		function := functionToTransport(value.Function)
		result.Function = &function
	} else {
		aggregate := declarationToTransport(*value.Aggregate)
		result.Aggregate = &aggregate
	}
	if value.Cleanup != nil {
		cleanup := functionToTransport(*value.Cleanup)
		result.Cleanup = &cleanup
	}
	for _, argument := range value.TypeArguments {
		result.TypeArguments = append(result.TypeArguments, typeToTransport(argument))
	}
	return result
}

func specializationFromTransport(value transportSpecialization) (semanticsnapshot.SpecializationSummary, error) {
	if (value.Function == nil) == (value.Aggregate == nil) {
		return semanticsnapshot.SpecializationSummary{}, fmt.Errorf("specialization requires exactly one function or aggregate")
	}
	result := semanticsnapshot.SpecializationSummary{ID: value.ID, Declaration: keyFromTransport(value.Declaration)}
	if value.Function != nil {
		function, err := functionFromTransport(*value.Function)
		if err != nil {
			return result, err
		}
		result.Function = function
	}
	if value.Aggregate != nil {
		aggregate, err := declarationFromTransport(*value.Aggregate)
		if err != nil {
			return result, err
		}
		result.Aggregate = &aggregate
	}
	if value.Cleanup != nil {
		cleanup, err := functionFromTransport(*value.Cleanup)
		if err != nil {
			return result, err
		}
		result.Cleanup = &cleanup
	}
	for _, argument := range value.TypeArguments {
		result.TypeArguments = append(result.TypeArguments, typeFromTransport(argument))
	}
	return result, nil
}
