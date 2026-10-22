package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
	"strconv"
	"strings"
)

func (e *emitter) emitOperation(output *strings.Builder, functionIndex int, operation mir.Operation) error {
	if err := e.contextError(); err != nil {
		return err
	}
	function := e.analysis.functions[functionIndex].Name
	switch operation.Kind {
	case mir.OperationCCallback:
		index, found := e.analysis.bySymbol[operation.Symbol]
		if !found {
			return fmt.Errorf("missing callback target %d", operation.Symbol)
		}
		fmt.Fprintf(output, "    fy_l_%d = &fy_cb_%d;\n", operation.DestinationLocal, index)
	case mir.OperationAssign, mir.OperationInitialize:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("assign in %s: %w", function, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("assign in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, value)
	case mir.OperationRawAddress:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandLocal {
			return fmt.Errorf("raw address in %s has invalid local operand", function)
		}
		fmt.Fprintf(output, "    fy_l_%d = &fy_l_%d;\n", operation.DestinationLocal, operand.Local)
	case mir.OperationRawBorrowPointer:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandLocal {
			return fmt.Errorf("raw borrow pointer in %s has invalid local operand", function)
		}
		fmt.Fprintf(output, "    fy_l_%d = fy_l_%d;\n", operation.DestinationLocal, operand.Local)
	case mir.OperationStrFromRawParts, mir.OperationCStrFromRawParts, mir.OperationByteSliceFromRawParts, mir.OperationMutByteSliceFromRawParts:
		if len(operation.Operands) != 3 {
			return fmt.Errorf("borrowed-view raw-parts construction in %s has invalid operands", function)
		}
		owner, err := cOperand(operation.Operands[0])
		if err != nil {
			return fmt.Errorf("borrowed-view raw-parts owner in %s: %w", function, err)
		}
		data, err := cOperand(operation.Operands[1])
		if err != nil {
			return fmt.Errorf("borrowed-view raw-parts data in %s: %w", function, err)
		}
		length, err := cOperand(operation.Operands[2])
		if err != nil {
			return fmt.Errorf("borrowed-view raw-parts length in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    (void)%s;\n", owner)
		if operation.Kind == mir.OperationStrFromRawParts {
			fmt.Fprintf(output, "    fy_l_%d = (fy_str){ .data = ((const uint8_t *)(%s) != (const uint8_t *)0 ? (const uint8_t *)(%s) : (const uint8_t *)\"\"), .length = %s };\n", operation.DestinationLocal, data, data, length)
		} else if operation.Kind == mir.OperationCStrFromRawParts {
			fmt.Fprintf(output, "    (void)%s;\n", length)
			fmt.Fprintf(output, "    fy_l_%d = (const uint8_t *)(%s);\n", operation.DestinationLocal, data)
		} else if operation.Kind == mir.OperationByteSliceFromRawParts {
			fmt.Fprintf(output, "    fy_l_%d = (fy_byte_slice){ .data = (const uint8_t *)(%s), .length = %s };\n", operation.DestinationLocal, data, length)
		} else {
			fmt.Fprintf(output, "    fy_l_%d = (fy_mut_byte_slice){ .data = (uint8_t *)(%s), .length = %s };\n", operation.DestinationLocal, data, length)
		}
	case mir.OperationStrDataAddress:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandString && operand.Kind != mir.OperandLocal {
			return fmt.Errorf("string data address in %s has invalid storage operand", function)
		}
		if operand.Kind == mir.OperandLocal {
			fmt.Fprintf(output, "    fy_l_%d = fy_l_%d.data;\n", operation.DestinationLocal, operand.Local)
		} else {
			fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, stringStorageSymbol(operand.StaticData))
		}
	case mir.OperationByteSliceDataAddress:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandByteString && operand.Kind != mir.OperandLocal {
			return fmt.Errorf("byte-slice data address in %s has invalid storage operand", function)
		}
		if operand.Kind == mir.OperandLocal {
			fmt.Fprintf(output, "    fy_l_%d = fy_l_%d.data;\n", operation.DestinationLocal, operand.Local)
		} else {
			fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, byteStringStorageSymbol(operand.StaticData))
		}
	case mir.OperationCStrDataAddress:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandCString && operand.Kind != mir.OperandLocal {
			return fmt.Errorf("C string data address in %s has invalid storage operand", function)
		}
		if operand.Kind == mir.OperandLocal {
			fmt.Fprintf(output, "    fy_l_%d = fy_l_%d;\n", operation.DestinationLocal, operand.Local)
		} else {
			value, err := cOperand(operand)
			if err != nil {
				return fmt.Errorf("C string data address in %s: %w", function, err)
			}
			fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, value)
		}
	case mir.OperationEnumConstruct:
		enumIndex, found := e.analysis.enumByType[operation.Type]
		if !found {
			return CapabilityError{Function: function, Feature: "enum construction type " + strconv.Quote(operation.Type.String())}
		}
		enumeration := &e.analysis.enumTypes[enumIndex]
		variantIndex, found := enumeration.variantIndex[operation.Variant]
		if !found {
			return fmt.Errorf("enum construction in %s names missing variant %q", function, operation.Variant)
		}
		if enumeration.definition.CBase != "" {
			fmt.Fprintf(output, "    fy_l_%d = (fy_enum_%d)UINT64_C(%d);\n", operation.DestinationLocal, enumIndex, variantIndex)
			return nil
		}
		fmt.Fprintf(output, "    fy_l_%d = (fy_enum_%d){ .tag = UINT32_C(%d)", operation.DestinationLocal, enumIndex, variantIndex)
		if len(operation.Operands) != 0 {
			fmt.Fprintf(output, ", .payload.v_%d = { ", variantIndex)
			for operandIndex, operand := range operation.Operands {
				if operandIndex != 0 {
					output.WriteString(", ")
				}
				fieldIndex, declared := enumeration.fieldIndex[variantIndex][operation.Fields[operandIndex]]
				if !declared {
					return fmt.Errorf("enum construction in %s names missing field %q", function, operation.Fields[operandIndex])
				}
				value, err := cOperand(operand)
				if err != nil {
					return fmt.Errorf("enum construction field %d in %s: %w", operandIndex, function, err)
				}
				fmt.Fprintf(output, ".f_%d = %s", fieldIndex, value)
			}
			output.WriteString(" }")
		}
		output.WriteString(" };\n")
	case mir.OperationEnumPayload:
		enumIndex, found := e.analysis.enumByType[operation.OwnerType]
		if !found {
			return CapabilityError{Function: function, Feature: "enum payload type " + strconv.Quote(operation.OwnerType.String())}
		}
		enumeration := &e.analysis.enumTypes[enumIndex]
		variantIndex, found := enumeration.variantIndex[operation.Variant]
		if !found {
			return fmt.Errorf("enum payload in %s names missing variant %q", function, operation.Variant)
		}
		fieldIndex, found := enumeration.fieldIndex[variantIndex][operation.Field]
		if !found {
			return fmt.Errorf("enum payload in %s names missing field %q", function, operation.Field)
		}
		container, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("enum payload in %s: %w", function, err)
		}
		value, err := cOperand(container)
		if err != nil {
			return fmt.Errorf("enum payload in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s).payload.v_%d.f_%d;\n", operation.DestinationLocal, value, variantIndex, fieldIndex)
	case mir.OperationStructConstruct:
		index, found := e.analysis.structByType[operation.Type]
		if !found {
			return CapabilityError{Function: function, Feature: "struct construction type " + strconv.Quote(operation.Type.String())}
		}
		for _, field := range e.analysis.structTypes[index].definition.Fields {
			if e.analysis.rawArrayMember(operation.Type, field.Type) {
				return e.writeReprCArrayConstruction(output, operation, index)
			}
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s){ ", operation.DestinationLocal, e.analysis.structCTypeName(index))
		if len(operation.Operands) == 0 {
			output.WriteString("0")
		}
		for fieldIndex, operand := range operation.Operands {
			if fieldIndex != 0 {
				output.WriteString(", ")
			}
			value, err := cOperand(operand)
			if err != nil {
				return fmt.Errorf("struct construction field %d in %s: %w", fieldIndex, function, err)
			}
			fmt.Fprintf(output, ".%s = %s", e.analysis.structCFieldName(index, fieldIndex), value)
		}
		output.WriteString(" };\n")
	case mir.OperationResourceConstruct:
		index, found := e.analysis.resourceByType[operation.Type]
		if !found {
			return CapabilityError{Function: function, Feature: "resource construction type " + strconv.Quote(operation.Type.String())}
		}
		fmt.Fprintf(output, "    fy_l_%d = (fy_resource_%d){ ", operation.DestinationLocal, index)
		for fieldIndex, operand := range operation.Operands {
			if fieldIndex != 0 {
				output.WriteString(", ")
			}
			value, err := cOperand(operand)
			if err != nil {
				return fmt.Errorf("resource construction field %d in %s: %w", fieldIndex, function, err)
			}
			fmt.Fprintf(output, ".f_%d = %s", fieldIndex, value)
		}
		output.WriteString(" };\n")
	case mir.OperationFieldStore:
		return e.writeFieldStore(output, operation)
	case mir.OperationDerefStore:
		return e.writeDerefStore(output, operation)
	case mir.OperationFieldBorrow:
		return e.writeFieldBorrow(output, operation)
	case mir.OperationFieldRead:
		aggregateType := fieldAggregateType(operation.OwnerType)
		_, _, borrowed := hir.BorrowElement(operation.OwnerType)
		if index, found := e.analysis.structByType[aggregateType]; found {
			fieldIndex, fieldFound := e.analysis.structTypes[index].fieldIndex[operation.Field]
			if !fieldFound {
				return fmt.Errorf("struct field read in %s names missing field %q", function, operation.Field)
			}
			receiver, err := oneOperand(operation.Operands)
			if err != nil {
				return fmt.Errorf("struct field read in %s: %w", function, err)
			}
			value, err := cOperand(receiver)
			if err != nil {
				return fmt.Errorf("struct field read in %s: %w", function, err)
			}
			if e.analysis.rawArrayMember(operation.OwnerType, operation.Type) {
				access := "."
				if borrowed {
					access = "->"
				}
				place := "(" + value + ")" + access + e.analysis.structCFieldName(index, fieldIndex)
				writeRawArrayCopy(output, fmt.Sprintf("fy_l_%d.data", operation.DestinationLocal), place, operation.Type)
			} else if borrowed {
				fmt.Fprintf(output, "    fy_l_%d = (%s)->%s;\n", operation.DestinationLocal, value, e.analysis.structCFieldName(index, fieldIndex))
			} else {
				fmt.Fprintf(output, "    fy_l_%d = (%s).%s;\n", operation.DestinationLocal, value, e.analysis.structCFieldName(index, fieldIndex))
			}
			break
		}
		index, found := e.analysis.resourceByType[aggregateType]
		if !found {
			return CapabilityError{Function: function, Feature: "field-read owner type " + strconv.Quote(operation.OwnerType.String())}
		}
		fieldIndex, found := e.analysis.resourceTypes[index].fieldIndex[operation.Field]
		if !found {
			return fmt.Errorf("resource field read in %s names missing field %q", function, operation.Field)
		}
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("resource field read in %s: %w", function, err)
		}
		value, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("resource field read in %s: %w", function, err)
		}
		if borrowed {
			fmt.Fprintf(output, "    fy_l_%d = (%s)->f_%d;\n", operation.DestinationLocal, value, fieldIndex)
		} else {
			fmt.Fprintf(output, "    fy_l_%d = (%s).f_%d;\n", operation.DestinationLocal, value, fieldIndex)
		}
	case mir.OperationViewLength:
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("borrowed-view length in %s: %w", function, err)
		}
		value, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("borrowed-view length in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s).length;\n", operation.DestinationLocal, value)
	case mir.OperationViewEmpty:
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("borrowed-view emptiness in %s: %w", function, err)
		}
		value, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("borrowed-view emptiness in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = ((%s).length == ((uintptr_t)UINT64_C(0))) ? INT32_C(1) : INT32_C(0);\n", operation.DestinationLocal, value)
	case mir.OperationArrayConstruct:
		arrayIndex, found := e.analysis.arrayByType[operation.Type]
		if !found {
			return CapabilityError{Function: function, Feature: "fixed-array construction type " + strconv.Quote(operation.Type.String())}
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s){ .data = { ", operation.DestinationLocal, e.analysis.arrayCTypeName(arrayIndex))
		for index, operand := range operation.Operands {
			if index != 0 {
				output.WriteString(", ")
			}
			value, err := cOperand(operand)
			if err != nil {
				return fmt.Errorf("fixed-array construction element %d in %s: %w", index, function, err)
			}
			output.WriteString(value)
		}
		output.WriteString(" } };\n")
	case mir.OperationArrayBorrowView:
		receiver, err := oneOperand(operation.Operands)
		if err != nil || receiver.Kind != mir.OperandLocal {
			return fmt.Errorf("fixed-array byte-view borrow in %s has no local receiver", function)
		}
		ownerType := operation.OwnerType
		access := "."
		if element, _, borrowed := hir.BorrowElement(ownerType); borrowed {
			ownerType, access = element, "->"
		}
		_, length, array := hir.FixedArrayParts(ownerType)
		if !array {
			return fmt.Errorf("fixed-array byte-view borrow in %s has type %q", function, operation.OwnerType)
		}
		viewType, err := e.analysis.cValueType(operation.Type)
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s){ .data = fy_l_%d%sdata, .length = ((uintptr_t)UINT64_C(%d)) };\n", operation.DestinationLocal, viewType, receiver.Local, access, length)
	case mir.OperationStorageLayout:
		value, err := e.analysis.storageQueryValue(operation)
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "    fy_l_%d = ((uintptr_t)UINT64_C(%d));\n", operation.DestinationLocal, value)
	case mir.OperationStorageTake:
		return e.writeStorageTake(output, operation)
	case mir.OperationStoragePut:
		return e.writeStoragePut(output, operation)
	case mir.OperationStorageBorrow, mir.OperationStorageBorrowMut:
		return e.writeStorageBorrow(output, operation)
	case mir.OperationStorageElementAddress:
		return e.writeStorageElementAddress(output, operation)
	case mir.OperationArrayLength:
		if _, err := oneOperand(operation.Operands); err != nil {
			return fmt.Errorf("fixed-array length in %s: %w", function, err)
		}
		_, length, array := hir.FixedArrayParts(operation.OwnerType)
		if !array {
			return fmt.Errorf("fixed-array length in %s has type %q", function, operation.OwnerType)
		}
		fmt.Fprintf(output, "    fy_l_%d = ((uintptr_t)UINT64_C(%d));\n", operation.DestinationLocal, length)
	case mir.OperationArrayIndex:
		receiver, index, err := twoOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("fixed-array index in %s: %w", function, err)
		}
		receiverValue, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("fixed-array index receiver in %s: %w", function, err)
		}
		indexValue, err := cOperand(index)
		if err != nil {
			return fmt.Errorf("fixed-array index offset in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s).data[%s];\n", operation.DestinationLocal, receiverValue, indexValue)
	case mir.OperationArrayIndexStore:
		receiver, index, value, err := threeOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("fixed-array index store in %s: %w", function, err)
		}
		receiverValue, receiverErr := cOperand(receiver)
		indexValue, indexErr := cOperand(index)
		storedValue, valueErr := cOperand(value)
		if receiverErr != nil || indexErr != nil || valueErr != nil {
			return fmt.Errorf("fixed-array index store in %s has an invalid operand", function)
		}
		fmt.Fprintf(output, "    (%s).data[%s] = %s;\n", receiverValue, indexValue, storedValue)
	case mir.OperationViewIndex:
		receiver, index, err := twoOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("byte-view index in %s: %w", function, err)
		}
		receiverValue, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("byte-view index receiver in %s: %w", function, err)
		}
		indexValue, err := cOperand(index)
		if err != nil {
			return fmt.Errorf("byte-view index offset in %s: %w", function, err)
		}
		// MSVC /O2 can retain stale reads after exclusive i64 stores (ADR 0433).
		// Keep shared-only programs and other targets on ordinary loads.
		if element, structView := hir.StructViewElement(operation.OwnerType); structView {
			typ, err := e.analysis.cValueType(element)
			if err != nil {
				return err
			}
			qualifier := "const "
			if e.analysis.targetID == target.X8664WindowsMSVC && e.analysis.needsMutStructSlice {
				qualifier += "volatile "
			}
			fmt.Fprintf(output, "    fy_l_%d = ((%s%s *)(%s).data)[%s];\n", operation.DestinationLocal, qualifier, typ, receiverValue, indexValue)
		} else if e.analysis.targetID == target.X8664WindowsMSVC && e.analysis.needsMutI64Slice && operation.Type == hir.TypeI64 {
			fmt.Fprintf(output, "    fy_l_%d = ((const volatile int64_t *)(%s).data)[%s];\n", operation.DestinationLocal, receiverValue, indexValue)
		} else {
			fmt.Fprintf(output, "    fy_l_%d = (%s).data[%s];\n", operation.DestinationLocal, receiverValue, indexValue)
		}
	case mir.OperationViewElementBorrow:
		receiver, err := cOperand(operation.Operands[0])
		if err != nil {
			return err
		}
		index, err := cOperand(operation.Operands[1])
		if err != nil {
			return err
		}
		element, mutable, _ := hir.BorrowElement(operation.Type)
		typ, err := e.analysis.cValueType(element)
		if err != nil {
			return err
		}
		qualifier := "const "
		if mutable {
			qualifier = ""
		}
		fmt.Fprintf(output, "    fy_l_%d = &((%s%s *)(%s).data)[%s];\n", operation.DestinationLocal, qualifier, typ, receiver, index)

	case mir.OperationViewIndexStore:
		receiver, index, value, err := threeOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("mutable byte-view index store in %s: %w", function, err)
		}
		receiverValue, receiverErr := cOperand(receiver)
		indexValue, indexErr := cOperand(index)
		storedValue, valueErr := cOperand(value)
		if receiverErr != nil || indexErr != nil || valueErr != nil {
			return fmt.Errorf("mutable byte-view index store in %s has an invalid operand", function)
		}
		if element, nominal := hir.StructViewElement(operation.OwnerType); nominal && e.analysis.targetID != target.X8664WindowsMSVC {
			cType, err := e.analysis.cValueType(element)
			if err != nil {
				return err
			}
			fmt.Fprintf(output, "    ((%s *)(%s).data)[%s] = %s;\n", cType, receiverValue, indexValue, storedValue)
		} else if e.analysis.targetID == target.X8664WindowsMSVC {
			// MSVC 14.51 /O2 loses inlined stores through a
			// copied subview descriptor. Keep this store observable without
			// changing descriptor layout or disabling function optimization.
			cType, err := e.analysis.cValueType(operation.Type)
			if err != nil {
				return err
			}
			fmt.Fprintf(output, "    ((volatile %s *)(%s).data)[%s] = %s;\n", cType, receiverValue, indexValue, storedValue)
		} else {
			fmt.Fprintf(output, "    (%s).data[%s] = %s;\n", receiverValue, indexValue, storedValue)
		}
	case mir.OperationViewSlice:
		receiver, start, end, err := threeOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("immutable byte-view slice in %s: %w", function, err)
		}
		receiverValue, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("immutable byte-view slice receiver in %s: %w", function, err)
		}
		startValue, err := cOperand(start)
		if err != nil {
			return fmt.Errorf("immutable byte-view slice start in %s: %w", function, err)
		}
		endValue, err := cOperand(end)
		if err != nil {
			return fmt.Errorf("immutable byte-view slice end in %s: %w", function, err)
		}
		viewType, err := e.analysis.cValueType(operation.Type)
		if err != nil {
			return err
		}
		if element, structView := hir.StructViewElement(operation.OwnerType); structView {
			typ, err := e.analysis.cValueType(element)
			if err != nil {
				return err
			}
			fmt.Fprintf(output, "    fy_l_%d = (%s){ .data = (%s).data, .length = (%s) - (%s) };\n", operation.DestinationLocal, viewType, receiverValue, endValue, startValue)
			if start.Kind != mir.OperandUSize || start.U64 != 0 {
				qualifier := "const "
				if hir.MutableIndexedView(operation.OwnerType) {
					qualifier = ""
				}
				fmt.Fprintf(output, "    if ((%s) != ((uintptr_t)UINT64_C(0))) { fy_l_%d.data = (%s%s *)(%s).data + (%s); }\n", startValue, operation.DestinationLocal, qualifier, typ, receiverValue, startValue)
			}
			return nil
		}
		switch {
		case start.Kind == mir.OperandUSize && start.U64 == 0:
			fmt.Fprintf(output, "    fy_l_%d = (%s){ .data = (%s).data, .length = (%s) };\n", operation.DestinationLocal, viewType, receiverValue, endValue)
		case start.Kind == mir.OperandUSize:
			fmt.Fprintf(output, "    fy_l_%d = (%s){ .data = (%s).data + (%s), .length = (%s) - (%s) };\n", operation.DestinationLocal, viewType, receiverValue, startValue, endValue, startValue)
		default:
			fmt.Fprintf(output, "    fy_l_%d = (%s){ .data = (%s).data, .length = (%s) - (%s) };\n", operation.DestinationLocal, viewType, receiverValue, endValue, startValue)
			fmt.Fprintf(output, "    if ((%s) != ((uintptr_t)UINT64_C(0))) { fy_l_%d.data += (%s); }\n", startValue, operation.DestinationLocal, startValue)
		}
	case mir.OperationSharedReborrow:
		value, err := cOperand(operation.Operands[0])
		if err != nil {
			return err
		}
		if hir.NonByteIndexedView(operation.Type) {
			descriptor, _, _ := indexedViewCType(operation.Type)
			fmt.Fprintf(output, "    fy_l_%d = (%s){ .data = (%s).data, .length = (%s).length };\n", operation.DestinationLocal, descriptor, value, value)
		} else if operation.Type == hir.SharedBorrow(hir.TypeByteSlice) {
			fmt.Fprintf(output, "    fy_l_%d = (fy_byte_slice){ .data = (%s).data, .length = (%s).length };\n", operation.DestinationLocal, value, value)
		} else {
			fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, value)
		}
	case mir.OperationScalarCast:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("value-preserving integer cast in %s: %w", function, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("value-preserving integer cast in %s: %w", function, err)
		}
		targetType, err := e.analysis.cValueType(operation.Type)
		if err != nil {
			return fmt.Errorf("value-preserving integer cast result in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s)(%s);\n", operation.DestinationLocal, targetType, value)
	case mir.OperationMove:
		fmt.Fprintf(output, "    (void)fy_l_%d;\n", operation.Symbol)
	case mir.OperationDrop:
		fmt.Fprintf(output, "    (void)fy_l_%d;\n", operation.Symbol)
		if e.analysis.instrumentation == InstrumentationDropTraceV0 {
			if err := e.emitDropTrace(output, functionIndex, operation); err != nil {
				return fmt.Errorf("drop trace in %s: %w", function, err)
			}
		}
		if err := e.emitDropGlue(output, functionIndex, operation); err != nil {
			return fmt.Errorf("drop glue in %s: %w", function, err)
		}
	case mir.OperationBurn:
		fmt.Fprintf(output, "    (void)fy_l_%d;\n", operation.Symbol)
	case mir.OperationRegionEnter:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandLocal {
			return fmt.Errorf("Region entry in %s has invalid Arena owner", function)
		}
		fmt.Fprintf(output, "    if (fy_std_region_begin_v0((void *)&fy_l_%d, (void *)&fy_region_%d) != UINT32_C(0)) %s;\n", operand.Local, operation.Symbol, e.trapCall("FY_TRAP_UNREACHABLE", operation.Span))
	case mir.OperationRegionPlace:
		if operation.Operator == "alloc_zeroed_bytes" || operation.Operator == "copy_bytes" {
			return e.emitRegionBytes(output, functionIndex, operation)
		}
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Region placement in %s: %w", function, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Region placement value in %s: %w", function, err)
		}
		ownerType, err := e.analysis.cValueType(operation.OwnerType)
		if err != nil {
			return fmt.Errorf("Region placement owner type in %s: %w", function, err)
		}
		taggedIndex, found := e.analysis.taggedByType[operation.Type]
		if !found {
			return fmt.Errorf("Region placement in %s has no tagged C type", function)
		}
		_, failureType, _, _ := e.analysis.resultCSpec(operation.Type)
		errorIndex, found := e.analysis.structByType[failureType]
		if !found {
			return fmt.Errorf("Region placement in %s has no allocation error C type", function)
		}
		errorField := e.analysis.structCFieldName(errorIndex, 0)
		// A MIR block label may precede this operation. C11 labels require a
		// statement, so keep placement temporaries inside a compound statement.
		output.WriteString("    {\n")
		fmt.Fprintf(output, "    void *fy_region_storage_%d = NULL;\n", operation.DestinationLocal)
		fmt.Fprintf(output, "    uint32_t fy_region_status_%d = fy_std_region_allocate_zeroed_v0((void *)&fy_region_%d, sizeof(%s), _Alignof(%s), &fy_region_storage_%d);\n", operation.DestinationLocal, operation.Symbol, ownerType, ownerType, operation.DestinationLocal)
		fmt.Fprintf(output, "    if (fy_region_status_%d == UINT32_C(0)) {\n", operation.DestinationLocal)
		fmt.Fprintf(output, "        *((%s *)fy_region_storage_%d) = %s;\n", ownerType, operation.DestinationLocal, value)
		fmt.Fprintf(output, "        fy_l_%d = (fy_tagged_%d){ .tag = FY_RESULT_OK, .p_0 = (const %s *)fy_region_storage_%d };\n", operation.DestinationLocal, taggedIndex, ownerType, operation.DestinationLocal)
		output.WriteString("    } else {\n")
		fmt.Fprintf(output, "        fy_l_%d = (fy_tagged_%d){ .tag = FY_RESULT_ERR, .p_1 = (%s){ .%s = fy_region_status_%d } };\n", operation.DestinationLocal, taggedIndex, e.analysis.structCTypeName(errorIndex), errorField, operation.DestinationLocal)
		output.WriteString("    }\n")
		output.WriteString("    }\n")
	case mir.OperationRegionReset:
		fmt.Fprintf(output, "    if (fy_std_region_reset_v0((void *)&fy_region_%d) != UINT32_C(0)) %s;\n", operation.Symbol, e.trapCall("FY_TRAP_UNREACHABLE", operation.Span))
	case mir.OperationUnary:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("unary in %s: %w", function, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("unary in %s: %w", function, err)
		}
		switch operation.Operator {
		case "+":
			fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, value)
		case "-":
			if operation.OwnerType == hir.TypeI64 {
				fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i64(UINT64_C(0) - (uint64_t)(%s));\n", operation.DestinationLocal, value)
				break
			}
			fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32(UINT32_C(0) - (uint32_t)(%s));\n", operation.DestinationLocal, value)
		case "!":
			fmt.Fprintf(output, "    fy_l_%d = (%s == INT32_C(0)) ? INT32_C(1) : INT32_C(0);\n", operation.DestinationLocal, value)
		case "~":
			if operation.OwnerType == hir.TypeI64 {
				fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i64(~(uint64_t)(%s));\n", operation.DestinationLocal, value)
				break
			}
			if operation.OwnerType == hir.TypeI32 {
				fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32(~(uint32_t)(%s));\n", operation.DestinationLocal, value)
			} else {
				valueType, err := e.analysis.cValueType(operation.OwnerType)
				if err != nil {
					return fmt.Errorf("bitwise not in %s: %w", function, err)
				}
				fmt.Fprintf(output, "    fy_l_%d = (%s)(~(%s)(%s));\n", operation.DestinationLocal, valueType, valueType, value)
			}
		case "&", "&mut":
			if operand.Kind != mir.OperandLocal {
				return fmt.Errorf("safe borrow in %s has invalid local operand", function)
			}
			fmt.Fprintf(output, "    fy_l_%d = &fy_l_%d;\n", operation.DestinationLocal, operand.Local)
		case "*":
			fmt.Fprintf(output, "    fy_l_%d = *(%s);\n", operation.DestinationLocal, value)
		}
	case mir.OperationBinary:
		left, right, err := twoOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("binary in %s: %w", function, err)
		}
		leftValue, err := cOperand(left)
		if err != nil {
			return fmt.Errorf("binary in %s: %w", function, err)
		}
		rightValue, err := cOperand(right)
		if err != nil {
			return fmt.Errorf("binary in %s: %w", function, err)
		}
		if operation.Operator == "<<" || operation.Operator == ">>" {
			rightValue = reachableShiftCount(operation.OwnerType, right, rightValue)
		}
		if operation.OwnerType == hir.TypeI64 {
			emitI64Binary(output, operation, leftValue, rightValue)
			break
		}
		switch operation.Operator {
		case "+", "-", "*":
			if operation.OwnerType == hir.TypeI32 {
				fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32((uint32_t)(%s) %s (uint32_t)(%s));\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
			} else {
				fmt.Fprintf(output, "    fy_l_%d = (%s) %s (%s);\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
			}
		case "/", "%":
			if directUnsignedDivisor(operation) {
				fmt.Fprintf(output, "    fy_l_%d = (%s) %s (%s);\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
				break
			}
			divisor, err := checkedDivisorName(operation.OwnerType)
			if err != nil {
				return fmt.Errorf("checked division in %s: %w", function, err)
			}
			output.WriteString("    " + divisor + " = ")
			output.WriteString(rightValue)
			output.WriteString(";\n")
			fmt.Fprintf(output, "    fy_l_%d = (%s) %s %s;\n", operation.DestinationLocal, leftValue, operation.Operator, divisor)
		case ">>":
			if operation.OwnerType == hir.TypeI32 {
				fmt.Fprintf(output, "    fy_l_%d = fy_sar_i32(%s, %s);\n", operation.DestinationLocal, leftValue, rightValue)
			} else {
				fmt.Fprintf(output, "    fy_l_%d = (%s) >> (%s);\n", operation.DestinationLocal, leftValue, rightValue)
			}
		case "<<":
			if operation.OwnerType == hir.TypeI32 {
				fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32((uint32_t)(%s) << (uint32_t)(%s));\n", operation.DestinationLocal, leftValue, rightValue)
			} else {
				fmt.Fprintf(output, "    fy_l_%d = (%s) << (%s);\n", operation.DestinationLocal, leftValue, rightValue)
			}
		case "&", "^", "|":
			if operation.OwnerType == hir.TypeI32 {
				fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32((uint32_t)(%s) %s (uint32_t)(%s));\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
			} else {
				fmt.Fprintf(output, "    fy_l_%d = (%s) %s (%s);\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
			}
		case "==", "!=", "<", "<=", ">", ">=":
			fmt.Fprintf(output, "    fy_l_%d = ((%s) %s (%s)) ? INT32_C(1) : INT32_C(0);\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
		}
	case mir.OperationResultConstruct:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Result construction in %s: %w", function, err)
		}
		payload, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Result construction in %s: %w", function, err)
		}
		tag := "FY_RESULT_OK"
		field := "payload"
		cType := "fy_result_i32_i32"
		if operation.Variant == "err" {
			tag = "FY_RESULT_ERR"
		}
		if index, resourceTagged := e.analysis.taggedByType[operation.Type]; resourceTagged {
			cType = fmt.Sprintf("fy_tagged_%d", index)
			field = "p_0"
			if operation.Variant == "err" {
				field = "p_1"
			}
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s){ .tag = %s, .%s = %s };\n", operation.DestinationLocal, cType, tag, field, payload)
	case mir.OperationResultPayload:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Result payload in %s: %w", function, err)
		}
		container, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Result payload in %s: %w", function, err)
		}
		containerType, err := e.analysis.operandType(functionIndex, operand)
		if err != nil {
			return fmt.Errorf("Result payload in %s: %w", function, err)
		}
		field := "payload"
		if _, resourceTagged := e.analysis.taggedByType[containerType]; resourceTagged {
			field = "p_0"
			if operation.Variant == "err" {
				field = "p_1"
			}
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s).%s;\n", operation.DestinationLocal, container, field)
	case mir.OperationOptionConstruct:
		cType := "fy_option_i32"
		field := "payload"
		nullNiche := sharedBorrowOptionNiche(operation.Type, e.analysis.structByType, e.analysis.resourceByType, e.analysis.arrayByType)
		if index, resourceTagged := e.analysis.taggedByType[operation.Type]; resourceTagged {
			cType = fmt.Sprintf("fy_tagged_%d", index)
			field = "p_0"
		}
		if operation.Variant == "none" {
			if nullNiche {
				fmt.Fprintf(output, "    fy_l_%d = (%s){ .p_0 = ((void *)0) };\n", operation.DestinationLocal, cType)
			} else {
				fmt.Fprintf(output, "    fy_l_%d = (%s){ .tag = FY_OPTION_NONE };\n", operation.DestinationLocal, cType)
			}
			break
		}
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Option::Some construction in %s: %w", function, err)
		}
		payload, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Option::Some construction in %s: %w", function, err)
		}
		if nullNiche {
			fmt.Fprintf(output, "    fy_l_%d = (%s){ .p_0 = %s };\n", operation.DestinationLocal, cType, payload)
		} else {
			fmt.Fprintf(output, "    fy_l_%d = (%s){ .tag = FY_OPTION_SOME, .%s = %s };\n", operation.DestinationLocal, cType, field, payload)
		}
	case mir.OperationOptionPayload:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Option payload in %s: %w", function, err)
		}
		container, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Option payload in %s: %w", function, err)
		}
		containerType, err := e.analysis.operandType(functionIndex, operand)
		if err != nil {
			return fmt.Errorf("Option payload in %s: %w", function, err)
		}
		field := "payload"
		if _, resourceTagged := e.analysis.taggedByType[containerType]; resourceTagged {
			field = "p_0"
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s).%s;\n", operation.DestinationLocal, container, field)
	case mir.OperationUse, mir.OperationArithmeticCheck, mir.OperationCheckedDivision, mir.OperationViewIndexCheck, mir.OperationViewSliceCheck,
		mir.OperationArrayIndexCheck, mir.OperationResultSwitch:
	case mir.OperationDropFlagInit, mir.OperationDropFlagClear:
		return CapabilityError{Function: function, Feature: "MIR operation " + strconv.Quote(operation.Kind.String())}
	default:
		return CapabilityError{Function: function, Feature: "MIR operation " + strconv.Quote(operation.Kind.String())}
	}
	return nil
}
