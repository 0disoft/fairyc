// Package msvccprobe emits private C11 types shared by Windows MSVC evidence probes.
package msvccprobe

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/target"
)

// TypeEmitter emits deterministic private typedefs and target-derived layout assertions.
type TypeEmitter struct {
	engine *layout.Engine
	output strings.Builder
	nextID int
}

// NewTypeEmitter binds private C type emission to one validated Windows MSVC target.
func NewTypeEmitter(spec target.Spec) (*TypeEmitter, error) {
	if spec.ID != target.X8664WindowsMSVC || spec.CallingConvention != "microsoft-x64" || spec.AggregateLayout != "msvc-x64-llp64" {
		return nil, fmt.Errorf("MSVC C probe target is %q, want %q", spec.ID, target.X8664WindowsMSVC)
	}
	if err := target.Validate(spec); err != nil {
		return nil, err
	}
	engine, err := layout.New(spec)
	if err != nil {
		return nil, err
	}
	return &TypeEmitter{engine: engine}, nil
}

// EmitType emits one recursively closed type and returns its private C typedef name.
func (e *TypeEmitter) EmitType(value layout.Type) (string, error) {
	if e == nil || e.engine == nil {
		return "", fmt.Errorf("MSVC C probe type emitter is nil")
	}
	name := fmt.Sprintf("fy_windows_probe_type_%d", e.nextID)
	id := e.nextID
	e.nextID++
	fieldTypes := make([]string, 0, len(value.Fields))
	switch value.Kind {
	case layout.KindLanguageScalar:
		spelling, found := LanguageScalarSpelling(value.Name)
		if !found {
			return "", fmt.Errorf("unsupported language scalar %q", value.Name)
		}
		fmt.Fprintf(&e.output, "typedef %s %s;\n", spelling, name)
	case layout.KindCScalar, layout.KindCEnum:
		spelling, found := CScalarSpelling(value.Name)
		if !found {
			return "", fmt.Errorf("unsupported Windows C scalar base %q", value.Name)
		}
		fmt.Fprintf(&e.output, "typedef %s %s;\n", spelling, name)
	case layout.KindPointer:
		fmt.Fprintf(&e.output, "typedef void *%s;\n", name)
	case layout.KindFunctionPtr:
		fmt.Fprintf(&e.output, "typedef void (*%s)(void);\n", name)
	case layout.KindArray:
		if value.Element == nil || value.Length == 0 {
			return "", fmt.Errorf("array probe is missing a positive element count")
		}
		element, err := e.EmitType(*value.Element)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&e.output, "typedef %s %s[UINT64_C(%d)];\n", element, name, value.Length)
	case layout.KindStruct:
		if len(value.Fields) == 0 {
			return "", fmt.Errorf("struct probe has no fields")
		}
		for _, field := range value.Fields {
			fieldType, err := e.EmitType(field.Type)
			if err != nil {
				return "", err
			}
			fieldTypes = append(fieldTypes, fieldType)
		}
		fmt.Fprintf(&e.output, "typedef struct fy_windows_probe_tag_%d {\n", id)
		for index, fieldType := range fieldTypes {
			fmt.Fprintf(&e.output, "    %s f_%d;\n", fieldType, index)
		}
		fmt.Fprintf(&e.output, "} %s;\n", name)
	default:
		return "", fmt.Errorf("unsupported layout kind %q", value.Kind)
	}
	computed, err := e.engine.Of(value)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&e.output, "_Static_assert(sizeof(%s) == UINT64_C(%d), \"fairyc Windows probe type %d size\");\n", name, computed.SizeBytes, id)
	fmt.Fprintf(&e.output, "_Static_assert(_Alignof(%s) == UINT64_C(%d), \"fairyc Windows probe type %d alignment\");\n", name, computed.AlignmentBytes, id)
	for index, field := range computed.Fields {
		fmt.Fprintf(&e.output, "_Static_assert(offsetof(%s, f_%d) == UINT64_C(%d), \"fairyc Windows probe type %d field %d offset\");\n", name, index, field.OffsetBytes, id, index)
		fmt.Fprintf(&e.output, "_Static_assert(sizeof(((%s *)0)->f_%d) == UINT64_C(%d), \"fairyc Windows probe type %d field %d size\");\n", name, index, field.SizeBytes, id, index)
		fmt.Fprintf(&e.output, "_Static_assert(_Alignof(%s) == UINT64_C(%d), \"fairyc Windows probe type %d field %d alignment\");\n", fieldTypes[index], field.AlignmentBytes, id, index)
	}
	e.output.WriteByte('\n')
	return name, nil
}

// Source returns every type emitted so far.
func (e *TypeEmitter) Source() string {
	if e == nil {
		return ""
	}
	return e.output.String()
}

// LanguageScalarSpelling maps one declared language scalar to its private C11 spelling.
func LanguageScalarSpelling(name string) (string, bool) {
	switch name {
	case "bool", "u8":
		return "uint8_t", true
	case "i8":
		return "int8_t", true
	case "u16":
		return "uint16_t", true
	case "i16":
		return "int16_t", true
	case "u32":
		return "uint32_t", true
	case "i32":
		return "int32_t", true
	case "u64":
		return "uint64_t", true
	case "i64":
		return "int64_t", true
	case "usize":
		return "uintptr_t", true
	case "isize":
		return "intptr_t", true
	case "f32":
		return "float", true
	case "f64":
		return "double", true
	default:
		return "", false
	}
}

// CScalarSpelling maps one declared Windows C scalar to its private C11 spelling.
func CScalarSpelling(name string) (string, bool) {
	switch name {
	case "c.bool":
		return "_Bool", true
	case "c.char":
		return "char", true
	case "c.double":
		return "double", true
	case "c.float":
		return "float", true
	case "c.int":
		return "int", true
	case "c.long":
		return "long", true
	case "c.longlong":
		return "long long", true
	case "c.schar":
		return "signed char", true
	case "c.short":
		return "short", true
	case "c.size_t":
		return "size_t", true
	case "c.uchar":
		return "unsigned char", true
	case "c.uint":
		return "unsigned int", true
	case "c.ulong":
		return "unsigned long", true
	case "c.ulonglong":
		return "unsigned long long", true
	case "c.ushort":
		return "unsigned short", true
	default:
		return "", false
	}
}
