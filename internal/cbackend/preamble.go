package cbackend

import (
	"fmt"
	"slices"
	"strings"
)

func (e *emitter) emitPreamble(output *strings.Builder) error {
	_, usesCSSizeT := e.analysis.features["c-ssize-t"]
	if usesCSSizeT {
		output.WriteString("#define _POSIX_C_SOURCE 200809L\n")
	}
	transportProfile := "ISO C11"
	if usesCSSizeT {
		transportProfile += " with POSIX.1-2008 typedefs"
	}
	fmt.Fprintf(output, "/* fairyc constrained C transport: %s, %s */\n", e.analysis.targetID, transportProfile)
	output.WriteString("#include <stdint.h>\n")
	_, usesF32 := e.analysis.features["fairy-f32-transport"]
	_, usesF64 := e.analysis.features["fairy-f64-transport"]
	if usesF32 || usesF64 {
		output.WriteString("#include <float.h>\n")
	}
	_, usesCSizeT := e.analysis.features["c-size-t"]
	_, usesRegion := e.analysis.features["region-lifecycle"]
	if len(e.analysis.structTypes) != 0 || len(e.analysis.enumTypes) != 0 || len(e.analysis.resourceTypes) != 0 || len(e.analysis.taggedTypes) != 0 || usesCSizeT || usesRegion {
		output.WriteString("#include <stddef.h>\n")
	}
	if usesCSSizeT {
		output.WriteString("#include <sys/types.h>\n")
	}
	_, usesStorageTake := e.analysis.features["private-storage-take"]
	_, usesStoragePut := e.analysis.features["private-storage-put"]
	if e.analysis.needsBitcast || e.analysis.i64Helpers.needsBitcast() || usesStorageTake || usesStoragePut {
		output.WriteString("#include <string.h>\n")
	}
	output.WriteByte('\n')
	if err := e.emitCallbackType(output); err != nil {
		return err
	}
	if usesRegion {
		output.WriteString("typedef struct fy_region {\n")
		output.WriteString("    void *arena;\n    void *mark;\n    uint64_t generation;\n    uint8_t active;\n")
		output.WriteString("} fy_region;\n")
		output.WriteString("extern uint32_t fy_std_region_begin_v0(void *arena, void *output);\n")
		output.WriteString("extern uint32_t fy_std_region_allocate_zeroed_v0(void *region, size_t size, size_t alignment, void **output);\n")
		if _, copyBytes := e.analysis.usedForeignSymbols["fy_std_region_copy_bytes_v0"]; copyBytes {
			output.WriteString("extern uint32_t fy_std_region_copy_bytes_v0(void *region, const void *source, size_t size, size_t alignment, void **output);\n")
		}
		output.WriteString("extern uint32_t fy_std_region_reset_v0(void *region);\n\n")
	}
	_, usesStaticByteString := e.analysis.features["static-byte-string"]
	if e.analysis.needsMutStructSlice {
		output.WriteString("typedef struct fy_mut_struct_slice { void *data; uintptr_t length; } fy_mut_struct_slice;\n\n")
	}
	if e.analysis.needsStructSlice {
		output.WriteString("typedef struct fy_struct_slice { const void *data; uintptr_t length; } fy_struct_slice;\n\n")
	}
	e.emitIntegerViewTypes(output)
	if e.analysis.needsI64Slice {
		output.WriteString("typedef struct fy_i64_slice {\n    const int64_t *data;\n    uintptr_t length;\n} fy_i64_slice;\n\n")
	}
	if e.analysis.needsMutI64Slice {
		output.WriteString("typedef struct fy_mut_i64_slice {\n    int64_t *data;\n    uintptr_t length;\n} fy_mut_i64_slice;\n\n")
	}
	_, usesArrayByteView := e.analysis.features["array-byte-view-borrow"]
	if usesStaticByteString || usesArrayByteView || e.analysis.needsByteSlice {
		output.WriteString("typedef struct fy_byte_slice {\n")
		output.WriteString("    const uint8_t *data;\n")
		output.WriteString("    uintptr_t length;\n")
		output.WriteString("} fy_byte_slice;\n\n")
	}
	if _, used := e.analysis.features["mutable-array-byte-view-borrow"]; used || e.analysis.needsMutByteSlice {
		output.WriteString("typedef struct fy_mut_byte_slice {\n")
		output.WriteString("    uint8_t *data;\n")
		output.WriteString("    uintptr_t length;\n")
		output.WriteString("} fy_mut_byte_slice;\n\n")
	}
	_, usesStaticUTF8String := e.analysis.features["static-utf8-string"]
	_, usesUTF8StringView := e.analysis.features["utf8-string-view"]
	if usesStaticUTF8String || usesUTF8StringView || e.analysis.needsStringView {
		output.WriteString("typedef struct fy_str {\n")
		output.WriteString("    const uint8_t *data;\n")
		output.WriteString("    uintptr_t length;\n")
		output.WriteString("} fy_str;\n\n")
	}
	for _, transport := range cScalarTransports {
		if _, used := e.analysis.features[transport.feature]; !used {
			continue
		}
		scalar, available := e.analysis.cScalars[transport.sourceType]
		if !available {
			return CapabilityError{Feature: fmt.Sprintf("C scalar %q on target %q", transport.targetName, e.analysis.targetID)}
		}
		fmt.Fprintf(output, "_Static_assert(sizeof(%s) == UINT64_C(%d), \"fairyc target %s size\");\n", transport.cType, scalar.Layout.SizeBytes, transport.targetName)
		fmt.Fprintf(output, "_Static_assert(_Alignof(%s) == UINT64_C(%d), \"fairyc target %s alignment\");\n", transport.cType, scalar.Layout.AlignmentBytes, transport.targetName)
		switch scalar.Signedness {
		case "signed":
			fmt.Fprintf(output, "_Static_assert(((%s)-1) < (%s)0, \"fairyc target %s signed\");\n", transport.cType, transport.cType, transport.targetName)
		case "unsigned":
			fmt.Fprintf(output, "_Static_assert(((%s)-1) > (%s)0, \"fairyc target %s unsigned\");\n", transport.cType, transport.cType, transport.targetName)
		case "not-applicable":
		default:
			return fmt.Errorf("C scalar %q has invalid signedness %q", transport.targetName, scalar.Signedness)
		}
		output.WriteByte('\n')
	}
	_, scalarOption := e.analysis.features["scalar-option"]
	if err := e.emitFloatingTransportAssertions(output); err != nil {
		return err
	}
	if err := e.emitIntegerTransportAssertions(output); err != nil {
		return err
	}
	_, copyOption := e.analysis.features["copy-option"]
	_, resourceOption := e.analysis.features["resource-option"]
	if scalarOption || copyOption || resourceOption {
		output.WriteString("#define FY_OPTION_NONE UINT32_C(0)\n")
		output.WriteString("#define FY_OPTION_SOME UINT32_C(1)\n")
	}
	if scalarOption {
		output.WriteString("typedef struct { uint32_t tag; int32_t payload; } fy_option_i32;\n\n")
	}
	_, scalarResult := e.analysis.features["scalar-result"]
	_, copyResult := e.analysis.features["copy-result"]
	_, resourceResult := e.analysis.features["resource-result"]
	if scalarResult || copyResult || resourceResult {
		output.WriteString("#define FY_RESULT_OK UINT32_C(0)\n")
		output.WriteString("#define FY_RESULT_ERR UINT32_C(1)\n")
	}
	if scalarResult {
		output.WriteString("typedef struct { uint32_t tag; int32_t payload; } fy_result_i32_i32;\n\n")
	}
	if err := e.emitTaggedTypes(output, true); err != nil {
		return err
	}
	if err := e.emitScalarArrayTypes(output, false); err != nil {
		return err
	}
	if err := e.emitAggregateTypes(output); err != nil {
		return err
	}
	for index, resource := range e.analysis.resourceTypes {
		output.WriteString("typedef struct {\n")
		for fieldIndex, field := range resource.definition.Fields {
			cType, err := e.analysis.cValueType(field.Type)
			if err != nil {
				return fmt.Errorf("resource type %q field %q: %w", resource.definition.Type, field.Name, err)
			}
			fmt.Fprintf(output, "    %s f_%d;\n", cType, fieldIndex)
		}
		fmt.Fprintf(output, "} fy_resource_%d;\n", index)
		fmt.Fprintf(output, "_Static_assert(sizeof(fy_resource_%d) == UINT64_C(%d), \"fairyc resource %d size\");\n", index, resource.layout.SizeBytes, index)
		fmt.Fprintf(output, "_Static_assert(_Alignof(fy_resource_%d) == UINT64_C(%d), \"fairyc resource %d alignment\");\n", index, resource.layout.AlignmentBytes, index)
		for fieldIndex, field := range resource.layout.Fields {
			fmt.Fprintf(output, "_Static_assert(offsetof(fy_resource_%d, f_%d) == UINT64_C(%d), \"fairyc resource %d field %d offset\");\n", index, fieldIndex, field.OffsetBytes, index, fieldIndex)
		}
		output.WriteByte('\n')
	}
	if err := e.emitTaggedTypes(output, false); err != nil {
		return err
	}
	if e.analysis.instrumentation == InstrumentationDropTraceV0 {
		output.WriteString("void fy_runtime_trace_drop(uint32_t site_id);\n\n")
	}
	if len(e.analysis.trapCategories) != 0 {
		categories := make([]string, 0, len(e.analysis.trapCategories))
		for category := range e.analysis.trapCategories {
			categories = append(categories, category)
		}
		slices.Sort(categories)
		for _, category := range categories {
			name, code, _ := cTrapSpec(category)
			fmt.Fprintf(output, "#define %s UINT32_C(%d)\n", name, code)
		}
		output.WriteString("_Noreturn void fy_runtime_trap(uint32_t category);\n\n")
		output.WriteString("_Noreturn void fy_runtime_trap_at_v0(uint32_t category, const char *source, uint32_t source_length, uint64_t start_byte, uint64_t end_byte);\n\n")
		output.WriteString("struct fy_runtime_location_v0 { const char *source; uint32_t length; uint64_t start, end; };\n\n")
	}
	if e.analysis.needsBitcast {
		output.WriteString("static int32_t fy_bits_to_i32(uint32_t bits) {\n")
		output.WriteString("    int32_t value;\n    memcpy(&value, &bits, sizeof value);\n    return value;\n}\n\n")
	}
	if e.analysis.needsShift {
		output.WriteString("static int32_t fy_sar_i32(int32_t value, int32_t amount) {\n")
		output.WriteString("    uint32_t bits = (uint32_t)value;\n")
		output.WriteString("    if (amount == 0 || value >= 0) {\n        return fy_bits_to_i32(bits >> (uint32_t)amount);\n    }\n")
		output.WriteString("    return fy_bits_to_i32((bits >> (uint32_t)amount) | (UINT32_MAX << (32u - (uint32_t)amount)));\n}\n\n")
	}
	e.analysis.i64Helpers.emit(output)
	return nil
}
