package genericsource

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/syntax"
)

const generatedPrefix = "_fy_generic_"

var scalarTypes = []string{"bool", "i8", "i16", "i32", "i64", "isize", "u8", "u16", "u32", "u64", "usize", "f32", "f64"}

// Expand prepares private module-local scalar instantiations. This is an internal
// admission step, not a stable package-interface or cache transport. The caller
// must run ordinary semantic analysis and MIR verification on the returned AST.
// The explicit owner is part of every planned declaration identity.
func Expand(module *ast.Module, owner genericplan.PackageID, limits genericplan.PlanLimits) (*ast.Module, genericplan.Plan, error) {
	if module == nil {
		return nil, genericplan.Plan{}, fmt.Errorf("generic source module is nil")
	}
	x := expander{module: module, owner: owner, templates: map[string]ast.Function{}, names: map[string]bool{}}
	result, plan, err := x.expand(limits)
	if err != nil {
		return nil, genericplan.Plan{}, err
	}
	return result, plan, nil
}

type expander struct {
	module    *ast.Module
	owner     genericplan.PackageID
	templates map[string]ast.Function
	names     map[string]bool
}

func (x *expander) id(name string) genericplan.DeclarationID {
	return genericplan.DeclarationID(string(x.owner) + "::" + x.module.Path + "::" + name)
}

func (x *expander) expand(limits genericplan.PlanLimits) (*ast.Module, genericplan.Plan, error) {
	b, err := genericplan.NewBuilder(genericplan.DefaultCatalogLimits())
	if err != nil {
		return nil, genericplan.Plan{}, err
	}
	for _, name := range scalarTypes {
		if err := b.AddType(genericplan.TypeDescriptor{ID: genericplan.TypeID(name), Owner: "builtin", Abilities: []genericplan.Ability{genericplan.AbilityScalar}, Definition: genericplan.DigestText("source-scalar-v0", name)}); err != nil {
			return nil, genericplan.Plan{}, err
		}
	}
	// Template removal must not hide an original module-namespace collision.
	for _, v := range x.module.Constants {
		x.names[v.Name] = true
	}
	for _, v := range x.module.Structs {
		x.names[v.Name] = true
	}
	for _, v := range x.module.Resources {
		x.names[v.Name] = true
	}
	for _, v := range x.module.Enums {
		x.names[v.Name] = true
	}
	for _, v := range x.module.ForeignFunctions {
		x.names[v.Name] = true
	}
	for name := range x.names {
		if strings.HasPrefix(name, generatedPrefix) {
			return nil, genericplan.Plan{}, fmt.Errorf("reserved module name %q", name)
		}
	}
	for _, fn := range x.module.Functions {
		if x.names[fn.Name] || strings.HasPrefix(fn.Name, generatedPrefix) {
			return nil, genericplan.Plan{}, fmt.Errorf("duplicate or reserved function %q", fn.Name)
		}
		x.names[fn.Name] = true
		if len(fn.TypeParameters) == 0 {
			continue
		}
		if fn.Public || fn.Owner != "" || fn.Name == "source_location" {
			return nil, genericplan.Plan{}, fmt.Errorf("generic function %q requires private module-local ownership", fn.Name)
		}
		for _, parameter := range fn.TypeParameters {
			if slices.Contains(scalarTypes, parameter) || slices.Contains([]string{"void", "never", "str", "Region", "Option", "Result"}, parameter) {
				return nil, genericplan.Plan{}, fmt.Errorf("type parameter %q shadows a builtin", parameter)
			}
		}
		x.templates[fn.Name] = fn
	}
	var requests []genericplan.Request
	for _, fn := range x.module.Functions {
		declaration := genericplan.Declaration{ID: x.id(fn.Name), Owner: x.owner}
		for _, parameter := range fn.TypeParameters {
			declaration.Parameters = append(declaration.Parameters, genericplan.TypeParameter{Name: parameter})
		}
		t := transform{binding: x.checkBinding, expression: func(e *ast.Expression) error {
			if err := x.checkExpression(e); err != nil {
				return err
			}
			if e.Kind != syntax.KindSpecialize {
				return nil
			}
			name, err := x.target(e)
			if err != nil {
				return err
			}
			dependency := genericplan.Dependency{Declaration: x.id(name)}
			request := genericplan.Request{Declaration: x.id(name)}
			for _, argument := range e.TypeArguments {
				if argument.Kind != ast.TypeKindNamed || len(argument.Arguments) != 0 {
					return fmt.Errorf("generic argument must be a scalar or direct type parameter")
				}
				if slices.Contains(fn.TypeParameters, argument.Path) {
					dependency.Arguments = append(dependency.Arguments, genericplan.ParameterType(argument.Path))
				} else {
					if !slices.Contains(scalarTypes, argument.Path) {
						return fmt.Errorf("unsupported concrete generic type %q", argument.Path)
					}
					dependency.Arguments = append(dependency.Arguments, genericplan.ConcreteType(genericplan.TypeID(argument.Path)))
					request.Arguments = append(request.Arguments, genericplan.TypeID(argument.Path))
				}
			}
			if len(fn.TypeParameters) != 0 {
				declaration.Dependencies = append(declaration.Dependencies, dependency)
			} else {
				requests = append(requests, request)
			}
			return nil
		}}
		if _, err := t.function(fn); err != nil {
			return nil, genericplan.Plan{}, fmt.Errorf("generic source %s: %w", fn.Name, err)
		}
		if len(fn.TypeParameters) != 0 {
			body, err := json.Marshal(fn)
			if err != nil {
				return nil, genericplan.Plan{}, err
			}
			declaration.Definition = genericplan.DigestText("source-function-ast-v0", string(body))
			if err := b.AddDeclaration(declaration); err != nil {
				return nil, genericplan.Plan{}, err
			}
		}
	}
	catalog, err := b.Build()
	if err != nil {
		return nil, genericplan.Plan{}, err
	}
	plan, err := catalog.Plan(requests, limits)
	if err != nil {
		return nil, genericplan.Plan{}, err
	}
	names := map[string]string{}
	for _, specialization := range plan.Specializations {
		arguments := make([]genericplan.TypeID, len(specialization.Bindings))
		for i, binding := range specialization.Bindings {
			arguments[i] = binding.Type
		}
		names[requestKey(specialization.Declaration, arguments)] = generatedPrefix + string(specialization.ID)
	}
	result := *x.module
	result.Functions = nil
	for _, fn := range x.module.Functions {
		if len(fn.TypeParameters) != 0 {
			continue
		}
		concrete, err := x.instantiate(fn, nil, names)
		if err != nil {
			return nil, genericplan.Plan{}, err
		}
		result.Functions = append(result.Functions, concrete)
	}
	for _, specialization := range plan.Specializations {
		name := strings.TrimPrefix(string(specialization.Declaration), string(x.id("")))
		bindings := map[string]ast.TypeReference{}
		for _, binding := range specialization.Bindings {
			bindings[binding.Parameter] = ast.NamedTypeReference(string(binding.Type), x.templates[name].Span)
		}
		concrete, err := x.instantiate(x.templates[name], bindings, names)
		if err != nil {
			return nil, genericplan.Plan{}, err
		}
		concrete.Name = generatedPrefix + string(specialization.ID)
		concrete.TypeParameters = nil
		result.Functions = append(result.Functions, concrete)
	}
	return &result, plan, nil
}

