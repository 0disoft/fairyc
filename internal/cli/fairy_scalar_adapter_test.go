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
		if kind < 0 || start < 0 || end <= start || end > len(input) || children < 0 || children > 256 || op != 0 {
			return nil
		}
		if ns < 0 || ne < ns || ne > len(input) || (ns != 0 || ne != 0) && (ns < start || ne > end || ns == ne) {
			return nil
		}
		n := &scalarWireNode{kind: kind, span: diagnostic.Span{StartByte: start, EndByte: end}, name: diagnostic.Span{StartByte: ns, EndByte: ne}}
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
		return n.kind == 301 && len(n.children) == 0 && n.name == zero && text(n.span) == "i32"
	}
	if !identifier(text(root.name)) || !typeOK(root.children[0]) {
		return fail()
	}
	fn := ast.Function{Name: text(root.name), NameSpan: root.name, Span: root.span, ResultType: ast.NamedTypeReference("i32", root.children[0].span)}
	last := root.name.EndByte
	for _, p := range root.children[1 : len(root.children)-1] {
		if p.kind != 91 || len(p.children) != 1 || !typeOK(p.children[0]) || !identifier(text(p.name)) || p.span.StartByte < last || p.children[0].span.EndByte > p.name.StartByte {
			return fail()
		}
		fn.Parameters = append(fn.Parameters, ast.Parameter{Name: text(p.name), NameSpan: p.name, Span: p.span, Type: ast.NamedTypeReference("i32", p.children[0].span)})
		last = p.span.EndByte
	}
	body := root.children[len(root.children)-1]
	if body.kind != 100 || body.name != zero || len(body.children) != 1 || body.span.StartByte < last {
		return fail()
	}
	ret := body.children[0]
	if ret.kind != 103 || ret.name != zero || len(ret.children) != 1 {
		return fail()
	}
	value := ret.children[0]
	if value.name != zero || len(value.children) != 0 || (value.kind != 200 && value.kind != 201) {
		return fail()
	}
	raw := text(value.span)
	expr := &ast.Expression{Kind: syntax.KindPath, Span: value.span, Text: raw}
	module := &ast.Module{Path: "main", Span: diagnostic.Span{StartByte: 0, EndByte: len(input)}, SourceSize: len(input), SourceLineStarts: []int{0}}
	if value.kind == 200 {
		if raw == "" || len(raw) > 1 && raw[0] == '0' {
			return fail()
		}
		for _, c := range raw {
			if c < '0' || c > '9' {
				return fail()
			}
		}
		expr.Kind = syntax.KindLiteral
		expr.Cooked = raw
		module.Literals = []ast.Literal{{Raw: raw, Kind: "integer", Cooked: raw}}
	} else if !identifier(raw) {
		return fail()
	}
	fn.Body = &ast.Block{Span: body.span, Statements: []ast.Statement{{Kind: syntax.KindReturn, Span: ret.span, Expression: expr}}}
	module.Functions = []ast.Function{fn}
	for i, b := range []byte(input) {
		if b == '\n' {
			module.SourceLineStarts = append(module.SourceLineStarts, i+1)
		}
	}
	return module, nil
}
