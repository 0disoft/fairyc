package atlasmap

import "github.com/0disoft/fairyc/internal/semanticsnapshot"

type transportSpecialization struct {
	ID            string                   `json:"id"`
	Declaration   transportDeclarationKey  `json:"declaration"`
	TypeArguments []transportTypeRef       `json:"type_arguments"`
	Function      transportFunctionSummary `json:"function"`
}

func specializationToTransport(value semanticsnapshot.SpecializationSummary) transportSpecialization {
	result := transportSpecialization{ID: value.ID, Declaration: keyToTransport(value.Declaration), Function: functionToTransport(value.Function), TypeArguments: make([]transportTypeRef, 0, len(value.TypeArguments))}
	for _, argument := range value.TypeArguments {
		result.TypeArguments = append(result.TypeArguments, typeToTransport(argument))
	}
	return result
}

func specializationFromTransport(value transportSpecialization) (semanticsnapshot.SpecializationSummary, error) {
	function, err := functionFromTransport(value.Function)
	if err != nil {
		return semanticsnapshot.SpecializationSummary{}, err
	}
	result := semanticsnapshot.SpecializationSummary{ID: value.ID, Declaration: keyFromTransport(value.Declaration), Function: function}
	for _, argument := range value.TypeArguments {
		result.TypeArguments = append(result.TypeArguments, typeFromTransport(argument))
	}
	return result, nil
}
