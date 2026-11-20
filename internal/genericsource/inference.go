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
	bindings   map[string]string
}

func (s *inferenceScope) child() *inferenceScope {
	return &inferenceScope{module: s.module, parameters: s.parameters, parent: s}
}

func (s *inferenceScope) bind(name, typ string) {
	if s.bindings == nil {
		s.bindings = make(map[string]string)
	}
	s.bindings[name] = typ
}

func (s *inferenceScope) binding(name string) string {
	for current := s; current != nil; current = current.parent {
		if typ, found := current.bindings[name]; found {
			return typ
		}
	}
	return ""
}

func (s *inferenceScope) scalar(ref *ast.TypeReference) string {
	for ref != nil && ref.Kind == ast.TypeKindGrouped {
		ref = ref.Element
	}
	if ref != nil && ref.Kind == ast.TypeKindNamed && len(ref.Arguments) == 0 &&
		(slices.Contains(scalarTypes, ref.Path) || slices.Contains(s.parameters, ref.Path)) {
		return ref.Path
	}
	return ""
}

// evidence returns a fixed scalar type or an integer literal that can take a
// type from another occurrence of the same binder.
func (s *inferenceScope) evidence(e *ast.Expression) (typ string, integer bool) {
	for e != nil && e.Kind == syntax.KindGroup && len(e.Children) == 1 {
		e = e.Children[0]
	}
	if e == nil {
		return "", false
	}
	switch e.Kind {
	case syntax.KindPath:
		return s.binding(e.Text), false
	case syntax.KindCast:
		return s.scalar(e.Type), false
	case syntax.KindLiteral:
		if e.Text == "true" || e.Text == "false" {
			return "bool", false
		}
		raw := e.Text
		if raw != "" && raw[0] >= '0' && raw[0] <= '9' {
			base := strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") ||
				strings.HasPrefix(raw, "0b") || strings.HasPrefix(raw, "0B")
			return "", base || !strings.ContainsAny(raw, ".eE")
		}
	case syntax.KindCall:
		return s.genericResult(e), false
	}
	return "", false
}

func (s *inferenceScope) genericResult(call *ast.Expression) string {
	if len(call.Children) == 0 {
		return ""
	}
	callee := call.Children[0]
	arguments := s.module.inferred[call]
	if callee.Kind == syntax.KindSpecialize && len(callee.Children) == 1 {
		arguments, callee = callee.TypeArguments, callee.Children[0]
	}
	if callee.Kind != syntax.KindPath {
		return ""
	}
	target, found := s.module.lookup(callee.Text)
	if !found || (target.module != s.module && !target.module.templates[target.name].Public) {
		return ""
	}
	fn := target.module.templates[target.name]
	if fn.ResultType.Kind != ast.TypeKindNamed || len(fn.ResultType.Arguments) != 0 {
		return ""
	}
	if index := slices.Index(fn.TypeParameters, fn.ResultType.Path); index >= 0 {
		if index < len(arguments) {
			return s.scalar(&arguments[index])
		}
		return ""
	}
	if slices.Contains(scalarTypes, fn.ResultType.Path) {
		return fn.ResultType.Path
	}
	return ""
}

func (s *inferenceScope) infer(original, call *ast.Expression) error {
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
	type constraint struct {
		typ     string
		integer bool
	}
	bindings := make(map[string]constraint, len(fn.TypeParameters))
	for index, formal := range fn.Parameters {
		ref := formal.Type
		if ref.Kind != ast.TypeKindNamed || len(ref.Arguments) != 0 || !slices.Contains(fn.TypeParameters, ref.Path) {
			continue
		}
		typ, integer := s.evidence(call.Children[index+1])
		previous := bindings[ref.Path]
		if typ != "" {
			if previous.typ != "" && previous.typ != typ {
				return fmt.Errorf("conflicting type arguments for %q parameter %s: %s and %s", callee.Text, ref.Path, previous.typ, typ)
			}
			previous.typ = typ
		}
		previous.integer = previous.integer || integer
		bindings[ref.Path] = previous
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
