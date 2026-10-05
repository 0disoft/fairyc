package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func storeField(locals map[mir.LocalID]runtimeValue, op mir.Operation, structs map[hir.Type]*mir.StructType) error {
	if len(op.Operands) != 2 {
		return fmt.Errorf("invalid field store operand count")
	}
	receiver, err := evaluate(locals, op.Operands[0])
	if err != nil {
		return err
	}
	value, err := evaluate(locals, op.Operands[1])
	if err != nil {
		return err
	}
	if receiver.typ != op.OwnerType {
		return fmt.Errorf("field store receiver type mismatch")
	}
	if mir.ValidFieldStoreShape(op) {
		if !scalarStoreValueMatches(value, op.Type) {
			return fmt.Errorf("field store scalar type mismatch")
		}
	} else if _, _, array := hir.FixedArrayParts(op.Type); array && mir.ValidPlainStructFieldStoreShape(op, func(typ hir.Type) *mir.StructType { return structs[typ] }) {
		element, length, _ := hir.FixedArrayParts(op.Type)
		if value.kind != valueArray || value.typ != op.Type || value.array == nil || uint64(len(value.array.elements)) != length {
			return fmt.Errorf("invalid array field store")
		}
		for _, item := range value.array.elements {
			if !indexedElementRuntimeMatches(item, element) {
				return fmt.Errorf("invalid array field element")
			}
		}
	} else {
		if value.kind != valueStruct || value.typ != op.Type || value.structure == nil ||
			(!mir.ValidOneWordStructFieldStoreShape(op, value.structure.definition) &&
				!mir.ValidPlainStructFieldStoreShape(op, func(typ hir.Type) *mir.StructType { return structs[typ] })) {
			return fmt.Errorf("invalid struct field store")
		}
		if mir.ValidOneWordStructFieldStoreShape(op, value.structure.definition) {
			field := value.structure.definition.Fields[0]
			if raw, found := value.structure.fields[field.Name]; !found || !scalarStoreValueMatches(raw, hir.TypeUSize) {
				return fmt.Errorf("invalid one-word struct field value")
			}
		}
	}
	if len(op.Fields) > 0 {
		current := receiver
		for _, name := range op.Fields {
			if current.structure == nil || current.structure.definition == nil {
				return fmt.Errorf("missing live nested field store owner")
			}
			child, found := current.structure.fields[name]
			if !found || child.structure == nil {
				return fmt.Errorf("invalid live nested field store path")
			}
			current = child
		}
		if current.structure == nil || current.structure.definition == nil {
			return fmt.Errorf("missing nested leaf owner")
		}
		for _, field := range current.structure.definition.Fields {
			if field.Name == op.Field && field.Type == op.Type {
				current.structure.fields[op.Field] = value.cloneForValueCopy()
				return nil
			}
		}
		return fmt.Errorf("nested field store leaf type mismatch")
	}
	aggregate := fieldAggregateType(op.OwnerType)
	if receiver.structure != nil && receiver.structure.definition != nil && receiver.structure.definition.Type == aggregate {
		for _, field := range receiver.structure.definition.Fields {
			if field.Name == op.Field && field.Type == op.Type {
				receiver.structure.fields[op.Field] = value.cloneForValueCopy()
				return nil
			}
		}
	} else if receiver.resource != nil && receiver.resource.definition != nil && receiver.resource.definition.Type == aggregate && !receiver.resource.dropped {
		for _, field := range receiver.resource.definition.Fields {
			if field.Name == op.Field && field.Type == op.Type {
				receiver.resource.fields[op.Field] = value.cloneForValueCopy()
				return nil
			}
		}
	}
	return fmt.Errorf("field store does not match a live declared field")
}

func scalarStoreValueMatches(value runtimeValue, typ hir.Type) bool {
	if typ == hir.TypeI64 {
		return comparisonRuntimeMatches(value, typ)
	}
	return typ == hir.TypeI32 && value.kind == valueI32 && (value.typ == "" || value.typ == hir.TypeI32) ||
		typ == hir.TypeBool && value.kind == valueI32 && value.typ == hir.TypeBool && (value.i32 == 0 || value.i32 == 1) ||
		typ == hir.TypeUSize && value.kind == valueUSize && value.typ == hir.TypeUSize ||
		typ == hir.TypeU8 && value.kind == valueU8 && value.typ == hir.TypeU8 && value.u64 <= 255 ||
		typ == hir.TypeU32 && value.kind == valueU32 && value.typ == hir.TypeU32 && value.u64 <= 4294967295
}
