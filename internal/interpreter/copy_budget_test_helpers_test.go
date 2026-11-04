package interpreter

import (
	"context"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// Existing value-model tests use the production default through a fresh engine.
// Budget-specific tests construct executionState directly.
func defaultCopyTestState() *executionState {
	return &executionState{context: context.Background(), copyUnits: DefaultMaxCopyUnits}
}

func mustCopyValue(value runtimeValue) runtimeValue {
	copy, err := defaultCopyTestState().cloneForValueCopy(value)
	if err != nil {
		panic(err)
	}
	return copy
}

func benchmarkCopyValue(value runtimeValue) runtimeValue {
	return mustCopyValue(value)
}

func (v runtimeValue) cloneForValueCopy() runtimeValue {
	return mustCopyValue(v)
}

func (v runtimeValue) indexedAt(index uint64) (runtimeValue, error) {
	return defaultCopyTestState().indexedAt(v, index)
}

func (v runtimeValue) storeIndexed(index uint64, value runtimeValue) error {
	return defaultCopyTestState().storeIndexed(v, index, value)
}

func (v runtimeValue) borrowViewElement(index uint64, typ hir.Type) (runtimeValue, error) {
	return defaultCopyTestState().borrowViewElement(v, index, typ)
}

func inertDropValue(value runtimeValue, depth int) bool {
	inert, err := defaultCopyTestState().inertDropValue(value, depth)
	if err != nil {
		panic(err)
	}
	return inert
}

func enumValue(definition *mir.EnumType, variant string, fields []string, values []runtimeValue) (runtimeValue, error) {
	return defaultCopyTestState().enumValue(definition, variant, fields, values)
}

func resourceValue(definition *mir.ResourceType, values []runtimeValue) (runtimeValue, error) {
	return defaultCopyTestState().resourceValue(definition, values)
}

func structValue(definition *mir.StructType, values []runtimeValue) (runtimeValue, error) {
	return defaultCopyTestState().structValue(definition, values)
}

func resultValue(variant string, payload runtimeValue) (runtimeValue, error) {
	return defaultCopyTestState().resultValue(variant, payload)
}

func optionValue(variant string, payload *runtimeValue) (runtimeValue, error) {
	return defaultCopyTestState().optionValue(variant, payload)
}

func copyValueFields(fields []runtimeValue) []runtimeValue {
	copy, err := defaultCopyTestState().copyValueFields(fields)
	if err != nil {
		panic(err)
	}
	return copy
}

func evaluateUnary(locals map[mir.LocalID]runtimeValue, operation mir.Operation) (runtimeValue, string, error) {
	return defaultCopyTestState().evaluateUnary(locals, operation)
}
