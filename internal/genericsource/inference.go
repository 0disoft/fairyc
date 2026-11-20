package genericsource

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/syntax"
)

type inferenceScope struct {
	module     *expander
	parameters []string
	parent     *inferenceScope
	bindings   map[string]*ast.TypeReference
}

func (s *inferenceScope) child() *inferenceScope {
	return &inferenceScope{module: s.module, parameters: s.parameters, parent: s}
}

func (s *inferenceScope) bind(name string, typ *ast.TypeReference) {
	if s.bindings == nil {
		s.bindings = make(map[string]*ast.TypeReference)
	}
	s.bindings[name] = typ
}

func (s *inferenceScope) binding(name string) *ast.TypeReference {
	for current := s; current != nil; current = current.parent {
		if typ, found := current.bindings[name]; found {
			return typ
		}
	}
	return nil
}

func ungroupInferenceType(ref *ast.TypeReference) *ast.TypeReference {
	for ref != nil && ref.Kind == ast.TypeKindGrouped {
		ref = ref.Element
	}
	return ref
}

func (s *inferenceScope) scalar(ref *ast.TypeReference) string {
	ref = ungroupInferenceType(ref)
	if ref != nil && ref.Kind == ast.TypeKindNamed && len(ref.Arguments) == 0 &&
		(slices.Contains(scalarTypes, ref.Path) || slices.Contains(s.parameters, ref.Path)) {
		return ref.Path
	}
	return ""
}

func inferenceIntegerLiteral(e *ast.Expression) bool {
	if e == nil || e.Kind != syntax.KindLiteral || e.Text == "" || e.Text[0] < '0' || e.Text[0] > '9' {
		return false
	}
	raw := e.Text
	base := strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") ||
		strings.HasPrefix(raw, "0b") || strings.HasPrefix(raw, "0B")
	return base || !strings.ContainsAny(raw, ".eE")
}

// evidence returns a known source type or an integer literal that can take a
// type from another occurrence of the same binder.
func (s *inferenceScope) evidence(e *ast.Expression) (typ *ast.TypeReference, integer bool) {
	if e == nil {
		return nil, false
	}
	switch e.Kind {
	case syntax.KindPath:
		return s.binding(e.Text), false
	case syntax.KindCast:
		return e.Type, false
	case syntax.KindLiteral:
		if e.Text == "true" || e.Text == "false" {
			typ := ast.NamedTypeReference("bool", e.Span)
			return &typ, false
		}
		return nil, inferenceIntegerLiteral(e)
	case syntax.KindPrefix:
		return nil, e.Text == "-" && len(e.Children) == 1 && inferenceIntegerLiteral(e.Children[0])
	case syntax.KindCall:
		return s.genericResult(e), false
	}
	return nil, false
}

func (s *inferenceScope) genericResult(call *ast.Expression) *ast.TypeReference {
	if len(call.Children) == 0 {
		return nil
	}
	callee := call.Children[0]
	arguments := s.module.inferred[call]
	if callee.Kind == syntax.KindSpecialize && len(callee.Children) == 1 {
		arguments, callee = callee.TypeArguments, callee.Children[0]
	}
	if callee.Kind != syntax.KindPath {
		return nil
	}
	target, found := s.module.lookup(callee.Text)
	if !found || (target.module != s.module && !target.module.templates[target.name].Public) {
		return nil
	}
	fn := target.module.templates[target.name]
	if len(arguments) != len(fn.TypeParameters) {
		return nil
	}
	return specializeInferenceResult(&fn.ResultType, fn.TypeParameters, arguments)
}

func specializeInferenceResult(ref *ast.TypeReference, parameters []string, arguments []ast.TypeReference) *ast.TypeReference {
	ref = ungroupInferenceType(ref)
	if ref == nil || ref.Kind != ast.TypeKindNamed {
		return nil
	}
	if len(ref.Arguments) == 0 {
		if index := slices.Index(parameters, ref.Path); index >= 0 {
			return &arguments[index]
		}
		if slices.Contains(scalarTypes, ref.Path) {
			return ref
		}
		return nil
	}
	if !inferenceContainer(ref) {
		return nil
	}
	result := *ref
	result.Arguments = make([]ast.TypeReference, len(ref.Arguments))
	for i := range ref.Arguments {
		argument := specializeInferenceResult(&ref.Arguments[i], parameters, arguments)
		if argument == nil {
			return nil
		}
		result.Arguments[i] = *argument
	}
	return &result
}

func inferenceContainer(ref *ast.TypeReference) bool {
	return ref != nil && ref.Kind == ast.TypeKindNamed &&
		(ref.Path == "Option" && len(ref.Arguments) == 1 || ref.Path == "Result" && len(ref.Arguments) == 2)
}

