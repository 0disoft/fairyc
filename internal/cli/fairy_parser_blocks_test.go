package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func emitParserBlockOracle(t *testing.T, program *strings.Builder) string {
	t.Helper()
	input := "module main; fn i32 f(i32 a) { var x = 1; { let x = 2; call(x); } if (a > 0) { x += a; } else if (false) {} else { x = 3; } while (x < a) { while (true) { break; } if (false) { continue; } x += 1; } if (true) { return x; } return x; }"
	result, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || result.AST == nil {
		t.Fatalf("block oracle: %v", err)
	}
	program.WriteString("fn i32 block_tree(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 0);\n")
	count, entries := 0, 1
	var emitBlock func(*ast.Block) int
	var emitStatement func(ast.Statement) int
	checkExpression := func(field string, expression *ast.Expression) {
		if expression == nil {
			fmt.Fprintf(program, "assert(node.%s == 0);\n", field)
			return
		}
		kind := map[syntax.Kind]int{syntax.KindLiteral: 1, syntax.KindPath: 2, syntax.KindGroup: 3, syntax.KindPrefix: 4, syntax.KindBinary: 5, syntax.KindCall: 6}[expression.Kind]
		if kind == 0 {
			t.Fatalf("unsupported expression oracle %s", expression.Kind)
		}
		fmt.Fprintf(program, "{ let syntax::Expression expr = syntax::expression_at(ref parsed, node.%s); assert(expr.kind == %d && expr.start == %d && expr.end == %d); }\n", field, kind, expression.Span.StartByte, expression.Span.EndByte)
	}
	emitBlock = func(block *ast.Block) int {
		tail := 0
		var children []int
		for _, s := range block.Statements {
			child := emitStatement(s)
			children = append(children, child)
			count++
			fmt.Fprintf(program, "{ let syntax::Statement link = syntax::statement_at(ref parsed, %d); assert(link.kind == 7 && link.first == %d && link.second == %d && link.start == %d && link.end == %d); }\n", count, child, tail, s.Span.StartByte, s.Span.EndByte)
			tail = count
		}
		if block.Tail != nil {
			t.Fatal("tail outside admitted subset")
		}
		count++
		fmt.Fprintf(program, "{ let syntax::Statement node = syntax::statement_at(ref parsed, %d); assert(node.kind == 1 && node.start == %d && node.end == %d && node.first == %d && node.second == %d); }\n", count, block.Span.StartByte, block.Span.EndByte, tail, len(children))
		fmt.Fprintf(program, "assert(syntax::block_statement_count(ref parsed,%d) == %d);\n", count, len(children))
		for index, child := range children {
			fmt.Fprintf(program, "assert(syntax::block_statement(ref parsed,%d,%d) == %d);\n", count, index, child)
		}
		return count
	}
	emitStatement = func(s ast.Statement) int {
		// Go wraps block-like statements in expression-statement nodes. The
		// candidate's statement layer retains the block/if directly.
		if s.Kind == syntax.KindExprStmt && s.Expression != nil && (s.Expression.Kind == syntax.KindBlock || s.Expression.Kind == syntax.KindIf) {
			s.Kind = s.Expression.Kind
		}
		if s.Kind == syntax.KindBlock {
			return emitBlock(s.Expression.Block)
		}
		kind, first, second, entry := 0, 0, 0, 0
		expression := s.Expression
		switch s.Kind {
		case syntax.KindBinding:
			kind = 2
			entries++
			entry = entries
			expression = s.Binding.Expression
		case syntax.KindReturn:
			kind = 3
			entries++
			entry = entries
		case syntax.KindAssignment:
			kind = 4
		case syntax.KindExprStmt:
			kind = 6
		case syntax.KindWhile:
			kind = 8
			expression = s.Expression.Children[0]
			first = emitBlock(s.Expression.Children[1].Block)
		case syntax.KindBreak:
			kind, expression = 9, nil
		case syntax.KindContinue:
			kind, expression = 10, nil
		case syntax.KindIf:
			kind = 5
			expression = s.Expression.Children[0]
			first = emitBlock(s.Expression.Children[1].Block)
			if len(s.Expression.Children) > 2 {
				other := s.Expression.Children[2]
				if other.Kind == syntax.KindBlock {
					second = emitBlock(other.Block)
				} else {
					second = emitStatement(ast.Statement{Kind: other.Kind, Span: other.Span, Expression: other})
				}
			}
		default:
			t.Fatalf("unsupported statement oracle %s", s.Kind)
		}
		count++
		fmt.Fprintf(program, "{ let syntax::Statement node = syntax::statement_at(ref parsed,%d); assert(node.kind == %d && node.start == %d && node.end == %d && node.first == %d && node.second == %d && node.entry == %d);\n", count, kind, s.Span.StartByte, s.Span.EndByte, first, second, entry)
		checkExpression("expression", expression)
		checkExpression("target", s.Target)
		if kind == 4 {
			fmt.Fprintf(program, "let ref u8[] op = syntax::assignment_operator(ref parsed,%d); assert(op.len == %d);", count, len(s.Operator))
			for index, b := range []byte(s.Operator) {
				fmt.Fprintf(program, "assert(op[%d] == %d);", index, b)
			}
		}
		program.WriteString("}\n")
		return count
	}
	root := emitBlock(result.AST.Functions[0].Body)
	fmt.Fprintf(program, "assert(syntax::declaration_at(ref parsed,1).body == %d && syntax::statement_count(ref parsed) == %d && syntax::entry_count(ref parsed) == %d); return 0; }\n", root, count, entries)
	return input
}
