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
	modules, plan, err := expandAll([]*expander{&x}, limits)
	if err != nil {
		return nil, genericplan.Plan{}, err
	}
	return modules[0], plan, nil
}

type expander struct {
	module      *ast.Module
	owner       genericplan.PackageID
	templates   map[string]ast.Function
	aggregates  map[string]aggregateTemplate
	names       map[string]bool
	imports     map[string]*expander
	allowPublic bool
	requests    map[string]diagnostic.Span
}

func (x *expander) id(name string) genericplan.DeclarationID {
	return genericplan.DeclarationID(string(x.owner) + "::" + x.module.Path + "::" + name)
}

func (x *expander) collect() error {
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
	for _, imported := range x.module.Imports {
		if x.names[imported.Alias] {
			return fmt.Errorf("duplicate import alias %q", imported.Alias)
		}
		x.names[imported.Alias] = true
	}
	for name := range x.names {
		if strings.HasPrefix(name, generatedPrefix) {
			return fmt.Errorf("reserved module name %q", name)
		}
	}
	for _, fn := range x.module.Functions {
		if x.names[fn.Name] || strings.HasPrefix(fn.Name, generatedPrefix) {
			return x.at(fn.NameSpan, FailureDeclaration, fmt.Errorf("duplicate or reserved function %q", fn.Name))
		}
		x.names[fn.Name] = true
		if len(fn.TypeParameters) == 0 {
			continue
		}
		if (fn.Public && !x.allowPublic) || fn.Owner != "" || fn.Name == "source_location" {
			return x.at(fn.NameSpan, FailureDeclaration, fmt.Errorf("generic function %q requires private module-local ownership", fn.Name))
		}
		for _, parameter := range fn.TypeParameters {
			if slices.Contains(scalarTypes, parameter) || slices.Contains([]string{"void", "never", "str", "Region", "Option", "Result"}, parameter) {
				return x.at(fn.Span, FailureDeclaration, fmt.Errorf("type parameter %q shadows a builtin", parameter))
			}
		}
		x.templates[fn.Name] = fn
	}
	return x.collectAggregateTemplates()
}

func (x *expander) declare(b *genericplan.Builder) ([]genericplan.Request, error) {
	var requests []genericplan.Request
	for _, fn := range x.module.Functions {
		declaration := genericplan.Declaration{ID: x.id(fn.Name), Owner: x.owner}
		for _, parameter := range fn.TypeParameters {
			declaration.Parameters = append(declaration.Parameters, genericplan.TypeParameter{Name: parameter})
		}
		var typeError error
		t := transform{binding: func(name string, span diagnostic.Span) error {
			return x.at(span, FailureDeclaration, x.checkBinding(name))
		}, expression: func(e *ast.Expression) (err error) {
			defer func() { err = x.at(e.Span, FailureRequest, err) }()
			if err := x.checkExpression(e); err != nil {
				return err
			}
			if e.Kind == syntax.KindRecord && len(e.TypeArguments) != 0 {
				return x.registerAggregateRecord(e, fn.TypeParameters, &declaration, &requests)
			}
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
				// Calls remain in the AST; the planner needs each static edge only once.
				if !slices.ContainsFunc(declaration.Dependencies, func(existing genericplan.Dependency) bool {
					return existing.Declaration == dependency.Declaration && slices.Equal(existing.Arguments, dependency.Arguments)
				}) {
					declaration.Dependencies = append(declaration.Dependencies, dependency)
				}
			} else {
				requests = append(requests, request)
				if x.requests == nil {
					x.requests = map[string]diagnostic.Span{}
				}
				key := requestKey(request.Declaration, request.Arguments)
				if previous, found := x.requests[key]; !found || e.Span.StartByte < previous.StartByte {
					x.requests[key] = e.Span
				}
			}
			return nil
		}}
		t.typeReference = func(ref *ast.TypeReference) {
			if typeError == nil {
				typeError = x.registerAggregateType(*ref, fn.TypeParameters, &declaration, &requests)
			}
		}
		if _, err := t.function(fn); err != nil {
			return nil, fmt.Errorf("generic source %s: %w", fn.Name, err)
		}
		if typeError != nil {
			return nil, typeError
		}
		if len(fn.TypeParameters) != 0 {
			body, err := json.Marshal(fn)
			if err != nil {
				return nil, err
			}
			declaration.Definition = genericplan.DigestText("source-function-ast-v0", string(body))
			if x.allowPublic {
				// Ordinary calls also depend on exact import binding, even when
				// they do not create an edge in the generic dependency graph.
				var imports []string
				for alias, target := range x.imports {
					imports = append(imports, alias+"="+string(target.owner)+"::"+target.module.Path)
				}
				slices.Sort(imports)
				encoded, err := json.Marshal(imports)
				if err != nil {
					return nil, err
				}
				declaration.Definition = genericplan.DigestText("source-package-function-v0", string(body)+"\n"+string(encoded))
			}
			if err := b.AddDeclaration(declaration); err != nil {
				return nil, x.at(fn.Span, FailureDeclaration, err)
			}
		}
	}
	aggregateRequests, err := x.declareAggregates(b)
	return append(requests, aggregateRequests...), err
}

