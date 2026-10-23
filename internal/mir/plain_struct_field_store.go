package mir

import "github.com/0disoft/fairyc/internal/hir"

// ValidPlainStructFieldStoreShape admits copied plain aggregates and direct
// transport-only array fields of repr(C) records. Exact path and local types
// remain verifier-owned facts.
func ValidPlainStructFieldStoreShape(op Operation, lookup func(hir.Type) *StructType) bool {
	if !validFieldStoreCommonShape(op) || op.Operands[1].Kind != OperandLocal || op.Operands[1].Local < 0 {
		return false
	}
	if len(op.Fields) == 0 && hir.TransportOnlyArray(op.Type) {
		owner := lookup(fieldAggregateType(op.OwnerType))
		return owner != nil && owner.ReprC
	}
	return (hir.InlineArrayField(op.Type, plainFieldStoreLookup(lookup)) || plainStoredStruct(op.Type, lookup)) && plainStoredStruct(fieldAggregateType(op.OwnerType), lookup)
}

func plainStoredStruct(typ hir.Type, lookup func(hir.Type) *StructType) bool {
	return hir.PlainStoredStruct(typ, func(value hir.Type) ([]hir.Type, bool) {
		info := lookup(value)
		if info == nil || info.ReprC && len(info.Fields) == 0 {
			return nil, false
		}
		return plainFieldStoreLookup(lookup)(value)
	})
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
