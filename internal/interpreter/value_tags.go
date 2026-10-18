package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func i32Value(value int32) runtimeValue {
	return runtimeValue{kind: valueI32, scalar: i32Bits(value)}
}

func resultValue(variant string, payload runtimeValue) (runtimeValue, error) {
	if variant != "ok" && variant != "err" {
		return runtimeValue{}, fmt.Errorf("invalid Result variant %q", variant)
	}
	copy := payload.cloneForValueCopy()
	return runtimeValue{kind: valueResult, payload: &taggedState{variant: variant, payload: &copy}}, nil
}

func optionValue(variant string, payload *runtimeValue) (runtimeValue, error) {
	switch variant {
	case "some":
		if payload == nil {
			return runtimeValue{}, fmt.Errorf("Option::Some has no payload")
		}
		copy := payload.cloneForValueCopy()
		return runtimeValue{kind: valueOption, payload: &taggedState{variant: variant, payload: &copy}}, nil
	case "none":
		if payload != nil {
			return runtimeValue{}, fmt.Errorf("Option::None has a payload")
		}
		return runtimeValue{kind: valueOption, payload: &taggedState{variant: variant}}, nil
	default:
		return runtimeValue{}, fmt.Errorf("invalid Option variant %q", variant)
	}
}

func (v runtimeValue) asI32() (int32, error) {
	if v.kind != valueI32 {
		return 0, capabilityError("value is not an executable i32 scalar")
	}
	return v.i32(), nil
}

func (v runtimeValue) resultPayload(expected string, move bool) (runtimeValue, error) {
	if v.kind != valueResult || v.tagged() == nil || v.tagged().payload == nil || v.tagged().dropped {
		return runtimeValue{}, capabilityError("value is not an executable Result")
	}
	if v.tagged().variant != expected {
		return runtimeValue{}, fmt.Errorf("Result payload expects %q, got %q", expected, v.tagged().variant)
	}
	payload := *v.tagged().payload
	if move {
		v.tagged().payload = nil
	}
	return payload, nil
}

func (v runtimeValue) optionPayload(move bool) (runtimeValue, error) {
	if v.kind != valueOption || v.tagged() == nil || v.tagged().variant != "some" || v.tagged().payload == nil || v.tagged().dropped {
		return runtimeValue{}, capabilityError("value is not an executable Option::Some")
	}
	payload := *v.tagged().payload
	if move {
		v.tagged().payload = nil
	}
	return payload, nil
}

func fieldAggregateType(value hir.Type) hir.Type {
	if element, _, borrow := hir.BorrowElement(value); borrow {
		return element
	}
	return value
}
