package mir

import "github.com/0disoft/fairyc/internal/hir"

// Cache per-function type facts: shared Result branches must not expand a DAG
// into exponentially many ownership visits, including for inert payloads.
func (v *operationVerifier) payloadNeedsDrop(value hir.Type) bool {
	if v.payloadOwnership == nil {
		v.payloadOwnership = make(map[hir.Type]bool)
	}
	if owned, found := v.payloadOwnership[value]; found {
		return owned
	}
	owned := v.resourceTypes[value] != nil
	if !owned {
		if element, option := hir.OptionElement(value); option {
			owned = v.payloadNeedsDrop(element)
		} else if success, failure, result := hir.ResultElements(value); result {
			owned = v.payloadNeedsDrop(success) || v.payloadNeedsDrop(failure)
		}
	}
	v.payloadOwnership[value] = owned
	return owned
}

func (v *operationVerifier) verifyOwningPayload(op Operation, entity string) []VerificationError {
	if (op.Kind != OperationOptionPayload && op.Kind != OperationResultPayload) || len(op.Operands) != 1 || op.Type == hir.TypeInvalid || op.Type == (hir.Type{}) {
		return nil
	}
	container := fieldStoreOperandType(op.Operands[0], v.storeTypes)
	expected := hir.TypeInvalid
	if op.Kind == OperationOptionPayload {
		if element, ok := hir.OptionElement(container); ok {
			expected = element
		}
	} else if success, failure, ok := hir.ResultElements(container); ok {
		expected = success
		if op.Variant == "err" {
			expected = failure
		}
	}
	if expected == hir.TypeInvalid || op.Type != expected || op.MovePayload != v.payloadNeedsDrop(expected) {
		return []VerificationError{{Code: CodeOperationType, Entity: entity, Detail: "tagged payload type or ownership transfer does not match its container"}}
	}
	return nil
}
