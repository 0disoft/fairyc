package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
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
	if reference.typ != op.OwnerType || reference.reference() == nil || !scalarStoreValueMatches(value, op.Type) {
		return fmt.Errorf("scalar dereference store type or reference mismatch")
	}
	referent, exists := reference.reference().load()
	if !exists || !scalarStoreValueMatches(referent, op.Type) {
		return fmt.Errorf("scalar dereference store has invalid referent")
	}
	reference.reference().store(value)
	return nil
}

func (r *scalarReference) load() (runtimeValue, bool) {
	if r.array != nil {
		if r.index >= uint64(len(r.array.elements)) {
			return runtimeValue{}, false
		}
		return r.array.elements[r.index], true
	}
	if r.static != nil {
		if r.index >= uint64(len(r.static.staticData)) {
			return runtimeValue{}, false
		}
		return runtimeValue{kind: valueU8, typ: hir.TypeU8, scalar: uint64(r.static.staticData[r.index])}, true
	}
	if r.fields != nil {
		return indexedField(*r.fields, r.field)
	}
	v, ok := r.locals[r.local]
	return v, ok
}
func (r *scalarReference) store(v runtimeValue) {
	if r.array != nil {
		r.array.elements[r.index] = v
		return
	}
	if r.fields != nil {
		(*r.fields)[r.field] = v
	} else {
		r.locals[r.local] = v
	}
}

// Aggregate references retain the stored structState identity. Replace its fields
// with detached copies so array elements, named owners and returned refs see it.
func storeDeref(locals map[mir.LocalID]runtimeValue, op mir.Operation, structs map[hir.Type]*mir.StructType) error {
	if mir.ValidDerefStoreShape(op) {
		return storeScalarDeref(locals, op)
	}
	if !mir.ValidPlainStructDerefStoreShape(op, func(typ hir.Type) *mir.StructType { return structs[typ] }) {
		return fmt.Errorf("invalid plain struct dereference store")
	}
	receiver, err := evaluate(locals, op.Operands[0])
	if err != nil {
		return err
	}
	value, err := evaluate(locals, op.Operands[1])
	if err != nil {
		return err
	}
	if receiver.typ != op.OwnerType || receiver.kind != valueStruct || receiver.structure() == nil || receiver.structure().definition == nil || receiver.structure().definition.Type != op.Type || value.typ != op.Type || value.kind != valueStruct || value.structure() == nil || value.structure().definition == nil || value.structure().definition.Type != op.Type {
		return fmt.Errorf("plain struct dereference store type or referent mismatch")
	}
	receiver.structure().fields = copyValueFields(value.structure().fields)
	return nil
}
