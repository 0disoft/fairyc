package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/mir"
)

func storeScalarDeref(locals map[mir.LocalID]runtimeValue, op mir.Operation) error {
	if !mir.ValidDerefStoreShape(op) {
		return fmt.Errorf("invalid scalar dereference store shape")
	}
	reference, err := evaluate(locals, op.Operands[0])
	if err != nil {
		return err
	}
	value, err := evaluate(locals, op.Operands[1])
	if err != nil {
		return err
	}
	if reference.typ != op.OwnerType || reference.reference == nil || !scalarStoreValueMatches(value, op.Type) {
		return fmt.Errorf("scalar dereference store type or reference mismatch")
	}
	referent, exists := reference.reference.locals[reference.reference.local]
	if !exists || !scalarStoreValueMatches(referent, op.Type) {
		return fmt.Errorf("scalar dereference store has invalid referent")
	}
	reference.reference.locals[reference.reference.local] = value
	return nil
}
