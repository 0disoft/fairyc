// Package genericsource connects explicit source functions to bounded static planning.
package genericsource

import (
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
)

// transform owns newly allocated syntax. Source locations keep pointing at the
// template; no source text is reparsed and no caller-owned AST is rewritten.
type transform struct {
	bindings   map[string]ast.TypeReference
	expression func(*ast.Expression) error
	binding    func(string, diagnostic.Span) error
}

func (t transform) typ(v ast.TypeReference) ast.TypeReference {
	if replacement, ok := t.bindings[v.Path]; ok && v.Kind == ast.TypeKindNamed && len(v.Arguments) == 0 {
		replacement.Span = v.Span
		return (transform{}).typ(replacement)
	}
	v.Element = t.typePointer(v.Element)
	v.Result = t.typePointer(v.Result)
	v.Arguments = t.types(v.Arguments)
	v.Parameters = t.types(v.Parameters)
	if v.Length != nil {
		length := *v.Length
		v.Length = &length
	}
	return v
}

func (t transform) typePointer(v *ast.TypeReference) *ast.TypeReference {
	if v == nil {
		return nil
	}
	result := t.typ(*v)
	return &result
}

func (t transform) types(values []ast.TypeReference) []ast.TypeReference {
	if values == nil {
		return nil
	}
	result := make([]ast.TypeReference, len(values))
	for i, v := range values {
		result[i] = t.typ(v)
	}
	return result
}

func (t transform) function(v ast.Function) (ast.Function, error) {
	v.TypeParameters = slices.Clone(v.TypeParameters)
	v.Attributes = slices.Clone(v.Attributes)
	for i := range v.Attributes {
		v.Attributes[i].Arguments = slices.Clone(v.Attributes[i].Arguments)
	}
	v.Parameters = slices.Clone(v.Parameters)
	for i := range v.Parameters {
		if t.binding != nil {
			if err := t.binding(v.Parameters[i].Name, v.Parameters[i].Span); err != nil {
				return ast.Function{}, err
			}
		}
		v.Parameters[i].Type = t.typ(v.Parameters[i].Type)
	}
	v.ResultType = t.typ(v.ResultType)
	var err error
	v.Body, err = t.block(v.Body)
	return v, err
}

func (t transform) block(v *ast.Block) (*ast.Block, error) {
	if v == nil {
		return nil, nil
	}
	result := *v
	result.Statements = slices.Clone(v.Statements)
	for i := range result.Statements {
		if err := t.statement(&result.Statements[i]); err != nil {
			return nil, err
		}
	}
	var err error
	result.Tail, err = t.expr(v.Tail)
	return &result, err
}

func (t transform) statement(v *ast.Statement) error {
	var err error
	if v.Binding != nil {
		binding := *v.Binding
		if t.binding != nil {
			if err := t.binding(binding.Name, binding.Span); err != nil {
				return err
			}
		}
		binding.Type = t.typePointer(binding.Type)
		binding.Expression, err = t.expr(binding.Expression)
		if err != nil {
			return err
		}
		v.Binding = &binding
	}
	v.Target, err = t.expr(v.Target)
	if err != nil {
		return err
	}
	v.Expression, err = t.expr(v.Expression)
	if err != nil {
		return err
	}
	if v.For != nil {
		loop := *v.For
		if loop.Initializer != nil {
			statement := *loop.Initializer
			if err := t.statement(&statement); err != nil {
				return err
			}
			loop.Initializer = &statement
		}
		if loop.Post != nil {
			statement := *loop.Post
			if err := t.statement(&statement); err != nil {
				return err
			}
			loop.Post = &statement
		}
		loop.Condition, err = t.expr(loop.Condition)
		if err != nil {
			return err
		}
		loop.Body, err = t.block(loop.Body)
		if err != nil {
			return err
		}
		v.For = &loop
	}
	return nil
}

func (t transform) expr(v *ast.Expression) (*ast.Expression, error) {
	if v == nil {
		return nil, nil
	}
	result := *v
	result.Type = t.typePointer(v.Type)
	result.TypeArguments = t.types(v.TypeArguments)
	result.Children = slices.Clone(v.Children)
	for i, child := range v.Children {
		var err error
		result.Children[i], err = t.expr(child)
		if err != nil {
			return nil, err
		}
	}
	var err error
	result.Block, err = t.block(v.Block)
	if err != nil {
		return nil, err
	}
	if t.expression != nil {
		if err := t.expression(&result); err != nil {
			return nil, err
		}
	}
	return &result, nil
}
