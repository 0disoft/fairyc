// Package windowsabiprobe generates closed bidirectional C sources for the
// validated Microsoft x64 fixture subset.
package windowsabiprobe

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/cabi"
	"github.com/0disoft/fairyc/internal/cabifixture"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/msvccprobe"
	"github.com/0disoft/fairyc/internal/target"
)

// Unit contains two separately compiled C translation units covering every pass case.
type Unit struct {
	Target       target.ID
	Cases        []string
	CallerSource string
	CalleeSource string
}

type emittedCase struct {
	Signature      cabi.Signature
	ResultType     string
	ParameterTypes []string
}

// Generate validates the Windows C ABI manifest and emits one caller/callee pair.
func Generate(manifestPath string) (Unit, error) {
	targetID, cases, err := cabifixture.NativeProbeCases(manifestPath)
	if err != nil {
		return Unit{}, err
	}
	if targetID != target.X8664WindowsMSVC {
		return Unit{}, fmt.Errorf("Windows MSVC ABI probe target is %q, want %q", targetID, target.X8664WindowsMSVC)
	}
	spec, found := target.Lookup(targetID)
	if !found {
		return Unit{}, fmt.Errorf("Windows MSVC ABI probe target %q is unknown", targetID)
	}
	classifier, err := cabi.New(spec)
	if err != nil {
		return Unit{}, err
	}
	typeEmitter, err := msvccprobe.NewTypeEmitter(spec)
	if err != nil {
		return Unit{}, err
	}
	emitted := make([]emittedCase, 0, len(cases))
	caseIDs := make([]string, 0, len(cases))
	previous := ""
	for _, fixture := range cases {
		if fixture.ID == "" || previous != "" && previous >= fixture.ID {
			return Unit{}, fmt.Errorf("Windows MSVC ABI probe case IDs are empty, duplicate, or unsorted")
		}
		previous = fixture.ID
		plan, err := classifier.Plan(fixture.Signature)
		if err != nil {
			return Unit{}, fmt.Errorf("Windows MSVC ABI probe case %q: %w", fixture.ID, err)
		}
		if !reflect.DeepEqual(plan, fixture.Plan) {
			return Unit{}, fmt.Errorf("Windows MSVC ABI probe case %q drifted from its fixture plan", fixture.ID)
		}
		value := emittedCase{Signature: fixture.Signature, ResultType: "void", ParameterTypes: make([]string, len(fixture.Signature.Parameters))}
		if fixture.Signature.Result != nil {
			value.ResultType, err = typeEmitter.EmitType(*fixture.Signature.Result)
			if err != nil {
				return Unit{}, fmt.Errorf("Windows MSVC ABI probe case %q result: %w", fixture.ID, err)
			}
		}
		for index, parameter := range fixture.Signature.Parameters {
			value.ParameterTypes[index], err = typeEmitter.EmitType(parameter)
			if err != nil {
				return Unit{}, fmt.Errorf("Windows MSVC ABI probe case %q parameter %d: %w", fixture.ID, index, err)
			}
		}
		emitted = append(emitted, value)
		caseIDs = append(caseIDs, fixture.ID)
	}
	common := emitCommon(typeEmitter.Source(), emitted)
	caller, err := emitCaller(common, emitted)
	if err != nil {
		return Unit{}, err
	}
	callee, err := emitCallee(common, emitted)
	if err != nil {
		return Unit{}, err
	}
	return Unit{Target: targetID, Cases: caseIDs, CallerSource: caller, CalleeSource: callee}, nil
}

func emitCommon(typeSource string, cases []emittedCase) string {
	var output strings.Builder
	output.WriteString("/* fairyc Windows MSVC ABI probe: separate ISO C11 caller and callee */\n")
	output.WriteString("#include <stddef.h>\n#include <stdint.h>\n\n")
	output.WriteString(typeSource)
	output.WriteString("void fy_windows_abi_callback(void);\n")
	for index, fixture := range cases {
		fmt.Fprintf(&output, "extern int fy_windows_abi_failure_%d;\n", index)
		fmt.Fprintf(&output, "%s fy_windows_abi_target_%d(%s);\n", fixture.ResultType, index, parameterList(fixture.ParameterTypes))
	}
	output.WriteByte('\n')
	return output.String()
}

