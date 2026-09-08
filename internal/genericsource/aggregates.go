package genericsource

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/syntax"
)

type aggregateTemplate struct {
	structure *ast.Struct
	resource  *ast.Resource
}

func (a aggregateTemplate) shape() ast.Struct {
	if a.structure != nil {
		return *a.structure
	}
	r := a.resource
	return ast.Struct{Name: r.Name, Public: r.Public, Span: r.Span, NameSpan: r.NameSpan,
		TypeParameters: r.TypeParameters, Fields: r.Fields}
}

func (x *expander) sourceAggregates() []aggregateTemplate {
	var result []aggregateTemplate
	for i := range x.module.Structs {
		result = append(result, aggregateTemplate{structure: &x.module.Structs[i]})
	}
	for i := range x.module.Resources {
		result = append(result, aggregateTemplate{resource: &x.module.Resources[i]})
	}
	return result
}

func (x *expander) collectAggregateTemplates() error {
	x.aggregates = map[string]aggregateTemplate{}
	counts := map[string]int{}
	for _, a := range x.sourceAggregates() {
		counts[a.shape().Name]++
	}
	for _, v := range x.module.Constants {
		counts[v.Name]++
	}
	for _, v := range x.module.Enums {
		counts[v.Name]++
	}
	for _, v := range x.module.ForeignFunctions {
		counts[v.Name]++
	}
	for _, v := range x.module.Imports {
		counts[v.Alias]++
	}
	for _, aggregate := range x.sourceAggregates() {
		shape := aggregate.shape()
		if len(shape.TypeParameters) == 0 {
			continue
		}
		if x.scalarTemplatesOnly {
			return x.at(shape.NameSpan, FailureDeclaration, fmt.Errorf("generic aggregate declarations are not admitted outside the embedded SDK"))
		}
		if slices.Contains(scalarTypes, shape.Name) || slices.Contains([]string{"void", "never", "str", "Region", "Option", "Result", "source_location"}, shape.Name) {
			return x.at(shape.NameSpan, FailureDeclaration, fmt.Errorf("aggregate name %q shadows a builtin", shape.Name))
		}
		if !x.allowPublic && shape.Public {
			return x.at(shape.Span, FailureDeclaration, fmt.Errorf("generic aggregates require the private module-local admission path"))
		}
		if counts[shape.Name] != 1 {
			return x.at(shape.NameSpan, FailureDeclaration, fmt.Errorf("duplicate generic aggregate %q", shape.Name))
		}
		for _, parameter := range shape.TypeParameters {
			if slices.Contains(scalarTypes, parameter) || slices.Contains([]string{"void", "never", "str", "Region", "Option", "Result"}, parameter) {
				return x.at(shape.Span, FailureDeclaration, fmt.Errorf("aggregate parameter %q shadows a builtin", parameter))
			}
		}
		x.aggregates[shape.Name] = aggregate
	}
	return nil
}

func (x *expander) registerAggregateType(ref ast.TypeReference, parameters []string, declaration *genericplan.Declaration, requests *[]genericplan.Request) error {
	if ref.Kind != ast.TypeKindNamed {
		return nil
	}
	if strings.Contains(ref.Path, generatedPrefix) {
		return x.at(ref.Span, FailureRequest, fmt.Errorf("source cannot name generated aggregate types"))
	}
	target, found, err := x.aggregateTarget(ref)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	aggregate := target.module.aggregates[target.name]
	if len(ref.Arguments) != len(aggregate.shape().TypeParameters) {
		return x.at(ref.Span, FailureRequest, &genericplan.Error{Code: genericplan.CodeTypeArgumentArity, Message: "aggregate type argument count does not match the declaration"})
	}
	dependency := genericplan.Dependency{Declaration: target.id()}
	request := genericplan.Request{Declaration: dependency.Declaration}
	for _, argument := range ref.Arguments {
		if argument.Kind != ast.TypeKindNamed || len(argument.Arguments) != 0 {
			return x.at(argument.Span, FailureRequest, fmt.Errorf("aggregate argument requires a scalar or direct type parameter"))
		}
		if slices.Contains(parameters, argument.Path) {
			dependency.Arguments = append(dependency.Arguments, genericplan.ParameterType(argument.Path))
		} else {
			typeID, found := x.concreteType(argument.Path)
			if !found {
				return x.at(argument.Span, FailureRequest, fmt.Errorf("unsupported aggregate argument %q", argument.Path))
			}
			dependency.Arguments = append(dependency.Arguments, genericplan.ConcreteType(typeID))
			request.Arguments = append(request.Arguments, typeID)
		}
	}
	if len(parameters) != 0 {
		if !slices.ContainsFunc(declaration.Dependencies, func(v genericplan.Dependency) bool {
			return v.Declaration == dependency.Declaration && slices.Equal(v.Arguments, dependency.Arguments)
		}) {
			declaration.Dependencies = append(declaration.Dependencies, dependency)
		}
	} else {
		*requests = append(*requests, request)
		if x.requests == nil {
			x.requests = map[string]diagnostic.Span{}
		}
		key := requestKey(request.Declaration, request.Arguments)
		if previous, found := x.requests[key]; !found || ref.Span.StartByte < previous.StartByte {
			x.requests[key] = ref.Span
		}
	}
	return nil
}

