package cbackendnative

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/0disoft/fairyc/internal/cabi"
	"github.com/0disoft/fairyc/internal/cabifixture"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/target"
)

const abiResultSeed = uint64(29)

type abiProbeCase struct {
	ID           string
	CallerSource string
	CalleeSource string
}

func generateABIProbes(targetID target.ID, cases []cabifixture.NativeProbeCase) ([]abiProbeCase, error) {
	if len(cases) == 0 {
		return nil, fmt.Errorf("native C ABI probe has no pass cases")
	}
	spec, found := target.Lookup(targetID)
	if !found {
		return nil, fmt.Errorf("native C ABI probe target %q is unknown", targetID)
	}
	classifier, err := cabi.New(spec)
	if err != nil {
		return nil, err
	}
	result := make([]abiProbeCase, 0, len(cases))
	previous := ""
	for _, fixture := range cases {
		if fixture.ID == "" || previous != "" && previous >= fixture.ID {
			return nil, fmt.Errorf("native C ABI probe case IDs are empty, duplicate, or unsorted")
		}
		previous = fixture.ID
		actualPlan, err := classifier.Plan(fixture.Signature)
		if err != nil {
			return nil, fmt.Errorf("native C ABI probe case %q: %w", fixture.ID, err)
		}
		if !reflect.DeepEqual(actualPlan, fixture.Plan) {
			return nil, fmt.Errorf("native C ABI probe case %q drifted from its fixture plan", fixture.ID)
		}
		caller, callee, err := emitABISources(spec, fixture.Signature)
		if err != nil {
			return nil, fmt.Errorf("native C ABI probe case %q: %w", fixture.ID, err)
		}
		result = append(result, abiProbeCase{ID: fixture.ID, CallerSource: caller, CalleeSource: callee})
	}
	return result, nil
}

func emitABISources(spec target.Spec, signature cabi.Signature) (string, string, error) {
	layoutEngine, err := layout.New(spec)
	if err != nil {
		return "", "", err
	}
	emitter := layoutProbeEmitter{engine: layoutEngine}
	resultType := "void"
	if signature.Result != nil {
		resultType, err = emitter.emitType(*signature.Result)
		if err != nil {
			return "", "", err
		}
	}
	parameterTypes := make([]string, len(signature.Parameters))
	for index, parameter := range signature.Parameters {
		parameterTypes[index], err = emitter.emitType(parameter)
		if err != nil {
			return "", "", err
		}
	}
	var common strings.Builder
	common.WriteString("/* fairyc bidirectional native C ABI probe: numeric private types, ISO C11 */\n")
	common.WriteString("#include <stddef.h>\n#include <stdint.h>\n\n")
	common.WriteString(emitter.output.String())
	common.WriteString("extern int fy_abi_failure;\n")
	common.WriteString("void fy_abi_callback(void);\n")
	fmt.Fprintf(&common, "%s fy_abi_target(%s);\n\n", resultType, abiParameterList(parameterTypes))

	var caller strings.Builder
	caller.WriteString(common.String())
	caller.WriteString("int fy_abi_failure = 0;\n")
	caller.WriteString("void fy_abi_callback(void) {}\n\n")
	caller.WriteString("int main(void) {\n")
	for index, parameter := range signature.Parameters {
		fmt.Fprintf(&caller, "    %s p_%d = {0};\n", parameterTypes[index], index)
		if err := emitABIInitialization(&caller, parameter, fmt.Sprintf("p_%d", index), uint64(index+1), "    "); err != nil {
			return "", "", err
		}
	}
	arguments := abiArgumentList(len(signature.Parameters))
	if signature.Result == nil {
		fmt.Fprintf(&caller, "    fy_abi_target(%s);\n", arguments)
	} else {
		fmt.Fprintf(&caller, "    %s result = fy_abi_target(%s);\n", resultType, arguments)
	}
	caller.WriteString("    if (fy_abi_failure != 0) { return 1; }\n")
	if signature.Result != nil {
		check, err := abiCheckExpression(*signature.Result, "result", abiResultSeed)
		if err != nil {
			return "", "", err
		}
		fmt.Fprintf(&caller, "    if (!(%s)) { return 2; }\n", check)
	}
	caller.WriteString("    return 0;\n}\n")

	var callee strings.Builder
	callee.WriteString(common.String())
	fmt.Fprintf(&callee, "%s fy_abi_target(%s) {\n", resultType, abiParameterList(parameterTypes))
	for index, parameter := range signature.Parameters {
		check, err := abiCheckExpression(parameter, fmt.Sprintf("p_%d", index), uint64(index+1))
		if err != nil {
			return "", "", err
		}
		fmt.Fprintf(&callee, "    if (!(%s)) { fy_abi_failure = 1; }\n", check)
	}
	if signature.Result == nil {
		callee.WriteString("    return;\n}\n")
	} else {
		fmt.Fprintf(&callee, "    %s result = {0};\n", resultType)
		if err := emitABIInitialization(&callee, *signature.Result, "result", abiResultSeed, "    "); err != nil {
			return "", "", err
		}
		callee.WriteString("    return result;\n}\n")
	}
	return caller.String(), callee.String(), nil
}

