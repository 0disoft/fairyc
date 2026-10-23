package cbackend

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func (e *emissionAnalysis) fieldStoreEnum(value hir.Type) *mir.EnumType {
	if index, found := e.enumByType[value]; found {
		return &e.enumTypes[index].definition
	}
	return nil
}

func (e *emissionAnalysis) cEnumBase(value hir.Type) string {
	if definition := e.fieldStoreEnum(value); definition != nil {
		return definition.CBase
	}
	return ""
}

func (e *emissionAnalysis) reprCArrayField(value hir.Type) bool {
	return hir.ReprCArrayField(value) || hir.CEnumArray(value, e.cEnumBase)
}

func (r *aggregateLayoutResolver) cEnumBase(value hir.Type) string {
	return r.enumerations[value].CBase
}