func aggregateRecordType(e *ast.Expression) (ast.TypeReference, error) {
	if len(e.Children) == 0 || e.Children[0] == nil || e.Children[0].Kind != syntax.KindPath {
		return ast.TypeReference{}, fmt.Errorf("generic aggregate construction requires a direct type path")
	}
	ref := ast.NamedTypeReference(e.Children[0].Text, e.Span)
	ref.Arguments = e.TypeArguments
	return ref, nil
}

func (x *expander) registerAggregateRecord(e *ast.Expression, parameters []string, declaration *genericplan.Declaration, requests *[]genericplan.Request) error {
	ref, err := aggregateRecordType(e)
	if err != nil {
		return err
	}
	if _, found, err := x.aggregateTarget(ref); err != nil {
		return err
	} else if !found {
		return fmt.Errorf("unknown generic aggregate %q", ref.Path)
	}
	return x.registerAggregateType(ref, parameters, declaration, requests)
}

func (x *expander) declareAggregates(builder *genericplan.Builder) ([]genericplan.Request, error) {
	var requests []genericplan.Request
	for _, aggregate := range x.sourceAggregates() {
		shape := aggregate.shape()
		declaration := genericplan.Declaration{ID: x.id(shape.Name), Owner: x.owner}
		for _, parameter := range shape.TypeParameters {
			declaration.Parameters = append(declaration.Parameters, genericplan.TypeParameter{Name: parameter})
		}
		var typeError error
		t := transform{typeReference: func(ref *ast.TypeReference) {
			if typeError == nil {
				typeError = x.registerAggregateType(*ref, shape.TypeParameters, &declaration, &requests)
			}
		}, binding: func(name string, at diagnostic.Span) error {
			return x.at(at, FailureDeclaration, x.checkBinding(name))
		}, expression: func(e *ast.Expression) (err error) {
			defer func() { err = x.at(e.Span, FailureRequest, err) }()
			if err := x.checkExpression(e); err != nil {
				return err
			}
			if e.Kind == syntax.KindRecord && len(e.TypeArguments) != 0 {
				return x.registerAggregateRecord(e, shape.TypeParameters, &declaration, &requests)
			}
			return x.registerFunctionCall(e, shape.TypeParameters, &declaration, &requests)
		}}
		for _, field := range shape.Fields {
			t.typ(field.Type)
		}
		if aggregate.resource != nil && aggregate.resource.Cleanup != nil {
			if _, err := t.block(aggregate.resource.Cleanup.Body); err != nil {
				return nil, x.at(shape.Span, FailureDeclaration, err)
			}
		}
		if typeError != nil {
			return nil, typeError
		}
		if len(shape.TypeParameters) == 0 {
			continue
		}
		encoded, err := json.Marshal(struct {
			Struct   *ast.Struct
			Resource *ast.Resource
		}{aggregate.structure, aggregate.resource})
		if err != nil {
			return nil, err
		}
		declaration.Definition = genericplan.DigestText("source-aggregate-ast-v0", string(encoded))
		if x.allowPublic {
			var imports []string
			for alias, target := range x.imports {
				imports = append(imports, alias+"="+string(target.owner)+"::"+target.module.Path)
			}
			slices.Sort(imports)
			bindings, err := json.Marshal(imports)
			if err != nil {
				return nil, err
			}
			declaration.Definition = genericplan.DigestText("source-package-aggregate-v0", string(encoded)+"\n"+string(bindings))
		}
		if err := builder.AddDeclaration(declaration); err != nil {
			return nil, x.at(shape.Span, FailureDeclaration, err)
		}
	}
	return requests, nil
}

