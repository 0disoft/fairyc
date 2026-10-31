package semantic

import "github.com/0disoft/fairyc/internal/hir"

func (a *analyzer) cRecordArray(value hir.Type) bool {
	return hir.CRecordArray(value, func(typ hir.Type) ([]hir.Type, bool) {
		info := a.structTypes[typ]
		if info == nil || !info.reprC {
			return nil, false
		}
		return a.plainStructFields(typ)
	}, a.cEnumBase)
}
