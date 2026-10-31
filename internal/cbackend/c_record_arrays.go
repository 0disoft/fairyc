package cbackend

import "github.com/0disoft/fairyc/internal/hir"

func (e *emissionAnalysis) cRecordArray(value hir.Type) bool {
	return hir.CRecordArray(value, func(typ hir.Type) ([]hir.Type, bool) {
		index, ok := e.structByType[typ]
		if !ok || !e.structTypes[index].definition.ReprC {
			return nil, false
		}
		definition := e.structTypes[index].definition
		fields := make([]hir.Type, len(definition.Fields))
		for i, field := range definition.Fields {
			fields[i] = field.Type
		}
		return fields, true
	}, e.cEnumBase)
}

func (r *aggregateLayoutResolver) cRecordArray(value hir.Type) bool {
	return hir.CRecordArray(value, func(typ hir.Type) ([]hir.Type, bool) {
		definition, ok := r.structures[typ]
		if !ok || !definition.ReprC {
			return nil, false
		}
		fields := make([]hir.Type, len(definition.Fields))
		for i, field := range definition.Fields {
			fields[i] = field.Type
		}
		return fields, true
	}, r.cEnumBase)
}
