package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

type flagState struct {
	initialized bool
	active      bool
}

type verificationState struct {
	live    map[hir.SymbolID]bool
	flags   map[string]flagState
	regions map[hir.SymbolID]string
}

func verifyOperands(state verificationState, operands []Operand, entity string) []VerificationError {
	var failures []VerificationError
	for _, operand := range operands {
		switch operand.Kind {
		case OperandInvalid:
		case OperandLocal:
			if operand.Local < 0 || !state.live[hir.SymbolID(operand.Local)] {
				failures = append(failures, VerificationError{Code: "V1002", Entity: entity, Detail: fmt.Sprintf("local %d used before initialization", operand.Local)})
			}
		case OperandI32:
			if operand.Type != hir.ParseType("") && operand.Type != hir.TypeI32 && operand.Type != hir.TypeBool ||
				operand.Type == hir.TypeBool && operand.I32 != 0 && operand.I32 != 1 {
				failures = append(failures, VerificationError{Code: CodeOperandShape, Entity: entity, Detail: "i32 operand has invalid scalar type metadata"})
			}
		case OperandI64:
			if operand.Type != hir.ParseType("") && operand.Type != hir.TypeI64 || operand.I32 != 0 || operand.U64 != 0 || operand.StaticData != "" {
				failures = append(failures, VerificationError{Code: CodeOperandShape, Entity: entity, Detail: "i64 operand has invalid scalar metadata"})
			}
		case OperandU8:
			if operand.U64 > 255 {
				failures = append(failures, VerificationError{Code: CodeOperandShape, Entity: entity, Detail: "u8 operand is outside its value range"})
			}
		case OperandU32:
			if operand.U64 > uint64(^uint32(0)) {
				failures = append(failures, VerificationError{Code: CodeOperandShape, Entity: entity, Detail: "u32 operand is outside its value range"})
			}
		case OperandUSize:
		case OperandNull:
			if _, _, pointer := hir.RawPointerElement(operand.Type); !pointer {
				failures = append(failures, VerificationError{Code: CodeOperandShape, Entity: entity, Detail: "null operand has no raw-pointer type"})
			}
		case OperandByteString:
			if operand.Type != hir.SharedBorrow(hir.TypeByteSlice) {
				failures = append(failures, VerificationError{Code: CodeOperandShape, Entity: entity, Detail: "byte-string operand has invalid static storage"})
			}
		case OperandString:
			if operand.Type != hir.SharedBorrow(hir.TypeStr) || !validUTF8StringStaticData(operand.StaticData) {
				failures = append(failures, VerificationError{Code: CodeOperandShape, Entity: entity, Detail: "string operand has invalid static storage"})
			}
		case OperandCString:
			if operand.Type != hir.SharedBorrow(hir.TypeCStr) || !validCStringStaticData(operand.StaticData) {
				failures = append(failures, VerificationError{Code: CodeOperandShape, Entity: entity, Detail: "C string operand has invalid static storage"})
			}
		default:
			failures = append(failures, VerificationError{Code: CodeOperandShape, Entity: entity, Detail: fmt.Sprintf("operand has unknown kind %q", operand.Kind)})
		}
	}
	return failures
}

func cloneVerificationState(input verificationState) verificationState {
	result := verificationState{
		live:    cloneLive(input.live),
		flags:   make(map[string]flagState, len(input.flags)),
		regions: make(map[hir.SymbolID]string, len(input.regions)),
	}
	for entity, state := range input.flags {
		result.flags[entity] = state
	}
	for symbol, entity := range input.regions {
		result.regions[symbol] = entity
	}
	return result
}

func mergeVerificationState(previous, incoming verificationState, resources map[hir.SymbolID]ResourceLocal) (verificationState, bool, bool) {
	if len(previous.flags) != len(incoming.flags) || len(previous.regions) != len(incoming.regions) {
		return verificationState{}, false, true
	}
	for entity, state := range previous.flags {
		if incoming.flags[entity] != state {
			return verificationState{}, false, true
		}
	}
	for symbol, entity := range previous.regions {
		if incoming.regions[symbol] != entity {
			return verificationState{}, false, true
		}
	}
	merged := cloneVerificationState(previous)
	changed := false
	keys := make(map[hir.SymbolID]struct{}, len(previous.live)+len(incoming.live))
	for symbol := range previous.live {
		keys[symbol] = struct{}{}
	}
	for symbol := range incoming.live {
		keys[symbol] = struct{}{}
	}
	for symbol := range keys {
		left := previous.live[symbol]
		right := incoming.live[symbol]
		if _, resource := resources[symbol]; resource {
			if left != right {
				return verificationState{}, false, true
			}
			continue
		}
		value := left && right
		if merged.live[symbol] != value {
			merged.live[symbol] = value
			changed = true
		}
	}
	return merged, changed, false
}

func sameLiveState(left, right map[hir.SymbolID]bool) bool {
	for symbol, live := range left {
		if right[symbol] != live {
			return false
		}
	}
	for symbol, live := range right {
		if left[symbol] != live {
			return false
		}
	}
	return true
}

func blockEntityFor(function *Function, id BlockID) string {
	if int(id) >= 0 && int(id) < len(function.Blocks) && function.Blocks[id].Entity != "" {
		return function.Blocks[id].Entity
	}
	return fmt.Sprintf("block:%d", id)
}

func operationEntity(block BlockID, index int, operation Operation) string {
	return fallbackEntity(operation.Entity, fmt.Sprintf("block:%d.operation:%d", block, index))
}

func terminatorEntity(block BlockID, terminator Terminator) string {
	return fallbackEntity(terminator.Entity, fmt.Sprintf("block:%d.terminator", block))
}

func resourceEntity(resource ResourceLocal, fallback string) string {
	return fallbackEntity(resource.Entity, fallback)
}

func fallbackEntity(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
