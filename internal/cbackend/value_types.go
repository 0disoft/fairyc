package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func canonicalCValueType(value hir.Type) hir.Type {
	if value == hir.TypeBool {
		return hir.TypeI32
	}
	return value
}

func fieldAggregateType(value hir.Type) hir.Type {
	if element, _, borrow := hir.BorrowElement(value); borrow {
		return element
	}
	return value
}

func isOptionI32(value hir.Type) bool {
	element, ok := hir.OptionElement(value)
	return ok && element == hir.TypeI32
}

func isResultI32I32(value hir.Type) bool {
	success, failure, ok := hir.ResultElements(value)
	return ok && success == hir.TypeI32 && failure == hir.TypeI32
}

func (e *emissionAnalysis) supportedCValueType(value hir.Type) bool {
	value = canonicalCValueType(value)
	if value == hir.SharedBorrow(hir.TypeByteSlice) || value == hir.MutableBorrow(hir.TypeByteSlice) || hir.NonByteIndexedView(value) ||
		value == hir.SharedBorrow(hir.TypeStr) || value == hir.SharedBorrow(hir.TypeCStr) {
		return true
	}
	if _, borrow := e.safeBorrowCType(value); borrow {
		return true
	}
	if value == hir.TypeCCallbackI32UserData || value == hir.TypeU8 || value == hir.TypeI32 || value == hir.TypeI64 || value == hir.TypeU32 || value == hir.TypeUSize || hir.IsCScalar(value) || hir.IsFloatScalar(value) || hir.IsIntegerTransportScalar(value) || isOptionI32(value) || isResultI32I32(value) {
		return true
	}
	if _, array := e.arrayByType[value]; array {
		return true
	}
	if element, _, pointer := hir.RawPointerElement(value); pointer {
		_, err := e.foreignPointerElementCType(element)
		return err == nil
	}
	_, resource := e.resourceByType[value]
	if resource {
		return true
	}
	if _, structure := e.structByType[value]; structure {
		return true
	}
	if _, enumeration := e.enumByType[value]; enumeration {
		return true
	}
	_, tagged := e.taggedByType[value]
	return tagged
}

func (e *emissionAnalysis) cValueType(value hir.Type) (string, error) {
	value = canonicalCValueType(value)
	if value == hir.TypeCCallbackI32UserData {
		return "fy_c_callback_i32_userdata", nil
	}
	if cType, ok := integerTransportCType(value); ok {
		return cType, nil
	}
	if transport, found := cScalarTransportForType(value); found {
		if _, available := e.cScalars[value]; !available {
			return "", fmt.Errorf("C scalar %q is unavailable for target %q", transport.targetName, e.targetID)
		}
		return transport.cType, nil
	}
	if cType, borrow := e.safeBorrowCType(value); borrow {
		return cType, nil
	}
	if descriptor, _, view := indexedViewCType(value); view {
		return descriptor, nil
	}
	switch {
	case value == hir.SharedBorrow(hir.TypeStr):
		return "fy_str", nil
	case value == hir.SharedBorrow(hir.TypeCStr):
		return "const uint8_t *", nil
	case value == hir.TypeU8:
		return "uint8_t", nil
	case value == hir.TypeF32:
		return "float", nil
	case value == hir.TypeF64:
		return "double", nil
	case value == hir.TypeI64:
		return "int64_t", nil
	case value == hir.TypeI32:
		return "int32_t", nil
	case value == hir.TypeU32:
		return "uint32_t", nil
	case value == hir.TypeUSize:
		return "uintptr_t", nil
	case isRawPointer(value):
		return e.foreignCType(value, false)
	case isOptionI32(value):
		return "fy_option_i32", nil
	case isResultI32I32(value):
		return "fy_result_i32_i32", nil
	default:
		if index, found := e.arrayByType[value]; found {
			return e.arrayCTypeName(index), nil
		}
		if index, found := e.structByType[value]; found {
			return e.structCTypeName(index), nil
		}
		if index, found := e.enumByType[value]; found {
			return fmt.Sprintf("fy_enum_%d", index), nil
		}
		if index, found := e.resourceByType[value]; found {
			return fmt.Sprintf("fy_resource_%d", index), nil
		}
		if index, found := e.taggedByType[value]; found {
			return fmt.Sprintf("fy_tagged_%d", index), nil
		}
		return "", fmt.Errorf("unsupported C value type %q", value)
	}
}

func (e *emissionAnalysis) safeBorrowCType(value hir.Type) (string, bool) {
	element, mutable, borrow := hir.BorrowElement(value)
	if !borrow {
		return "", false
	}
	pointee := ""
	canonical := canonicalCValueType(element)
	if index, found := e.resourceByType[canonical]; found {
		pointee = fmt.Sprintf("fy_resource_%d", index)
	} else if index, found := e.arrayByType[canonical]; found {
		pointee = e.arrayCTypeName(index)
	} else if index, found := e.structByType[canonical]; found {
		pointee = e.structCTypeName(index)
	} else {
		switch canonical {
		case hir.TypeU8:
			pointee = "uint8_t"
		case hir.TypeI64:
			pointee = "int64_t"
		case hir.TypeI32:
			pointee = "int32_t"
		case hir.TypeU32:
			pointee = "uint32_t"
		case hir.TypeUSize:
			pointee = "uintptr_t"
		}
	}
	if pointee == "" {
		return "", false
	}
	if mutable {
		return pointee + " *", true
	}
	return "const " + pointee + " *", true
}

func (e *emissionAnalysis) structCTypeName(index int) string {
	definition := e.structTypes[index].definition
	if definition.ReprC {
		return "struct " + definition.Name
	}
	return fmt.Sprintf("fy_struct_%d", index)
}