func emitCaller(common string, cases []emittedCase) (string, error) {
	var output strings.Builder
	output.WriteString(common)
	for index := range cases {
		fmt.Fprintf(&output, "int fy_windows_abi_failure_%d = 0;\n", index)
	}
	output.WriteString("void fy_windows_abi_callback(void) {}\n\n")
	output.WriteString("int main(void) {\n")
	for caseIndex, fixture := range cases {
		for parameterIndex, parameter := range fixture.Signature.Parameters {
			fmt.Fprintf(&output, "    %s c_%d_p_%d = {0};\n", fixture.ParameterTypes[parameterIndex], caseIndex, parameterIndex)
			if err := emitInitialization(&output, parameter, fmt.Sprintf("c_%d_p_%d", caseIndex, parameterIndex), seed(caseIndex, parameterIndex), "    "); err != nil {
				return "", fmt.Errorf("initialize Windows ABI case %d parameter %d: %w", caseIndex, parameterIndex, err)
			}
		}
		arguments := argumentList(caseIndex, len(fixture.ParameterTypes))
		if fixture.Signature.Result == nil {
			fmt.Fprintf(&output, "    fy_windows_abi_target_%d(%s);\n", caseIndex, arguments)
		} else {
			fmt.Fprintf(&output, "    %s c_%d_result = fy_windows_abi_target_%d(%s);\n", fixture.ResultType, caseIndex, caseIndex, arguments)
		}
		fmt.Fprintf(&output, "    if (fy_windows_abi_failure_%d != 0) { return %d; }\n", caseIndex, caseIndex+1)
		if fixture.Signature.Result != nil {
			check, err := checkExpression(*fixture.Signature.Result, fmt.Sprintf("c_%d_result", caseIndex), resultSeed(caseIndex))
			if err != nil {
				return "", fmt.Errorf("check Windows ABI case %d result: %w", caseIndex, err)
			}
			fmt.Fprintf(&output, "    if (!(%s)) { return %d; }\n", check, caseIndex+1)
		}
	}
	output.WriteString("    return 0;\n}\n")
	return output.String(), nil
}

func emitCallee(common string, cases []emittedCase) (string, error) {
	var output strings.Builder
	output.WriteString(common)
	for caseIndex, fixture := range cases {
		fmt.Fprintf(&output, "%s fy_windows_abi_target_%d(%s) {\n", fixture.ResultType, caseIndex, parameterList(fixture.ParameterTypes))
		for parameterIndex, parameter := range fixture.Signature.Parameters {
			check, err := checkExpression(parameter, fmt.Sprintf("p_%d", parameterIndex), seed(caseIndex, parameterIndex))
			if err != nil {
				return "", fmt.Errorf("check Windows ABI case %d parameter %d: %w", caseIndex, parameterIndex, err)
			}
			fmt.Fprintf(&output, "    if (!(%s)) { fy_windows_abi_failure_%d = 1; }\n", check, caseIndex)
		}
		if fixture.Signature.Result == nil {
			output.WriteString("    return;\n}\n\n")
			continue
		}
		fmt.Fprintf(&output, "    %s result = {0};\n", fixture.ResultType)
		if err := emitInitialization(&output, *fixture.Signature.Result, "result", resultSeed(caseIndex), "    "); err != nil {
			return "", fmt.Errorf("initialize Windows ABI case %d result: %w", caseIndex, err)
		}
		output.WriteString("    return result;\n}\n\n")
	}
	return output.String(), nil
}

func parameterList(types []string) string {
	if len(types) == 0 {
		return "void"
	}
	values := make([]string, len(types))
	for index, typeName := range types {
		values[index] = fmt.Sprintf("%s p_%d", typeName, index)
	}
	return strings.Join(values, ", ")
}

func argumentList(caseIndex, count int) string {
	values := make([]string, count)
	for index := range values {
		values[index] = fmt.Sprintf("c_%d_p_%d", caseIndex, index)
	}
	return strings.Join(values, ", ")
}

