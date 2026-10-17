package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func (e *analysisBuilder) inspectFieldStore(functionIndex int, operation mir.Operation) error {
	valid := mir.ValidFieldStoreShape(operation) || mir.ValidPlainStructFieldStoreShape(operation, e.fieldStoreStruct)
	if index, found := e.structByType[operation.Type]; found {
		valid = valid || mir.ValidOneWordStructFieldStoreShape(operation, &e.structTypes[index].definition)
	}
	if !valid {
		return fmt.Errorf("invalid field store shape")
	}
	valueType, err := e.operandType(functionIndex, operation.Operands[1])
	if err != nil || valueType != operation.Type {
		return fmt.Errorf("field store value type %q does not match %q: %v", valueType, operation.Type, err)
	}
	if len(operation.Fields) > 0 {
		receiverType, err := e.operandType(functionIndex, operation.Operands[0])
		if err != nil || receiverType != operation.OwnerType {
			return fmt.Errorf("nested field store receiver type mismatch")
		}
		if _, ok := mir.ResolveNestedFieldStore(operation, e.fieldStoreStruct); !ok {
			return fmt.Errorf("invalid nested field store path")
		}
		return nil
	}
	// The read inspector owns the exact declared-field and receiver-type checks.
	operation.Kind, operation.Operands = mir.OperationFieldRead, operation.Operands[:1]
	if _, found := e.structByType[fieldAggregateType(operation.OwnerType)]; found {
		return e.inspectStructOperation(functionIndex, operation)
	}
	return e.inspectResourceOperation(functionIndex, operation)
}

func (e *emitter) writeFieldStore(output *strings.Builder, operation mir.Operation) error {
	receiver, err := cOperand(operation.Operands[0])
	if err != nil {
		return err
	}
	value, err := cOperand(operation.Operands[1])
	if err != nil {
		return err
	}
	if len(operation.Fields) > 0 {
		path, ok := mir.ResolveNestedFieldStore(operation, e.analysis.fieldStoreStruct)
		if !ok {
			return fmt.Errorf("invalid nested field store path")
		}
		typ := fieldAggregateType(operation.OwnerType)
		place := "(" + receiver + ")"
		access := "."
		if _, _, borrowed := hir.BorrowElement(operation.OwnerType); borrowed {
			access = "->"
		}
		for _, field := range path {
			index := e.analysis.structByType[typ]
			place += access + e.analysis.structCFieldName(index, e.analysis.structTypes[index].fieldIndex[field.Name])
			access = "."
			typ = field.Type
		}
		fmt.Fprintf(output, "    %s = %s;\n", place, value)
		return nil
	}
	aggregate := fieldAggregateType(operation.OwnerType)
	name := ""
	if index, found := e.analysis.structByType[aggregate]; found {
		field, found := e.analysis.structTypes[index].fieldIndex[operation.Field]
		if !found {
			return fmt.Errorf("missing stored struct field")
		}
		name = e.analysis.structCFieldName(index, field)
	} else if index, found := e.analysis.resourceByType[aggregate]; found {
		field, found := e.analysis.resourceTypes[index].fieldIndex[operation.Field]
		if !found {
			return fmt.Errorf("missing stored resource field")
		}
		name = fmt.Sprintf("f_%d", field)
	} else {
		return fmt.Errorf("missing field store owner")
	}
	access := "."
	if _, _, borrowed := hir.BorrowElement(operation.OwnerType); borrowed {
		access = "->"
	}
	fmt.Fprintf(output, "    (%s)%s%s = %s;\n", receiver, access, name, value)
	return nil
}

func (e *emissionAnalysis) fieldStoreStruct(typ hir.Type) *mir.StructType {
	if index, ok := e.structByType[typ]; ok {
		return &e.structTypes[index].definition
	}
	return nil
}