func (x *expander) rewriteAggregateType(ref *ast.TypeReference, names map[string]string) error {
	if ref.Kind != ast.TypeKindNamed {
		return nil
	}
	target, found, err := x.aggregateTarget(*ref)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	var arguments []genericplan.TypeID
	for _, argument := range ref.Arguments {
		typeID, found := x.concreteReference(argument)
		if !found {
			return fmt.Errorf("unregistered aggregate argument %q", argument.Path)
		}
		arguments = append(arguments, typeID)
	}
	name, found := names[requestKey(target.id(), arguments)]
	if !found {
		return fmt.Errorf("aggregate specialization missing from verified plan")
	}
	*ref = ast.NamedTypeReference(target.prefix+name, ref.Span)
	return nil
}

func (x *expander) rewriteAggregateRecord(e *ast.Expression, names map[string]string) error {
	ref, err := aggregateRecordType(e)
	if err != nil {
		return err
	}
	if _, found, err := x.aggregateTarget(ref); err != nil {
		return err
	} else if !found {
		return fmt.Errorf("unknown generic aggregate construction")
	}
	if err := x.rewriteAggregateType(&ref, names); err != nil {
		return err
	}
	e.Text = ref.Path
	e.Children[0].Text = ref.Path
	e.TypeArguments = nil
	return nil
}

func (x *expander) emitAggregates(result *ast.Module, plan genericplan.Plan, names map[string]string) error {
	result.Structs, result.Resources = nil, nil
	emit := func(aggregate aggregateTemplate, name string, bindings map[string]ast.TypeReference) error {
		shape := aggregate.shape()
		var typeError error
		t := transform{bindings: bindings, typeReference: func(ref *ast.TypeReference) {
			if typeError == nil {
				typeError = x.rewriteAggregateType(ref, names)
			}
		}, expression: func(e *ast.Expression) error {
			return x.rewriteSpecialization(e, names)
		}}
		fields := slices.Clone(shape.Fields)
		for i := range fields {
			fields[i].Type = t.typ(fields[i].Type)
		}
		if aggregate.structure != nil {
			value := *aggregate.structure
			value.Name, value.Fields, value.TypeParameters = name, fields, nil
			result.Structs = append(result.Structs, value)
		} else {
			value := *aggregate.resource
			value.Name, value.Fields, value.TypeParameters = name, fields, nil
			if value.Cleanup != nil {
				cleanup := *value.Cleanup
				var err error
				cleanup.Body, err = t.block(cleanup.Body)
				if err != nil {
					return err
				}
				value.Cleanup = &cleanup
			}
			result.Resources = append(result.Resources, value)
		}
		return typeError
	}
	for _, aggregate := range x.sourceAggregates() {
		if shape := aggregate.shape(); len(shape.TypeParameters) == 0 {
			if err := emit(aggregate, shape.Name, nil); err != nil {
				return err
			}
		}
	}
	for _, specialization := range plan.Specializations {
		name := strings.TrimPrefix(string(specialization.Declaration), string(x.id("")))
		aggregate, found := x.aggregates[name]
		if !found {
			continue
		}
		bindings := map[string]ast.TypeReference{}
		for _, binding := range specialization.Bindings {
			bindings[binding.Parameter] = x.boundType(binding.Type, aggregate.shape().Span)
		}
		if err := emit(aggregate, generatedPrefix+string(specialization.ID), bindings); err != nil {
			return err
		}
	}
	return nil
}
