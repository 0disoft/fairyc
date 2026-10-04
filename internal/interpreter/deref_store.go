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
	referent, exists := reference.reference.load()
	if !exists || !scalarStoreValueMatches(referent, op.Type) {
		return fmt.Errorf("scalar dereference store has invalid referent")
	}
	reference.reference.store(value)
	return nil
}

func (r *scalarReference) load() (runtimeValue, bool) {
	if r.fields != nil {
		v, ok := r.fields[r.field]
		return v, ok
	}
	v, ok := r.locals[r.local]
	return v, ok
}
func (r *scalarReference) store(v runtimeValue) {
	if r.fields != nil {
		r.fields[r.field] = v
	} else {
		r.locals[r.local] = v
	}
}
