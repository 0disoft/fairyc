package genericsource

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/syntax"
)

func (x *expander) registerFunctionCall(e *ast.Expression, parameters []string, declaration *genericplan.Declaration, requests *[]genericplan.Request) error {
	if e.Kind != syntax.KindSpecialize {
		return nil
	}
	target, err := x.target(e)
	if err != nil {
		return err
	}
	if len(e.TypeArguments) != len(target.module.templates[target.name].TypeParameters) {
		return &genericplan.Error{Code: genericplan.CodeTypeArgumentArity, Message: "generic type argument count does not match the declaration"}
	}
	dependency := genericplan.Dependency{Declaration: target.id()}
	request := genericplan.Request{Declaration: target.id()}
	for _, argument := range e.TypeArguments {
		if argument.Kind != ast.TypeKindNamed || len(argument.Arguments) != 0 {
			return fmt.Errorf("generic argument must be a scalar or direct type parameter")
		}
		if slices.Contains(parameters, argument.Path) {
			dependency.Arguments = append(dependency.Arguments, genericplan.ParameterType(argument.Path))
		} else {
			if !slices.Contains(scalarTypes, argument.Path) {
				return fmt.Errorf("unsupported concrete generic type %q", argument.Path)
			}
			dependency.Arguments = append(dependency.Arguments, genericplan.ConcreteType(genericplan.TypeID(argument.Path)))
			request.Arguments = append(request.Arguments, genericplan.TypeID(argument.Path))
		}
	}
	if len(parameters) != 0 {
		if !slices.ContainsFunc(declaration.Dependencies, func(existing genericplan.Dependency) bool {
			return existing.Declaration == dependency.Declaration && slices.Equal(existing.Arguments, dependency.Arguments)
		}) {
			declaration.Dependencies = append(declaration.Dependencies, dependency)
		}
	} else {
		*requests = append(*requests, request)
		if x.requests == nil {
			x.requests = map[string]diagnostic.Span{}
		}
		key := requestKey(request.Declaration, request.Arguments)
		if previous, found := x.requests[key]; !found || e.Span.StartByte < previous.StartByte {
			x.requests[key] = e.Span
		}
	}
	return nil
}

func (x *expander) rewriteSpecialization(e *ast.Expression, names map[string]string) error {
	if e.Kind == syntax.KindRecord && len(e.TypeArguments) != 0 {
		return x.rewriteAggregateRecord(e, names)
	}
	if e.Kind != syntax.KindSpecialize {
		return nil
	}
	target, err := x.target(e)
	if err != nil {
		return err
	}
	arguments := make([]genericplan.TypeID, len(e.TypeArguments))
	for i, argument := range e.TypeArguments {
		arguments[i] = genericplan.TypeID(argument.Path)
	}
	generated, ok := names[requestKey(target.id(), arguments)]
	if !ok {
		return fmt.Errorf("specialization missing from verified plan")
	}
	*e = ast.Expression{Kind: syntax.KindPath, Span: e.Span, Text: target.prefix + generated}
	return nil
}
