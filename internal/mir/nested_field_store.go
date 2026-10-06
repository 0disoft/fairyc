package mir

import "github.com/0disoft/fairyc/internal/hir"

// ResolveNestedFieldStore validates a scalar or plain-struct leaf path through plain nested
// structs. Fields holds ancestor names in order, Field holds the final leaf.
// Definitions and the actual receiver/value locals remain verifier-owned facts.
func ResolveNestedFieldStore(op Operation, lookup func(hir.Type) *StructType) ([]StructField, bool) {
	if len(op.Fields) == 0 || (!ValidFieldStoreShape(op) && !ValidPlainStructFieldStoreShape(op, lookup)) {
		return nil, false
	}
	root := fieldAggregateType(op.OwnerType)
	if !plainStoredStruct(root, lookup) {
		return nil, false
	}
	path := make([]StructField, 0, len(op.Fields)+1)
	typ := root
	for _, name := range append(append([]string(nil), op.Fields...), op.Field) {
		info := lookup(typ)
		if info == nil {
			return nil, false
		}
		field, ok := structField(info.Fields, name)
		if !ok {
			return nil, false
		}
		path = append(path, field)
		typ = field.Type
	}
	return path, typ == op.Type
}
