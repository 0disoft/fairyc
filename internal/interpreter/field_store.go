package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func storeScalarField(locals map[mir.LocalID]runtimeValue, op mir.Operation) error {
	if !mir.ValidFieldStoreShape(op) {
		return fmt.Errorf("invalid scalar field store shape")
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
	validValue := op.Type == hir.TypeI32 && value.kind == valueI32 && value.typ != hir.TypeBool ||
		op.Type == hir.TypeBool && value.kind == valueI32 && value.typ == hir.TypeBool && (value.i32 == 0 || value.i32 == 1) ||
		op.Type == hir.TypeUSize && value.kind == valueUSize
	if !validValue {
		return fmt.Errorf("field store scalar type mismatch")
	}
	aggregate := fieldAggregateType(op.OwnerType)
	if receiver.structure != nil && receiver.structure.definition != nil && receiver.structure.definition.Type == aggregate {
		for _, field := range receiver.structure.definition.Fields {
			if field.Name == op.Field && field.Type == op.Type {
				receiver.structure.fields[op.Field] = value
				return nil
			}
		}
	} else if receiver.resource != nil && receiver.resource.definition != nil && receiver.resource.definition.Type == aggregate && !receiver.resource.dropped {
		for _, field := range receiver.resource.definition.Fields {
			if field.Name == op.Field && field.Type == op.Type {
				receiver.resource.fields[op.Field] = value
				return nil
			}
		}
	}
	return fmt.Errorf("field store does not match a live declared field")
}
