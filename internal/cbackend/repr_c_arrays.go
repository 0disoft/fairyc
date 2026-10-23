package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func (e *emissionAnalysis) rawArrayMember(owner, field hir.Type) bool {
	index, ok := e.structByType[fieldAggregateType(owner)]
	return ok && e.structTypes[index].definition.ReprC && e.reprCArrayField(field)
}

// Copy actual C array elements, never reinterpret them as private wrapper structs.
// The operands are already-evaluated MIR locals; no source expression is repeated.
func writeRawArrayCopy(output *strings.Builder, destination, source string, typ hir.Type) {
	_, length, _ := hir.FixedArrayParts(typ)
	fmt.Fprintf(output, "    { for (uintptr_t fy_array_i = 0; fy_array_i < UINT64_C(%d); ++fy_array_i) { (%s)[fy_array_i] = (%s)[fy_array_i]; } }\n", length, destination, source)
}

func (e *emitter) writeReprCArrayConstruction(output *strings.Builder, operation mir.Operation, index int) error {
	definition := e.analysis.structTypes[index].definition
	fmt.Fprintf(output, "    fy_l_%d = (%s){ 0 };\n", operation.DestinationLocal, e.analysis.structCTypeName(index))
	for fieldIndex, field := range definition.Fields {
		value, err := cOperand(operation.Operands[fieldIndex])
		if err != nil {
			return err
		}
		place := fmt.Sprintf("fy_l_%d.%s", operation.DestinationLocal, field.Name)
		if e.analysis.reprCArrayField(field.Type) {
			writeRawArrayCopy(output, place, "("+value+").data", field.Type)
		} else {
			fmt.Fprintf(output, "    %s = %s;\n", place, value)
		}
	}
	return nil
}