func emitInitialization(output *strings.Builder, value layout.Type, expression string, valueSeed uint64, indentation string) error {
	switch value.Kind {
	case layout.KindLanguageScalar, layout.KindCScalar, layout.KindCEnum:
		literal, err := scalarLiteral(value, valueSeed)
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "%s%s = %s;\n", indentation, expression, literal)
		return nil
	case layout.KindPointer:
		fmt.Fprintf(output, "%s%s = NULL;\n", indentation, expression)
		return nil
	case layout.KindFunctionPtr:
		fmt.Fprintf(output, "%s%s = fy_windows_abi_callback;\n", indentation, expression)
		return nil
	case layout.KindArray:
		if value.Element == nil || value.Length == 0 {
			return fmt.Errorf("Windows ABI initializer has an invalid array")
		}
		for index := uint64(0); index < value.Length; index++ {
			if err := emitInitialization(output, *value.Element, fmt.Sprintf("%s[UINT64_C(%d)]", expression, index), valueSeed+index+1, indentation); err != nil {
				return err
			}
		}
		return nil
	case layout.KindStruct:
		if len(value.Fields) == 0 {
			return fmt.Errorf("Windows ABI initializer has an empty struct")
		}
		for index, field := range value.Fields {
			if err := emitInitialization(output, field.Type, fmt.Sprintf("%s.f_%d", expression, index), valueSeed+uint64(index)+1, indentation); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("Windows ABI initializer does not support layout kind %q", value.Kind)
	}
}

func checkExpression(value layout.Type, expression string, valueSeed uint64) (string, error) {
	switch value.Kind {
	case layout.KindLanguageScalar, layout.KindCScalar, layout.KindCEnum:
		literal, err := scalarLiteral(value, valueSeed)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s == %s)", expression, literal), nil
	case layout.KindPointer:
		return fmt.Sprintf("(%s == NULL)", expression), nil
	case layout.KindFunctionPtr:
		return fmt.Sprintf("(%s == fy_windows_abi_callback)", expression), nil
	case layout.KindArray:
		if value.Element == nil || value.Length == 0 {
			return "", fmt.Errorf("Windows ABI checker has an invalid array")
		}
		checks := make([]string, 0, value.Length)
		for index := uint64(0); index < value.Length; index++ {
			check, err := checkExpression(*value.Element, fmt.Sprintf("%s[UINT64_C(%d)]", expression, index), valueSeed+index+1)
			if err != nil {
				return "", err
			}
			checks = append(checks, check)
		}
		return strings.Join(checks, " && "), nil
	case layout.KindStruct:
		if len(value.Fields) == 0 {
			return "", fmt.Errorf("Windows ABI checker has an empty struct")
		}
		checks := make([]string, 0, len(value.Fields))
		for index, field := range value.Fields {
			check, err := checkExpression(field.Type, fmt.Sprintf("%s.f_%d", expression, index), valueSeed+uint64(index)+1)
			if err != nil {
				return "", err
			}
			checks = append(checks, check)
		}
		return strings.Join(checks, " && "), nil
	default:
		return "", fmt.Errorf("Windows ABI checker does not support layout kind %q", value.Kind)
	}
}

func scalarLiteral(value layout.Type, valueSeed uint64) (string, error) {
	if value.Kind == layout.KindCEnum && (value.Name == "c.float" || value.Name == "c.double") {
		return "", fmt.Errorf("Windows ABI literal has invalid floating enum base %q", value.Name)
	}
	spelling := ""
	found := false
	switch value.Kind {
	case layout.KindLanguageScalar:
		spelling, found = msvccprobe.LanguageScalarSpelling(value.Name)
	case layout.KindCScalar, layout.KindCEnum:
		spelling, found = msvccprobe.CScalarSpelling(value.Name)
	}
	if !found {
		return "", fmt.Errorf("Windows ABI literal does not support %s %q", value.Kind, value.Name)
	}
	if value.Kind == layout.KindLanguageScalar && value.Name == "bool" || value.Kind == layout.KindCScalar && value.Name == "c.bool" {
		return fmt.Sprintf("((%s)UINT64_C(%d))", spelling, valueSeed&1), nil
	}
	if value.Kind == layout.KindLanguageScalar && (value.Name == "f32" || value.Name == "f64") ||
		value.Kind == layout.KindCScalar && (value.Name == "c.float" || value.Name == "c.double") {
		return fmt.Sprintf("((%s)%d.0)", spelling, valueSeed), nil
	}
	return fmt.Sprintf("((%s)UINT64_C(%d))", spelling, valueSeed), nil
}

func seed(caseIndex, parameterIndex int) uint64 {
	return uint64(caseIndex*17 + parameterIndex + 1)
}

func resultSeed(caseIndex int) uint64 {
	return uint64(caseIndex*17 + 13)
}

// CloneCases returns an isolated case-ID copy for callers that retain receipts.
func CloneCases(unit Unit) []string {
	return slices.Clone(unit.Cases)
}
