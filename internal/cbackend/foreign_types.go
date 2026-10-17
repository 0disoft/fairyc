package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func (e *emissionAnalysis) enumPayloadCType(value hir.Type) (string, error) {
	canonical := canonicalCValueType(value)
	if canonical == hir.TypeU8 || canonical == hir.TypeI32 || canonical == hir.TypeI64 || canonical == hir.TypeU32 || canonical == hir.TypeUSize ||
		isRawPointer(canonical) || isOptionI32(canonical) || isResultI32I32(canonical) {
		return e.cValueType(canonical)
	}
	if _, direct := copyScalarTaggedAggregateLayout(canonical); direct {
		return e.cValueType(canonical)
	}
	if index, found := e.taggedByType[canonical]; found && e.taggedParticipatesInAggregateGraph(index) {
		return e.cValueType(canonical)
	}
	if _, structure := e.structByType[canonical]; structure {
		return e.cValueType(canonical)
	}
	if _, enumeration := e.enumByType[canonical]; enumeration {
		return e.cValueType(canonical)
	}
	return "", fmt.Errorf("unsupported enum payload type %q", value)
}

func (e *emissionAnalysis) foreignCType(value hir.Type, result bool) (string, error) {
	if result && value == hir.TypeVoid {
		return "void", nil
	}
	if transport, found := cScalarTransportForType(value); found {
		if _, available := e.cScalars[value]; !available {
			return "", fmt.Errorf("C scalar %q is unavailable for target %q", transport.targetName, e.targetID)
		}
		return transport.cType, nil
	}
	switch value {
	case hir.TypeU8:
		return "uint8_t", nil
	case hir.TypeI64:
		return "int64_t", nil
	case hir.TypeI32:
		return "int32_t", nil
	case hir.TypeU32:
		return "uint32_t", nil
	case hir.TypeUSize:
		return "uintptr_t", nil
	}
	if element, mutable, pointer := hir.RawPointerElement(value); pointer {
		if nestedElement, nestedMutable, nested := hir.RawPointerElement(element); nested {
			if mutable && nestedMutable && nestedElement == hir.TypeVoid {
				return "void * *", nil
			}
			return "", fmt.Errorf("foreign raw-pointer nesting type %q", value)
		}
		pointee, err := e.foreignPointerElementCType(element)
		if err != nil {
			return "", err
		}
		if mutable {
			return pointee + " *", nil
		}
		return "const " + pointee + " *", nil
	}
	if index, structure := e.structByType[value]; structure {
		if !result {
			return "", fmt.Errorf("foreign parameter type %q", value)
		}
		if !e.structTypes[index].definition.ReprC {
			return "", fmt.Errorf("foreign result struct %q is not repr(C)", value)
		}
		return e.structCTypeName(index), nil
	}
	role := "parameter"
	if result {
		role = "result"
	}
	return "", fmt.Errorf("foreign %s type %q", role, value)
}

func (e *emissionAnalysis) foreignPointerElementCType(value hir.Type) (string, error) {
	switch value {
	case hir.TypeVoid:
		return "void", nil
	case hir.TypeU8:
		return "uint8_t", nil
	case hir.TypeI64:
		return "int64_t", nil
	case hir.TypeI32:
		return "int32_t", nil
	case hir.TypeU32:
		return "uint32_t", nil
	case hir.TypeUSize:
		return "uintptr_t", nil
	default:
		if index, found := e.structByType[value]; found {
			definition := e.structTypes[index].definition
			if !definition.ReprC {
				return "", fmt.Errorf("foreign raw-pointer element struct %q is not repr(C)", value)
			}
			return "struct " + definition.Name, nil
		}
		return "", fmt.Errorf("foreign raw-pointer element type %q", value)
	}
}
