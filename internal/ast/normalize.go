package ast

import "strings"

// NormalizeExpression renders the compact fixture oracle notation.
func NormalizeExpression(expression *Expression) string {
	if expression == nil {
		return "<missing>"
	}
	switch expression.Kind {
	case "literal", "path":
		return expression.Text
	case "group":
		return unaryForm("group", expression.Children)
	case "prefix", "binary":
		return listForm(expression.Text, expression.Children)
	case "propagate":
		return unaryForm("?", expression.Children)
	case "call":
		return listForm("call", expression.Children)
	case "index":
		return listForm("index", expression.Children)
	case "field-expression":
		parts := []string{"(field"}
		if len(expression.Children) > 0 {
			parts = append(parts, NormalizeExpression(expression.Children[0]))
		}
		parts = append(parts, expression.Text+")")
		return strings.Join(parts, " ")
	case "cast":
		parts := []string{"(as"}
		for _, child := range expression.Children {
			parts = append(parts, NormalizeExpression(child))
		}
		if expression.Type != nil {
			parts = append(parts, expression.Type.Text)
		}
		return strings.Join(parts, " ") + ")"
	case "error":
		return listForm("error", expression.Children)
	default:
		if expression.Text != "" {
			return expression.Text
		}
		return listForm(string(expression.Kind), expression.Children)
	}
}

func unaryForm(name string, children []*Expression) string {
	if len(children) == 0 {
		return "(" + name + ")"
	}
	return "(" + name + " " + NormalizeExpression(children[0]) + ")"
}

func listForm(name string, children []*Expression) string {
	parts := make([]string, 0, len(children)+1)
	parts = append(parts, "("+name)
	for _, child := range children {
		parts = append(parts, NormalizeExpression(child))
	}
	parts[len(parts)-1] += ")"
	return strings.Join(parts, " ")
}
