package genericsource

import (
	"encoding/json"
	"fmt"
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/syntax"
	"slices"
)

type staticOperation struct {
	capability genericplan.CapabilityID
	providers  map[string]ast.Function
	definition genericplan.Digest
	group      string
}

func marker(fn ast.Function, name string) (ast.Attribute, bool, error) {
	var value ast.Attribute
	found := false
	for _, a := range fn.Attributes {
		if a.Name == name {
			if found {
				return a, false, fmt.Errorf("duplicate @%s", name)
			}
			value = a
			found = true
		}
	}
	return value, found, nil
}

// Operations and implementations share one module owner. Importers may consume
// public operations; imported/orphan implementations and implicit lookup are closed.
func (x *expander) declareStaticOperations(b *genericplan.Builder) error {
	x.operations = map[string]*staticOperation{}
	for _, fn := range x.module.Functions {
		a, ok, err := marker(fn, "interface")
		if err != nil {
			return x.at(fn.Span, FailureDeclaration, err)
		}
		if !ok {
			continue
		}
		group := ""
		if len(a.Arguments) != 0 {
			if len(a.Arguments) != 1 || !validStaticGroupLabel(a.Arguments[0].Raw) || !fn.Public {
				return x.at(fn.Span, FailureDeclaration, fmt.Errorf("@interface(label) requires a public operation and a 1..64 byte ASCII identifier"))
			}
			group = a.Arguments[0].Raw
		}
		valid := len(fn.Attributes) == 1 && len(fn.TypeParameters) == 1 && len(fn.Parameters) <= 4 && !fn.Unsafe && fn.Owner == "" && fn.Body != nil && len(fn.Body.Statements) == 0 && fn.Body.Tail == nil
		if valid {
			valid = directType(fn.ResultType, fn.TypeParameters[0])
			for _, p := range fn.Parameters {
				valid = valid && directType(p.Type, fn.TypeParameters[0]) && !p.Mutable
			}
		}
		if !valid {
			return x.at(fn.Span, FailureDeclaration, fmt.Errorf("@interface requires one type parameter, T result, 0..4 T parameters and an empty safe body"))
		}
		x.operations[fn.Name] = &staticOperation{capability: genericplan.CapabilityID(string(x.id(fn.Name)) + "::operation"), providers: map[string]ast.Function{}, group: group}
	}
	for _, fn := range x.module.Functions {
		a, ok, err := marker(fn, "implements")
		if err != nil {
			return x.at(fn.Span, FailureDeclaration, err)
		}
		if !ok {
			continue
		}
		fail := func() error {
			return x.at(fn.Span, FailureDeclaration, fmt.Errorf("@implements requires a local operation and exact builtin scalar signature, once per tuple"))
		}
		if len(a.Arguments) != 2 || len(fn.Attributes) != 1 || len(fn.TypeParameters) != 0 || fn.Unsafe || fn.Owner != "" {
			return fail()
		}
		op := x.operations[a.Arguments[0].Raw]
		typ := a.Arguments[1].Raw
		if op == nil || !slices.Contains(scalarTypes, typ) {
			return fail()
		}
		stub := x.templates[a.Arguments[0].Raw]
		if len(fn.Parameters) != len(stub.Parameters) || !directType(fn.ResultType, typ) {
			return fail()
		}
		for _, p := range fn.Parameters {
			if !directType(p.Type, typ) || p.Mutable {
				return fail()
			}
		}
		if _, exists := op.providers[typ]; exists {
			return fail()
		}
		op.providers[typ] = fn
	}
	if err := x.defineStaticGroups(); err != nil {
		return err
	}
	names := make([]string, 0, len(x.operations))
	for name := range x.operations {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		op := x.operations[name]
		types := make([]string, 0, len(op.providers))
		for typ := range op.providers {
			types = append(types, typ)
		}
		slices.Sort(types)
		if op.group == "" {
			// Keep the legacy bytes and domain exact. Group members already
			// bind all siblings and all tuples through their shared definition.
			defs := []ast.Function{x.templates[name]}
			for _, typ := range types {
				defs = append(defs, op.providers[typ])
			}
			raw, err := json.Marshal(defs)
			if err != nil {
				return err
			}
			op.definition = genericplan.DigestText("source-static-operation-v1", string(raw))
		}
		if err := b.AddCapability(genericplan.CapabilityDescriptor{ID: op.capability, Owner: x.owner, Arity: 1, Definition: op.definition}); err != nil {
			return err
		}
		for _, typ := range types {
			fn := op.providers[typ]
			raw, err := json.Marshal(fn)
			if err != nil {
				return err
			}
			if err := b.AddImplementation(genericplan.Implementation{Capability: op.capability, Arguments: []genericplan.TypeID{genericplan.TypeID(typ)}, Owner: x.owner, Provider: genericplan.ProviderID(x.id(fn.Name)), ProviderDigest: genericplan.DigestText("source-static-provider-v1", string(raw))}); err != nil {
				return err
			}
		}
	}
	return nil
}

func directType(t ast.TypeReference, name string) bool {
	return t.Kind == ast.TypeKindNamed && t.Path == name && len(t.Arguments) == 0
}

func (x *expander) instantiateOperation(fn ast.Function, bindings map[string]ast.TypeReference) (ast.Function, error) {
	op := x.operations[fn.Name]
	if op == nil {
		return fn, nil
	}
	bound := bindings[fn.TypeParameters[0]]
	provider, ok := op.providers[bound.Path]
	if !ok {
		return ast.Function{}, fmt.Errorf("static provider missing after verified planning")
	}
	children := []*ast.Expression{{Kind: syntax.KindPath, Text: provider.Name, Span: fn.NameSpan}}
	for _, p := range fn.Parameters {
		children = append(children, &ast.Expression{Kind: syntax.KindPath, Text: p.Name, Span: p.NameSpan})
	}
	fn.Body = &ast.Block{Span: fn.Body.Span, Statements: []ast.Statement{{Kind: syntax.KindReturn, Span: fn.Span, Expression: &ast.Expression{Kind: syntax.KindCall, Span: fn.Span, Children: children}}}}
	return fn, nil
}
