package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func storeField(locals map[mir.LocalID]runtimeValue, op mir.Operation, structs map[hir.Type]*mir.StructType, fieldSlots *aggregateFieldIndex) error {
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
	} else if value.kind == valueEnum && value.typ == op.Type && value.enum() != nil && mir.ValidCEnumFieldStoreShape(op, value.enum().definition) {
		// Keep the nominal enum and copy it into the declared owner field below.
	} else if _, _, array := hir.FixedArrayParts(op.Type); array && mir.ValidPlainStructFieldStoreShape(op, func(typ hir.Type) *mir.StructType { return structs[typ] }) {
		element, length, _ := hir.FixedArrayParts(op.Type)
		if value.kind != valueArray || value.typ != op.Type || value.array() == nil || uint64(len(value.array().elements)) != length {
			return fmt.Errorf("invalid array field store")
		}
		for _, item := range value.array().elements {
			if !indexedElementRuntimeMatches(item, element) {
				return fmt.Errorf("invalid array field element")
			}
		}
	} else {
		if value.kind != valueStruct || value.typ != op.Type || value.structure() == nil ||
			(!mir.ValidOneWordStructFieldStoreShape(op, value.structure().definition) &&
				!mir.ValidPlainStructFieldStoreShape(op, func(typ hir.Type) *mir.StructType { return structs[typ] })) {
			return fmt.Errorf("invalid struct field store")
		}
		if mir.ValidOneWordStructFieldStoreShape(op, value.structure().definition) {
			if raw, found := indexedField(value.structure().fields, 0); !found || !scalarStoreValueMatches(raw, hir.TypeUSize) {
				return fmt.Errorf("invalid one-word struct field value")
			}
		}
	}
	if len(op.Fields) > 0 {
		current := receiver
		for _, name := range op.Fields {
			if current.structure() == nil || current.structure().definition == nil {
				return fmt.Errorf("missing live nested field store owner")
			}
			child, found := indexedField(current.structure().fields, fieldSlots.structField(current.structure().definition, name))
			if !found || child.structure() == nil {
				return fmt.Errorf("invalid live nested field store path")
			}
			current = child
		}
		if current.structure() == nil || current.structure().definition == nil {
			return fmt.Errorf("missing nested leaf owner")
		}
		definition := current.structure().definition
		// Start at the first name match. Continuing on a type mismatch preserves
		// the old first-name-and-type behavior even for malformed duplicates.
		start := 0
		if len(definition.Fields) > linearFieldLookupLimit {
			start = fieldSlots.structField(definition, op.Field)
		}
		for index := start; index >= 0 && index < len(definition.Fields); index++ {
			field := definition.Fields[index]
			if field.Name == op.Field && field.Type == op.Type {
				current.structure().fields[index] = value.cloneForValueCopy()
				return nil
			}
		}
		return fmt.Errorf("nested field store leaf type mismatch")
	}
	aggregate := fieldAggregateType(op.OwnerType)
	if receiver.structure() != nil && receiver.structure().definition != nil && receiver.structure().definition.Type == aggregate {
		definition := receiver.structure().definition
		start := 0
		if len(definition.Fields) > linearFieldLookupLimit {
			start = fieldSlots.structField(definition, op.Field)
		}
		for index := start; index >= 0 && index < len(definition.Fields); index++ {
			field := definition.Fields[index]
			if field.Name == op.Field && field.Type == op.Type {
				receiver.structure().fields[index] = value.cloneForValueCopy()
				return nil
			}
		}
	} else if receiver.resource() != nil && receiver.resource().definition != nil && receiver.resource().definition.Type == aggregate && !receiver.resource().dropped {
		definition := receiver.resource().definition
		start := 0
		if len(definition.Fields) > linearFieldLookupLimit {
			start = fieldSlots.resourceField(definition, op.Field)
		}
		for index := start; index >= 0 && index < len(definition.Fields); index++ {
			field := definition.Fields[index]
			if field.Name == op.Field && field.Type == op.Type {
				receiver.resource().fields[index] = value.cloneForValueCopy()
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
	return typ == hir.TypeI32 && value.kind == valueI32 && (value.typ == hir.ParseType("") || value.typ == hir.TypeI32) ||
		typ == hir.TypeBool && value.kind == valueI32 && value.typ == hir.TypeBool && (value.i32() == 0 || value.i32() == 1) ||
		typ == hir.TypeUSize && value.kind == valueUSize && value.typ == hir.TypeUSize ||
		typ == hir.TypeU8 && value.kind == valueU8 && value.typ == hir.TypeU8 && value.u64() <= 255 ||
		typ == hir.TypeU32 && value.kind == valueU32 && value.typ == hir.TypeU32 && value.u64() <= 4294967295
}