func (x *expander) target(e *ast.Expression) (string, error) {
	if len(e.Children) != 1 || e.Children[0].Kind != syntax.KindPath {
		return "", fmt.Errorf("generic call requires a direct function path")
	}
	name := e.Children[0].Text
	if _, ok := x.templates[name]; !ok {
		return "", fmt.Errorf("unknown local generic function %q", e.Children[0].Text)
	}
	return name, nil
}

func (x *expander) checkBinding(name string) error {
	if _, generic := x.templates[name]; generic || strings.HasPrefix(name, generatedPrefix) {
		return fmt.Errorf("binding %q shadows a generic function or reserved name", name)
	}
	return nil
}

func (x *expander) checkExpression(e *ast.Expression) error {
	if e.Kind == syntax.KindPath && strings.Contains(e.Text, generatedPrefix) {
		return fmt.Errorf("source cannot reference generated generic names")
	}
	if e.Kind == syntax.KindCall && len(e.Children) != 0 && e.Children[0].Kind == syntax.KindPath {
		name := strings.TrimPrefix(e.Children[0].Text, x.module.Path+"::")
		if _, generic := x.templates[name]; generic {
			return fmt.Errorf("generic function %q requires explicit type arguments", name)
		}
	}
	return nil
}

func requestKey(declaration genericplan.DeclarationID, arguments []genericplan.TypeID) string {
	parts := make([]string, len(arguments))
	for i, argument := range arguments {
		parts[i] = string(argument)
	}
	return string(declaration) + "<" + strings.Join(parts, ",") + ">"
}

func (x *expander) instantiate(fn ast.Function, bindings map[string]ast.TypeReference, names map[string]string) (ast.Function, error) {
	t := transform{bindings: bindings, expression: func(e *ast.Expression) error {
		if e.Kind != syntax.KindSpecialize {
			return nil
		}
		name, err := x.target(e)
		if err != nil {
			return err
		}
		arguments := make([]genericplan.TypeID, len(e.TypeArguments))
		for i, argument := range e.TypeArguments {
			arguments[i] = genericplan.TypeID(argument.Path)
		}
		generated, ok := names[requestKey(x.id(name), arguments)]
		if !ok {
			return fmt.Errorf("specialization missing from verified plan")
		}
		*e = ast.Expression{Kind: syntax.KindPath, Span: e.Span, Text: generated}
		return nil
	}}
	return t.function(fn)
}