func inferenceConstructorPayload(call *ast.Expression, expected *ast.TypeReference) *ast.TypeReference {
	if call.Kind != syntax.KindCall || len(call.Children) != 2 || call.Children[0].Kind != syntax.KindPath {
		return nil
	}
	container := ungroupInferenceType(expected)
	if !inferenceContainer(container) {
		return nil
	}
	switch call.Children[0].Text {
	case ".some":
		if container.Path == "Option" {
			return &container.Arguments[0]
		}
	case ".ok":
		if container.Path == "Result" {
			return &container.Arguments[0]
		}
	case ".err":
		if container.Path == "Result" {
			return &container.Arguments[1]
		}
	}
	return nil
}

// Forward a written result constraint only to expressions that produce it.
func inferenceChildExpected(parent *ast.Expression, index int, expected *ast.TypeReference) *ast.TypeReference {
	if expected == nil {
		return nil
	}
	switch parent.Kind {
	case syntax.KindGroup, syntax.KindUnsafe:
		if len(parent.Children) == 1 {
			return expected
		}
	case syntax.KindIf, syntax.KindSwitch:
		if index > 0 {
			return expected
		}
	case syntax.KindPattern:
		if parent.Text == "arm" && len(parent.Children) == 2 && index == 1 {
			return expected
		}
	case syntax.KindCall:
		if index == 1 {
			return inferenceConstructorPayload(parent, expected)
		}
	}
	return nil
}

type inferenceConstraint struct {
	typ     string
	integer bool
}

func (s *inferenceScope) constrainArgument(formal *ast.TypeReference, argument *ast.Expression, parameters []string, bindings map[string]inferenceConstraint) error {
	for argument != nil {
		if argument.Kind == syntax.KindGroup && len(argument.Children) == 1 {
			argument = argument.Children[0]
			continue
		}
		payload := inferenceConstructorPayload(argument, formal)
		if payload == nil {
			break
		}
		formal, argument = payload, argument.Children[1]
	}
	typ, integer := s.evidence(argument)
	return s.constrain(formal, typ, integer, parameters, bindings)
}

func (s *inferenceScope) constrain(formal, actual *ast.TypeReference, integer bool, parameters []string, bindings map[string]inferenceConstraint) error {
	formal, actual = ungroupInferenceType(formal), ungroupInferenceType(actual)
	if formal == nil || formal.Kind != ast.TypeKindNamed {
		return nil
	}
	if len(formal.Arguments) == 0 && slices.Contains(parameters, formal.Path) {
		typ := s.scalar(actual)
		previous := bindings[formal.Path]
		if typ != "" {
			if previous.typ != "" && previous.typ != typ {
				return fmt.Errorf("conflicting types for parameter %s: %s and %s", formal.Path, previous.typ, typ)
			}
			previous.typ = typ
		}
		previous.integer = previous.integer || integer
		bindings[formal.Path] = previous
		return nil
	}
	if !inferenceContainer(formal) || !inferenceContainer(actual) || formal.Path != actual.Path {
		return nil
	}
	for i := range formal.Arguments {
		if err := s.constrain(&formal.Arguments[i], &actual.Arguments[i], false, parameters, bindings); err != nil {
			return err
		}
	}
	return nil
}

func (s *inferenceScope) infer(original, call *ast.Expression, expected *ast.TypeReference) error {
	if call.Kind != syntax.KindCall || len(call.Children) == 0 || call.Children[0].Kind != syntax.KindPath {
		return nil
	}
	callee := call.Children[0]
	target, found := s.module.lookup(callee.Text)
	if !found {
		return nil
	}
	selector := ast.Expression{Kind: syntax.KindSpecialize, Span: callee.Span, Children: []*ast.Expression{callee}}
	if _, err := s.module.target(&selector); err != nil {
		return err
	}
	fn := target.module.templates[target.name]
	if len(call.Children)-1 != len(fn.Parameters) {
		return fmt.Errorf("cannot infer type arguments for %q with a different value argument count", callee.Text)
	}
	bindings := make(map[string]inferenceConstraint, len(fn.TypeParameters))
	for index := range fn.Parameters {
		if err := s.constrainArgument(&fn.Parameters[index].Type, call.Children[index+1], fn.TypeParameters, bindings); err != nil {
			return fmt.Errorf("cannot infer type arguments for %q: %w", callee.Text, err)
		}
	}
	if err := s.constrain(&fn.ResultType, expected, false, fn.TypeParameters, bindings); err != nil {
		return fmt.Errorf("cannot infer type arguments for %q: %w", callee.Text, err)
	}
	selector.TypeArguments = make([]ast.TypeReference, 0, len(fn.TypeParameters))
	for _, name := range fn.TypeParameters {
		binding := bindings[name]
		if binding.typ == "" && binding.integer {
			binding.typ = "i32"
		}
		if binding.typ == "" {
			return fmt.Errorf("cannot infer type argument %s for %q; supply explicit type arguments", name, callee.Text)
		}
		selector.TypeArguments = append(selector.TypeArguments, ast.NamedTypeReference(binding.typ, callee.Span))
	}
	if s.module.inferred == nil {
		s.module.inferred = make(map[*ast.Expression][]ast.TypeReference)
	}
	s.module.inferred[original] = selector.TypeArguments
	call.Children = slices.Clone(call.Children)
	call.Children[0] = &selector
	return nil
}
