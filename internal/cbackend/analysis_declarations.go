package cbackend

import (
	"github.com/0disoft/fairyc/internal/hir"
	"strconv"
)

func (e *analysisBuilder) inspectForeignFunctions() error {
	for _, function := range e.foreignFunctions {
		if function.ABI != "C" {
			return CapabilityError{Function: function.Name, Feature: "foreign ABI " + strconv.Quote(function.ABI)}
		}
		if !validExternalCIdentifier(function.LinkName, e.targetID) &&
			!(e.storageTransfers && validVectorRuntimeFunction(function, e.targetID)) {
			return CapabilityError{Function: function.Name, Feature: "external C symbol " + strconv.Quote(function.LinkName)}
		}
		if _, err := e.foreignCType(function.ResultType, true); err != nil {
			return CapabilityError{Function: function.Name, Feature: err.Error()}
		}
		if _, structure := e.structByType[function.ResultType]; structure {
			e.features["foreign-c-struct-result"] = struct{}{}
		}
		e.recordCScalarFeature(function.ResultType)
		for _, parameterType := range function.ParameterTypes {
			if _, err := e.foreignCType(parameterType, false); err != nil {
				return CapabilityError{Function: function.Name, Feature: err.Error()}
			}
			e.recordCScalarFeature(parameterType)
		}
	}
	return nil
}

func (e *analysisBuilder) recordCScalarFeature(value hir.Type) {
	if transport, found := cScalarTransportForType(value); found {
		e.features[transport.feature] = struct{}{}
	}
}

func (e *analysisBuilder) recordScalarTaggedFeature(value hir.Type) {
	if isOptionI32(value) {
		e.features["scalar-option"] = struct{}{}
		return
	}
	if isResultI32I32(value) {
		e.features["scalar-result"] = struct{}{}
		return
	}
	if _, direct := copyScalarTaggedAggregateLayout(value); !direct {
		return
	}
	if _, option := hir.OptionElement(value); option {
		e.features["copy-option"] = struct{}{}
	} else {
		e.features["copy-result"] = struct{}{}
	}
}

func (e *analysisBuilder) inspectEnumTypes() error {
	for _, enumeration := range e.enumTypes {
		for _, variant := range enumeration.definition.Variants {
			for _, field := range variant.Fields {
				if _, err := e.enumPayloadCType(field.Type); err != nil {
					return CapabilityError{Feature: "enum type " + strconv.Quote(enumeration.definition.Type.String()) + " variant " + strconv.Quote(variant.Name) + " field " + strconv.Quote(field.Name) + " of type " + strconv.Quote(field.Type.String())}
				}
			}
		}
	}
	return nil
}
