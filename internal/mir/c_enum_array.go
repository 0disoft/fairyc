package mir

import "github.com/0disoft/fairyc/internal/hir"

func (p typeProperties) cEnumBase(value hir.Type) string {
	if definition := p.Enum(value); definition != nil {
		return definition.CBase
	}
	return ""
}

func (p typeProperties) transportOnlyArray(value hir.Type) bool {
	return hir.TransportOnlyArray(value) || hir.CEnumArray(value, p.cEnumBase)
}

// ValidCEnumArrayFieldStoreShape admits only direct whole-field replacement in C records.
func ValidCEnumArrayFieldStoreShape(op Operation, structure func(hir.Type) *StructType, enumeration func(hir.Type) *EnumType) bool {
	if !validFieldStoreCommonShape(op) || len(op.Fields) != 0 || op.Operands[1].Kind != OperandLocal || op.Operands[1].Local < 0 {
		return false
	}
	owner := structure(fieldAggregateType(op.OwnerType))
	return owner != nil && owner.ReprC && hir.CEnumArray(op.Type, func(value hir.Type) string {
		if definition := enumeration(value); definition != nil {
			return definition.CBase
		}
		return ""
	})
}
