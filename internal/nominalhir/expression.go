package nominalhir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func (c *checker) expression(e hir.Expression, bindings map[hir.SymbolID]binding, depth int) (hir.Expression, error) {
	c.remaining--
	if depth > 128 || c.remaining < 0 {
		return hir.Expression{}, fmt.Errorf("nominal HIR expression budget exceeded")
	}
	if e.Block != nil || len(e.Arms)+len(e.Fields)+len(e.Effects) != 0 || e.UnsafeScope != nil || e.ForeignABI != "" || e.TargetType != "" || e.Cooked != "" || e.Variant != "" {
		return hir.Expression{}, fmt.Errorf("unsupported nominal expression metadata")
	}
	if !c.valueType(e.Type) && !(e.Kind == hir.ExpressionCall && e.Type == hir.TypeVoid) {
		return hir.Expression{}, fmt.Errorf("unknown or unsupported nominal value type %s", e.Type)
	}
	out := e
	out.Type = c.scalar(e.Type)
	out.Children = nil
	start := 0
	if e.Kind == hir.ExpressionCall {
		f, exists := c.functions[e.Symbol]
		if !exists || len(e.Children) != len(f.Parameters)+1 || e.Operator != "" {
			return hir.Expression{}, fmt.Errorf("invalid direct call")
		}
		head := e.Children[0]
		if head.Kind != hir.ExpressionPath || head.Symbol != e.Symbol || len(head.Children) != 0 || head.Block != nil || len(head.Arms)+len(head.Fields)+len(head.Effects) != 0 || head.UnsafeScope != nil || head.ForeignABI != "" || head.TargetType != "" || head.Operator != "" || head.Cooked != "" || head.Variant != "" || head.Type != "" {
			return hir.Expression{}, fmt.Errorf("invalid direct callee")
		}
		if e.Type != f.ResultType {
			return hir.Expression{}, mismatch(f.ResultType, e.Type, e.Span)
		}
		for i, p := range f.Parameters {
			if e.Children[i+1].Type != p.Type {
				return hir.Expression{}, mismatch(p.Type, e.Children[i+1].Type, e.Children[i+1].Span)
			}
		}
		out.Children = append(out.Children, head)
		start = 1
	}
	for _, child := range e.Children[start:] {
		value, err := c.expression(child, bindings, depth+1)
		if err != nil {
			return hir.Expression{}, err
		}
		out.Children = append(out.Children, value)
	}
	_, nominal := c.types[e.Type]
	switch e.Kind {
	case hir.ExpressionLiteral:
		if nominal || len(e.Children) != 0 || e.Operator != "" {
			return hir.Expression{}, fmt.Errorf("literal needs an explicit nominal wrap")
		}
	case hir.ExpressionPath:
		b, found := bindings[e.Symbol]
		if !found || !b.ready || len(e.Children) != 0 || e.Operator != "" {
			return hir.Expression{}, fmt.Errorf("invalid value path %d", e.Symbol)
		}
		if e.Type != b.typ {
			return hir.Expression{}, mismatch(b.typ, e.Type, e.Span)
		}
	case hir.ExpressionNominalWrap, hir.ExpressionNominalUnwrap, hir.ExpressionGroup:
		if len(e.Children) != 1 || e.Operator != "" {
			return hir.Expression{}, fmt.Errorf("invalid conversion shape")
		}
		child := e.Children[0]
		_, childNominal := c.types[child.Type]
		valid := e.Kind == hir.ExpressionGroup && e.Type == child.Type || e.Kind == hir.ExpressionNominalWrap && nominal && child.Type == hir.TypeUSize || e.Kind == hir.ExpressionNominalUnwrap && e.Type == hir.TypeUSize && childNominal
		if !valid {
			return hir.Expression{}, fmt.Errorf("invalid %s from %s to %s", e.Kind, child.Type, e.Type)
		}
		if e.Kind != hir.ExpressionGroup {
			return out.Children[0], nil
		}
	case hir.ExpressionBinary:
		if len(e.Children) != 2 || e.Children[0].Type != e.Children[1].Type {
			return hir.Expression{}, fmt.Errorf("comparison/arithmetic operands require identical types")
		}
		t := e.Children[0].Type
		comparison := e.Operator == "==" || e.Operator == "!=" || e.Operator == "<" || e.Operator == "<=" || e.Operator == ">" || e.Operator == ">="
		if comparison {
			if e.Type != hir.TypeBool || t == hir.TypeBool && e.Operator != "==" && e.Operator != "!=" {
				return hir.Expression{}, fmt.Errorf("invalid comparison result/operator")
			}
		} else {
			if t != hir.TypeUSize && t != hir.TypeI32 || e.Type != t {
				return hir.Expression{}, fmt.Errorf("arithmetic requires explicit nominal extraction")
			}
			switch e.Operator {
			case "+", "-", "*", "/", "%":
			default:
				return hir.Expression{}, fmt.Errorf("unsupported scalar operator")
			}
		}
	case hir.ExpressionCall:
	default:
		return hir.Expression{}, fmt.Errorf("unsupported nominal expression %s", e.Kind)
	}
	return out, nil
}
