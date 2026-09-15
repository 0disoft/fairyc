package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/syntax"
)

// This private adapter accepts only this test's native writer output, not a
// public interchange format. It never invokes the Go lexer/parser or AST lowerer.
type scalarWireNode struct {
	kind       int
	operator   string
	span, name diagnostic.Span
	children   []*scalarWireNode
}

func adaptFairyScalar(input, wire string) (*ast.Module, error) {
	fail := func() (*ast.Module, error) { return nil, fmt.Errorf("outside scalar adapter contract") }
	if len(input) > 4096 || len(wire) > 262144 {
		return fail()
	}
	fields := strings.Fields(wire)
	if len(fields) > 36000 {
		return fail()
	}
	index := 0
	read := func() int {
		if index >= len(fields) {
			return -1
		}
		text := fields[index]
		index++
		for _, c := range text {
			if c < '0' || c > '9' {
				return -1
			}
		}
		v, err := strconv.Atoi(text)
		if err != nil {
			return -1
		}
		return v
	}
	if read() != 1 || read() != len(input) {
		return fail()
	}
	for _, b := range []byte(input) {
		if read() != int(b) {
			return fail()
		}
	}
	count := 0
	var node func(int) *scalarWireNode
	node = func(depth int) *scalarWireNode {
		count++
		if count > 256 || depth >= 64 {
			return nil
		}
		kind, start, end, children, ns, ne, op := read(), read(), read(), read(), read(), read(), read()
		if kind < 0 || start < 0 || end <= start || end > len(input) || children < 0 || children > 256 || op < 0 || op > 2 {
			return nil
		}
		if ns < 0 || ne < ns || ne > len(input) || (ns != 0 || ne != 0) && (ns < start || ne > end || ns == ne) {
			return nil
		}
		n := &scalarWireNode{kind: kind, span: diagnostic.Span{StartByte: start, EndByte: end}, name: diagnostic.Span{StartByte: ns, EndByte: ne}}
		if op > 0 {
			for i := 0; i < op; i++ {
				b := read()
				if b < 0 || b > 127 {
					return nil
				}
				n.operator += string(byte(b))
			}
			valid := kind == 104 && n.operator == "=" || kind == 203 && (n.operator == "+" || n.operator == "-")
			if kind == 204 {
				switch n.operator {
				case "+", "-", "*", "==", "!=", "<", "<=", ">", ">=":
					valid = true
				}
			}
			if !valid {
				return nil
			}
		} else if kind == 104 || kind == 203 || kind == 204 {
			return nil
		}
		for i := 0; i < children; i++ {
			child := node(depth + 1)
			if child == nil || child.span.StartByte < start || child.span.EndByte > end {
				return nil
			}
			n.children = append(n.children, child)
		}
		return n
	}
	root := node(0)
	if root == nil || index != len(fields) || root.kind != 90 || len(root.children) < 2 || len(root.children) > 18 {
		return fail()
	}
	if strings.TrimSpace(input[:root.span.StartByte]) != "module main;" || strings.TrimSpace(input[root.span.EndByte:]) != "" {
		return fail()
	}
	text := func(s diagnostic.Span) string { return input[s.StartByte:s.EndByte] }
	zero := diagnostic.Span{}
	identifier := func(s string) bool {
		if s == "" {
			return false
		}
		for i, c := range s {
			if c != '_' && !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(i > 0 && c >= '0' && c <= '9') {
				return false
			}
		}
		return true
	}
	typeOK := func(n *scalarWireNode) bool {
		return n.kind == 301 && len(n.children) == 0 && n.name == zero && (text(n.span) == "i32" || text(n.span) == "bool")
	}
	if !identifier(text(root.name)) || !typeOK(root.children[0]) {
		return fail()
	}
	fn := ast.Function{Name: text(root.name), NameSpan: root.name, Span: root.span, ResultType: ast.NamedTypeReference(text(root.children[0].span), root.children[0].span)}
	last := root.name.EndByte
	for _, p := range root.children[1 : len(root.children)-1] {
		if p.kind != 91 || len(p.children) != 1 || !typeOK(p.children[0]) || !identifier(text(p.name)) || p.span.StartByte < last || p.children[0].span.EndByte > p.name.StartByte {
			return fail()
		}
		fn.Parameters = append(fn.Parameters, ast.Parameter{Name: text(p.name), NameSpan: p.name, Span: p.span, Type: ast.NamedTypeReference(text(p.children[0].span), p.children[0].span)})
		last = p.span.EndByte
	}
	body := root.children[len(root.children)-1]
	if body.kind != 100 || body.name != zero || len(body.children) == 0 || body.span.StartByte < last {
		return fail()
	}
	module := &ast.Module{Path: "main", Span: diagnostic.Span{StartByte: 0, EndByte: len(input)}, SourceSize: len(input), SourceLineStarts: []int{0}}
	var expression func(*scalarWireNode, int) *ast.Expression
	expression = func(value *scalarWireNode, depth int) *ast.Expression {
		if depth >= 64 || value.name != zero {
			return nil
		}
		expr := &ast.Expression{Span: value.span}
		arity := 0
		switch value.kind {
		case 200:
			raw := text(value.span)
			literalKind := "integer"
			if raw == "true" || raw == "false" {
				literalKind = "boolean"
			} else {
				if raw == "" || len(raw) > 1 && raw[0] == '0' {
					return nil
				}
				for _, c := range raw {
					if c < '0' || c > '9' {
						return nil
					}
				}
			}
			expr.Kind = syntax.KindLiteral
			expr.Text = raw
			expr.Cooked = raw
			module.Literals = append(module.Literals, ast.Literal{Raw: raw, Kind: literalKind, Cooked: raw})
		case 201:
			expr.Kind = syntax.KindPath
			expr.Text = text(value.span)
			if !identifier(expr.Text) {
				return nil
			}
		case 202:
			expr.Kind = syntax.KindGroup
			arity = 1
		case 203:
			expr.Kind = syntax.KindPrefix
			expr.Text = value.operator
			arity = 1
		case 204:
			expr.Kind = syntax.KindBinary
			expr.Text = value.operator
			arity = 2
		case 207:
			expr.Kind = syntax.KindIf
			arity = 3
		default:
			return nil
		}
		if len(value.children) != arity {
			return nil
		}
		last := value.span.StartByte
		for i, child := range value.children {
			if child.span.StartByte < last {
				return nil
			}
			var part *ast.Expression
			if value.kind == 207 && i > 0 {
				if child.kind != 100 || child.name != zero || len(child.children) != 1 {
					return nil
				}
				tail := expression(child.children[0], depth+2)
				if tail == nil {
					return nil
				}
				part = &ast.Expression{Kind: syntax.KindBlock, Span: child.span, Block: &ast.Block{Span: child.span, Tail: tail}}
			} else {
				part = expression(child, depth+1)
			}
			if part == nil {
				return nil
			}
			expr.Children = append(expr.Children, part)
			last = child.span.EndByte
		}
		return expr
	}
	fn.Body = &ast.Block{Span: body.span}
	last = body.span.StartByte
	for i, statement := range body.children {
		if statement.span.StartByte < last {
			return fail()
		}
		last = statement.span.EndByte
		if i == len(body.children)-1 {
			if statement.kind != 103 || statement.name != zero || len(statement.children) != 1 {
				return fail()
			}
			expr := expression(statement.children[0], 3)
			if expr == nil {
				return fail()
			}
			fn.Body.Statements = append(fn.Body.Statements, ast.Statement{Kind: syntax.KindReturn, Span: statement.span, Expression: expr})
			continue
		}
		if statement.kind == 104 {
			if statement.name != zero || len(statement.children) != 2 || statement.children[0].kind != 201 || statement.children[0].span.EndByte > statement.children[1].span.StartByte {
				return fail()
			}
			target, value := expression(statement.children[0], 3), expression(statement.children[1], 3)
			if target == nil || value == nil {
				return fail()
			}
			fn.Body.Statements = append(fn.Body.Statements, ast.Statement{Kind: syntax.KindAssignment, Span: statement.span, Operator: statement.operator, Target: target, Expression: value})
			continue
		}
		if (statement.kind != 109 && statement.kind != 110) || len(statement.children) != 2 || !identifier(text(statement.name)) || !typeOK(statement.children[0]) {
			return fail()
		}
		typ, value := statement.children[0], statement.children[1]
		if typ.span.EndByte > statement.name.StartByte || statement.name.EndByte > value.span.StartByte {
			return fail()
		}
		expr := expression(value, 3)
		if expr == nil {
			return fail()
		}
		typeRef := ast.NamedTypeReference(text(typ.span), typ.span)
		binding := ast.Binding{Name: text(statement.name), Mutable: statement.kind == 110, Span: statement.span, NameSpan: statement.name, Type: &typeRef, Expression: expr}
		fn.Body.Statements = append(fn.Body.Statements, ast.Statement{Kind: syntax.KindBinding, Span: statement.span, Binding: &binding})
		module.Bindings = append(module.Bindings, binding)
	}
	module.Functions = []ast.Function{fn}
	for i, b := range []byte(input) {
		if b == '\n' {
			module.SourceLineStarts = append(module.SourceLineStarts, i+1)
		}
	}
	return module, nil
}
