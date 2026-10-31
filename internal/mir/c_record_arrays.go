package mir

import "github.com/0disoft/fairyc/internal/hir"

func cRecordFields(lookup func(hir.Type) *StructType) func(hir.Type) ([]hir.Type, bool) {
	return func(typ hir.Type) ([]hir.Type, bool) {
		info := lookup(typ)
		if info == nil || !info.ReprC {
			return nil, false
		}
		fields := make([]hir.Type, len(info.Fields))
		for i, field := range info.Fields {
			fields[i] = field.Type
		}
		return fields, true
	}
}

func (p typeProperties) cRecordArray(value hir.Type) bool {
	return hir.CRecordArray(value, cRecordFields(p.Struct), p.cEnumBase)
}

// ValidCRecordArrayFieldStoreShape retains existing executable plain-array stores
// and admits direct replacement of the new whole-value record-array fields.
func ValidCRecordArrayFieldStoreShape(op Operation, structure func(hir.Type) *StructType, enumeration func(hir.Type) *EnumType) bool {
	if !validFieldStoreCommonShape(op) || len(op.Fields) != 0 || op.Operands[1].Kind != OperandLocal || op.Operands[1].Local < 0 {
		return false
	}
	owner := structure(fieldAggregateType(op.OwnerType))
	return owner != nil && owner.ReprC && hir.CRecordArray(op.Type, cRecordFields(structure), func(typ hir.Type) string {
		if info := enumeration(typ); info != nil {
			return info.CBase
		}
		return ""
	})
}