func abiParameterList(types []string) string {
	if len(types) == 0 {
		return "void"
	}
	values := make([]string, len(types))
	for index, typeName := range types {
		values[index] = fmt.Sprintf("%s p_%d", typeName, index)
	}
	return strings.Join(values, ", ")
}

func abiArgumentList(count int) string {
	values := make([]string, count)
	for index := range values {
		values[index] = fmt.Sprintf("p_%d", index)
	}
	return strings.Join(values, ", ")
}

func emitABIInitialization(output *strings.Builder, value layout.Type, expression string, seed uint64, indentation string) error {
	switch value.Kind {
	case layout.KindLanguageScalar, layout.KindCScalar, layout.KindCEnum:
		literal, err := abiLiteral(value, seed)
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "%s%s = %s;\n", indentation, expression, literal)
		return nil
	case layout.KindPointer:
		fmt.Fprintf(output, "%s%s = NULL;\n", indentation, expression)
		return nil
	case layout.KindFunctionPtr:
		fmt.Fprintf(output, "%s%s = fy_abi_callback;\n", indentation, expression)
		return nil
	case layout.KindArray:
		if value.Element == nil || value.Length == 0 {
			return fmt.Errorf("C ABI initializer has an invalid array")
		}
		for index := uint64(0); index < value.Length; index++ {
			if err := emitABIInitialization(output, *value.Element, fmt.Sprintf("%s[UINT64_C(%d)]", expression, index), seed+index+1, indentation); err != nil {
				return err
			}
		}
		return nil
	case layout.KindStruct:
		if len(value.Fields) == 0 {
			return fmt.Errorf("C ABI initializer has an empty struct")
		}
		for index, field := range value.Fields {
			if err := emitABIInitialization(output, field.Type, fmt.Sprintf("%s.f_%d", expression, index), seed+uint64(index)+1, indentation); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("C ABI initializer does not support layout kind %q", value.Kind)
	}
}

func abiCheckExpression(value layout.Type, expression string, seed uint64) (string, error) {
	switch value.Kind {
	case layout.KindLanguageScalar, layout.KindCScalar, layout.KindCEnum:
		literal, err := abiLiteral(value, seed)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s == %s)", expression, literal), nil
	case layout.KindPointer:
		return fmt.Sprintf("(%s == NULL)", expression), nil
	case layout.KindFunctionPtr:
		return fmt.Sprintf("(%s == fy_abi_callback)", expression), nil
	case layout.KindArray:
		if value.Element == nil || value.Length == 0 {
			return "", fmt.Errorf("C ABI checker has an invalid array")
		}
		checks := make([]string, 0, value.Length)
		for index := uint64(0); index < value.Length; index++ {
			check, err := abiCheckExpression(*value.Element, fmt.Sprintf("%s[UINT64_C(%d)]", expression, index), seed+index+1)
			if err != nil {
				return "", err
			}
			checks = append(checks, check)
		}
		return strings.Join(checks, " && "), nil
	case layout.KindStruct:
		if len(value.Fields) == 0 {
			return "", fmt.Errorf("C ABI checker has an empty struct")
		}
		checks := make([]string, 0, len(value.Fields))
		for index, field := range value.Fields {
			check, err := abiCheckExpression(field.Type, fmt.Sprintf("%s.f_%d", expression, index), seed+uint64(index)+1)
			if err != nil {
				return "", err
			}
			checks = append(checks, check)
		}
		return strings.Join(checks, " && "), nil
	default:
		return "", fmt.Errorf("C ABI checker does not support layout kind %q", value.Kind)
	}
}

func abiLiteral(value layout.Type, seed uint64) (string, error) {
	if value.Kind == layout.KindCEnum && (value.Name == "c.float" || value.Name == "c.double") {
		return "", fmt.Errorf("C ABI literal has invalid floating enum base %q", value.Name)
	}
	spelling := ""
	var found bool
	switch value.Kind {
	case layout.KindLanguageScalar:
		spelling, found = languageScalarSpelling(value.Name)
	case layout.KindCScalar, layout.KindCEnum:
		spelling, found = cScalarSpelling(value.Name)
	}
	if !found {
		return "", fmt.Errorf("C ABI literal does not support %s %q", value.Kind, value.Name)
	}
	if value.Kind == layout.KindLanguageScalar && value.Name == "bool" || value.Kind == layout.KindCScalar && value.Name == "c.bool" {
		return fmt.Sprintf("((%s)UINT64_C(%d))", spelling, seed&1), nil
	}
	if value.Kind == layout.KindLanguageScalar && (value.Name == "f32" || value.Name == "f64") ||
		value.Kind == layout.KindCScalar && (value.Name == "c.float" || value.Name == "c.double") {
		return fmt.Sprintf("((%s)%d.0)", spelling, seed), nil
	}
	return fmt.Sprintf("((%s)UINT64_C(%d))", spelling, seed), nil
}
