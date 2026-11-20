package genericsource

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/syntax"
)

// sourceVisitor walks caller-owned syntax without cloning it. Planning uses the
// same walk for declaration discovery and materialization accounting.
type sourceVisitor struct {
	bindings      map[string]ast.TypeReference
	expression    func(*ast.Expression) error
	binding       func(string, diagnostic.Span) error
	typeReference func(*ast.TypeReference)
	nodes         *uint64
	inference     *inferenceScope
	inferred      map[*ast.Expression][]ast.TypeReference
}

func (v sourceVisitor) node() {
	if v.nodes != nil {
		*v.nodes++
	}
}

func (v sourceVisitor) typ(value ast.TypeReference) ast.TypeReference {
	v.node()
	if replacement, ok := v.bindings[value.Path]; ok && value.Kind == ast.TypeKindNamed && len(value.Arguments) == 0 {
		replacement.Span = value.Span
		return (sourceVisitor{nodes: v.nodes}).typ(replacement)
	}
	if value.Element != nil {
		element := v.typ(*value.Element)
		value.Element = &element
	}
	if value.Result != nil {
		result := v.typ(*value.Result)
		value.Result = &result
	}
	for _, argument := range value.Arguments {
		v.typ(argument)
	}
	for _, parameter := range value.Parameters {
		v.typ(parameter)
	}
	if value.Length != nil {
		v.node()
	}
	if v.typeReference != nil {
		v.typeReference(&value)
	}
	return value
}

func (v sourceVisitor) typePointer(value *ast.TypeReference) *ast.TypeReference {
	if value == nil {
		return nil
	}
	result := v.typ(*value)
	return &result
}

func (v sourceVisitor) function(value ast.Function) error {
	v.node()
	if v.inference != nil {
		v.inference = v.inference.child()
	}
	for _, attribute := range value.Attributes {
		v.node()
		for range attribute.Arguments {
			v.node()
		}
	}
	for _, parameter := range value.Parameters {
		v.node()
		if v.inference != nil {
			v.inference.bind(parameter.Name, v.inference.scalar(&parameter.Type))
		}
		if v.binding != nil {
			if err := v.binding(parameter.Name, parameter.Span); err != nil {
				return err
			}
		}
		v.typ(parameter.Type)
	}
	v.typ(value.ResultType)
	return v.block(value.Body)
}

func (v sourceVisitor) block(value *ast.Block) error {
	if value == nil {
		return nil
	}
	v.node()
	if v.inference != nil {
		v.inference = v.inference.child()
	}
	for i := range value.Statements {
		if err := v.statement(&value.Statements[i]); err != nil {
			return err
		}
	}
	return v.expr(value.Tail)
}

func (v sourceVisitor) statement(value *ast.Statement) error {
	if value == nil {
		return nil
	}
	v.node()
	if value.Binding != nil {
		v.node()
		if v.binding != nil {
			if err := v.binding(value.Binding.Name, value.Binding.Span); err != nil {
				return err
			}
		}
		if value.Binding.Type != nil {
			v.typ(*value.Binding.Type)
		}
		if err := v.expr(value.Binding.Expression); err != nil {
			return err
		}
		if v.inference != nil {
			typ := v.inference.scalar(value.Binding.Type)
			if value.Binding.Type == nil {
				var integer bool
				typ, integer = v.inference.evidence(value.Binding.Expression)
				if integer {
					typ = "i32"
				}
			}
			v.inference.bind(value.Binding.Name, typ)
		}
	}
	if err := v.expr(value.Target); err != nil {
		return err
	}
	if err := v.expr(value.Expression); err != nil {
		return err
	}
	if value.For != nil {
		v.node()
		if v.inference != nil {
			v.inference = v.inference.child()
		}
		if err := v.statement(value.For.Initializer); err != nil {
			return err
		}
		if err := v.statement(value.For.Post); err != nil {
			return err
		}
		if err := v.expr(value.For.Condition); err != nil {
			return err
		}
		if err := v.block(value.For.Body); err != nil {
			return err
		}
	}
	if value.ByteFor != nil {
		v.node()
		inert := sourceVisitor{bindings: v.bindings, nodes: v.nodes}
		loop := *value.ByteFor
		if loop.PreparedVector != nil {
			if loop.PreparedVector.Block == nil || len(loop.PreparedVector.Block.Statements) != 3 ||
				loop.PreparedVector.Block.Statements[0].Binding == nil ||
				loop.PreparedVector.Block.Statements[0].Binding.Expression != loop.View {
				return fmt.Errorf("vector loop preparation does not retain its written view")
			}
		}
		loop.Type = inert.typ(loop.Type)
		if err := inert.expr(loop.View); err != nil {
			return err
		}
		if value.Expression == nil {
			body := v
			if body.inference != nil {
				body.inference = body.inference.child()
				body.inference.bind(loop.Name, body.inference.scalar(&loop.Type))
			}
			if err := body.block(loop.Body); err != nil {
				return err
			}
		} else if err := inert.block(loop.Body); err != nil {
			return err
		}
		if err := inert.expr(loop.PreparedVector); err != nil {
			return err
		}
	}
	return nil
}

func (v sourceVisitor) expr(value *ast.Expression) error {
	if value == nil {
		return nil
	}
	v.node()
	result := *value
	if value.Closure != nil {
		v.node()
		result.Closure = value.Closure
		for _, capture := range value.Closure.Captures {
			v.node()
			v.typ(capture.Type)
		}
		closure := v
		if v.inference != nil {
			closure.inference = &inferenceScope{module: v.inference.module, parameters: v.inference.parameters}
			for _, capture := range value.Closure.Captures {
				closure.inference.bind(capture.Name, closure.inference.scalar(&capture.Type))
			}
		}
		if err := closure.function(value.Closure.Function); err != nil {
			return err
		}
	}
	if value.Type != nil {
		v.typ(*value.Type)
	}
	if value.CallLabels != nil {
		for range value.CallLabels.Labels {
			v.node()
		}
	}
	for _, argument := range value.TypeArguments {
		v.typ(argument)
	}
	if arguments := v.inferred[value]; len(arguments) != 0 {
		v.node()
		for _, argument := range arguments {
			v.typ(argument)
		}
	}
	if v.inference != nil && value.Kind == syntax.KindPattern && value.Text == "arm" && len(value.Children) == 2 {
		v.inference = v.inference.child()
		pattern := value.Children[0]
		if len(pattern.Children) > 1 {
			for _, binding := range pattern.Children[1:] {
				name := binding.Text
				if binding.Kind == syntax.KindField {
					name = binding.Cooked
				}
				v.inference.bind(name, "")
			}
		}
	}
	for _, child := range value.Children {
		if err := v.expr(child); err != nil {
			return err
		}
	}
	if err := v.block(value.Block); err != nil {
		return err
	}
	if v.expression != nil {
		if v.inference != nil {
			if err := v.inference.infer(value, &result); err != nil {
				return v.inference.module.at(value.Span, FailureRequest, err)
			}
			if len(v.inference.module.inferred[value]) != 0 {
				if err := v.expression(result.Children[0]); err != nil {
					return err
				}
			}
		}
		return v.expression(&result)
	}
	return nil
}
