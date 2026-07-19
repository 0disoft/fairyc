// Package windowslayoutprobe generates the closed MSVC C layout oracle for the
// declared x86_64 Windows profile.
package windowslayoutprobe

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/layoutfixture"
	"github.com/0disoft/fairyc/internal/target"
)

// Unit is one deterministic MSVC C translation unit derived from validated fixtures.
type Unit struct {
	Target target.ID
	Cases  []string
	Source string
}

type emitter struct {
	engine *layout.Engine
	output strings.Builder
	nextID int
}

// Generate validates the Windows fixture manifest and translates every pass case
// into C11 size, alignment, field-size, and offset assertions.
func Generate(manifestPath string) (Unit, error) {
	targetID, cases, err := layoutfixture.NativeProbeCases(manifestPath)
	if err != nil {
		return Unit{}, err
	}
	if targetID != target.X8664WindowsMSVC {
		return Unit{}, fmt.Errorf("Windows MSVC layout probe target is %q, want %q", targetID, target.X8664WindowsMSVC)
	}
	spec, found := target.Lookup(targetID)
	if !found {
		return Unit{}, fmt.Errorf("Windows MSVC layout probe target %q is unknown", targetID)
	}
	engine, err := layout.New(spec)
	if err != nil {
		return Unit{}, err
	}
	value := emitter{engine: engine}
	value.output.WriteString("/* fairyc Windows MSVC layout probe: numeric private types, ISO C11 */\n")
	value.output.WriteString("#include <stddef.h>\n#include <stdint.h>\n\n")
	caseIDs := make([]string, 0, len(cases))
	previous := ""
	for _, probeCase := range cases {
		if probeCase.ID == "" || previous != "" && previous >= probeCase.ID {
			return Unit{}, fmt.Errorf("Windows MSVC layout probe case IDs are empty, duplicate, or unsorted")
		}
		previous = probeCase.ID
		actual, err := engine.Of(probeCase.Type)
		if err != nil {
			return Unit{}, fmt.Errorf("Windows MSVC layout probe case %q: %w", probeCase.ID, err)
		}
		if !equalResult(actual, probeCase.Layout) {
			return Unit{}, fmt.Errorf("Windows MSVC layout probe case %q drifted from its fixture result", probeCase.ID)
		}
		if _, err := value.emitType(probeCase.Type); err != nil {
			return Unit{}, fmt.Errorf("Windows MSVC layout probe case %q: %w", probeCase.ID, err)
		}
		caseIDs = append(caseIDs, probeCase.ID)
	}
	value.output.WriteString("int main(void) { return 0; }\n")
	return Unit{Target: targetID, Cases: caseIDs, Source: value.output.String()}, nil
}

func (e *emitter) emitType(value layout.Type) (string, error) {
	name := fmt.Sprintf("fy_windows_layout_type_%d", e.nextID)
	id := e.nextID
	e.nextID++
	fieldTypes := make([]string, 0, len(value.Fields))
	switch value.Kind {
	case layout.KindLanguageScalar:
		spelling, found := languageScalarSpelling(value.Name)
		if !found {
			return "", fmt.Errorf("unsupported language scalar %q", value.Name)
		}
		fmt.Fprintf(&e.output, "typedef %s %s;\n", spelling, name)
	case layout.KindCScalar, layout.KindCEnum:
		spelling, found := cScalarSpelling(value.Name)
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
		element, err := e.emitType(*value.Element)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&e.output, "typedef %s %s[UINT64_C(%d)];\n", element, name, value.Length)
	case layout.KindStruct:
		if len(value.Fields) == 0 {
			return "", fmt.Errorf("struct probe has no fields")
		}
		for _, field := range value.Fields {
			fieldType, err := e.emitType(field.Type)
			if err != nil {
				return "", err
			}
			fieldTypes = append(fieldTypes, fieldType)
		}
		fmt.Fprintf(&e.output, "typedef struct fy_windows_layout_tag_%d {\n", id)
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
	fmt.Fprintf(&e.output, "_Static_assert(sizeof(%s) == UINT64_C(%d), \"fairyc Windows layout type %d size\");\n", name, computed.SizeBytes, id)
	fmt.Fprintf(&e.output, "_Static_assert(_Alignof(%s) == UINT64_C(%d), \"fairyc Windows layout type %d alignment\");\n", name, computed.AlignmentBytes, id)
	for index, field := range computed.Fields {
		fmt.Fprintf(&e.output, "_Static_assert(offsetof(%s, f_%d) == UINT64_C(%d), \"fairyc Windows layout type %d field %d offset\");\n", name, index, field.OffsetBytes, id, index)
		fmt.Fprintf(&e.output, "_Static_assert(sizeof(((%s *)0)->f_%d) == UINT64_C(%d), \"fairyc Windows layout type %d field %d size\");\n", name, index, field.SizeBytes, id, index)
		fmt.Fprintf(&e.output, "_Static_assert(_Alignof(%s) == UINT64_C(%d), \"fairyc Windows layout type %d field %d alignment\");\n", fieldTypes[index], field.AlignmentBytes, id, index)
	}
	e.output.WriteByte('\n')
	return name, nil
}

func equalResult(left, right layout.Result) bool {
	return left.SizeBytes == right.SizeBytes && left.AlignmentBytes == right.AlignmentBytes &&
		slices.Equal(left.Fields, right.Fields) && slices.Equal(left.Padding, right.Padding)
}

func languageScalarSpelling(name string) (string, bool) {
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

func cScalarSpelling(name string) (string, bool) {
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
