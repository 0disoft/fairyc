package mir

import "github.com/0disoft/fairyc/internal/hir"

// ValidPlainStructFieldStoreShape admits copied plain aggregates only under a
// plain struct root. Exact path and local types remain verifier-owned facts.
func ValidPlainStructFieldStoreShape(op Operation, lookup func(hir.Type) *StructType) bool {
	if !validFieldStoreCommonShape(op) || op.Operands[1].Kind != OperandLocal || op.Operands[1].Local < 0 {
		return false
	}
	return (hir.InlineArrayField(op.Type, plainFieldStoreLookup(lookup)) || plainFieldStoreStruct(op.Type, lookup)) && plainFieldStoreStruct(fieldAggregateType(op.OwnerType), lookup)
}

func plainFieldStoreStruct(typ hir.Type, lookup func(hir.Type) *StructType) bool {
	return hir.PlainViewStruct(typ, plainFieldStoreLookup(lookup))
}

func plainFieldStoreLookup(lookup func(hir.Type) *StructType) func(hir.Type) ([]hir.Type, bool) {
	return func(value hir.Type) ([]hir.Type, bool) {
		info := lookup(value)
		if info == nil {
			return nil, false
		}
		fields := make([]hir.Type, len(info.Fields))
		for i, field := range info.Fields {
			fields[i] = field.Type
		}
		return fields, true
	}
}
