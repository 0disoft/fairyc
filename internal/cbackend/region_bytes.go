package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/mir"
)

func (e *emitter) emitRegionBytes(output *strings.Builder, functionIndex int, operation mir.Operation) error {
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return err
	}
	value, err := cOperand(operand)
	if err != nil {
		return err
	}
	taggedIndex, found := e.analysis.taggedByType[operation.Type]
	if !found {
		return fmt.Errorf("Region byte allocation has no tagged C type")
	}
	_, failureType, _, _ := e.analysis.resultCSpec(operation.Type)
	errorIndex, found := e.analysis.structByType[failureType]
	if !found {
		return fmt.Errorf("Region byte allocation has no error C type")
	}
	id := operation.DestinationLocal
	// The compound statement also makes this valid directly after a C11 label.
	output.WriteString("    {\n")
	fmt.Fprintf(output, "    void *fy_region_storage_%d = NULL;\n", id)
	if operation.Operator == "copy_bytes" {
		fmt.Fprintf(output, "    uintptr_t fy_region_length_%d = (%s).length;\n", id, value)
		fmt.Fprintf(output, "    uint32_t fy_region_status_%d = fy_std_region_copy_bytes_v0((void *)&fy_region_%d, (%s).data, (size_t)fy_region_length_%d, _Alignof(uint8_t), &fy_region_storage_%d);\n", id, operation.Symbol, value, id, id)
	} else {
		fmt.Fprintf(output, "    uintptr_t fy_region_length_%d = %s;\n", id, value)
		fmt.Fprintf(output, "    uint32_t fy_region_status_%d = fy_std_region_allocate_zeroed_v0((void *)&fy_region_%d, (size_t)fy_region_length_%d, _Alignof(uint8_t), &fy_region_storage_%d);\n", id, operation.Symbol, id, id)
	}
	fmt.Fprintf(output, "    if (fy_region_status_%d == UINT32_C(0)) {\n", id)
	fmt.Fprintf(output, "        fy_l_%d = (fy_tagged_%d){ .tag = FY_RESULT_OK, .p_0 = (fy_mut_byte_slice){ .data = (uint8_t *)fy_region_storage_%d, .length = fy_region_length_%d } };\n", id, taggedIndex, id, id)
	output.WriteString("    } else {\n")
	fmt.Fprintf(output, "        fy_l_%d = (fy_tagged_%d){ .tag = FY_RESULT_ERR, .p_1 = (%s){ .%s = fy_region_status_%d } };\n", id, taggedIndex, e.analysis.structCTypeName(errorIndex), e.analysis.structCFieldName(errorIndex, 0), id)
	output.WriteString("    }\n    }\n")
	return nil
}