func (x *expander) emit(plan genericplan.Plan, names map[string]string) (*ast.Module, error) {
	result := *x.module
	result.Functions = nil
	for _, fn := range x.module.Functions {
		if len(fn.TypeParameters) != 0 {
			continue
		}
		concrete, err := x.instantiate(fn, nil, names)
		if err != nil {
			return nil, err
		}
		result.Functions = append(result.Functions, concrete)
	}
	for _, specialization := range plan.Specializations {
		prefix := string(x.id(""))
		if !strings.HasPrefix(string(specialization.Declaration), prefix) {
			continue
		}
		name := strings.TrimPrefix(string(specialization.Declaration), prefix)
		if _, belongs := x.templates[name]; !belongs {
			continue
		}
		bindings := map[string]ast.TypeReference{}
		for _, binding := range specialization.Bindings {
			bindings[binding.Parameter] = ast.NamedTypeReference(string(binding.Type), x.templates[name].Span)
		}
		concrete, err := x.instantiate(x.templates[name], bindings, names)
		if err != nil {
			return nil, err
		}
		concrete.Name = generatedPrefix + string(specialization.ID)
		concrete.TypeParameters = nil
		result.Functions = append(result.Functions, concrete)
	}
	if err := x.emitAggregates(&result, plan, names); err != nil {
		return nil, err
	}
	return &result, nil
}

type callTarget struct {
	module *expander
	name   string
	prefix string
}

func (t callTarget) id() genericplan.DeclarationID { return t.module.id(t.name) }

func (x *expander) lookup(path string) (callTarget, bool) {
	parts := strings.Split(path, "::")
	owner, name, prefix := x, parts[0], ""
	if len(parts) == 2 {
		owner, name, prefix = x.imports[parts[0]], parts[1], parts[0]+"::"
	} else if len(parts) != 1 {
		return callTarget{}, false
	}
	if owner == nil {
		return callTarget{}, false
	}
	_, found := owner.templates[name]
	return callTarget{module: owner, name: name, prefix: prefix}, found
}

func (x *expander) target(e *ast.Expression) (callTarget, error) {
	if len(e.Children) != 1 || e.Children[0].Kind != syntax.KindPath {
		return callTarget{}, fmt.Errorf("generic call requires a direct function path")
	}
	target, found := x.lookup(e.Children[0].Text)
	if !found {
		return callTarget{}, x.at(e.Span, FailureUnknown, fmt.Errorf("unknown generic function %q", e.Children[0].Text))
	}
	if target.module != x && !target.module.templates[target.name].Public {
		return callTarget{}, x.at(e.Span, FailurePrivate, fmt.Errorf("generic function %q is private", e.Children[0].Text))
	}
	return target, nil
}

func (x *expander) checkBinding(name string) error {
	if _, generic := x.aggregates[name]; generic {
		return fmt.Errorf("binding %q shadows a generic aggregate", name)
	}
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
		name := e.Children[0].Text
		_, importedGeneric := x.lookup(name)
		_, localGeneric := x.templates[strings.TrimPrefix(name, x.module.Path+"::")]
		if importedGeneric || (!x.allowPublic && localGeneric) {
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
	var typeError error
	t := transform{bindings: bindings, expression: func(e *ast.Expression) error {
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
	}}
	t.typeReference = func(ref *ast.TypeReference) {
		if typeError == nil {
			typeError = x.rewriteAggregateType(ref, names)
		}
	}
	result, err := t.function(fn)
	if err == nil {
		err = typeError
	}
	return result, err
}
