package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func emitParserEntryOracle(t *testing.T, program *strings.Builder) string {
	t.Helper()
	input := "module main; fn i32 f(i32 a, usize b,) { let i32 x = a + 2; var y = (x * b); return y - 1; } fn void g() { return; }"
	reference, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || reference.AST == nil {
		t.Fatalf("entry oracle: %v", err)
	}
	program.WriteString("fn i32 entry_tree(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 0);\n")
	count := 0
	emit := func(kind int, span, name, typ diagnostic.Span, expression *ast.Expression) {
		fmt.Fprintf(program, "{ let syntax::Entry item = syntax::entry_at(ref parsed, %d); assert(item.kind == %d && item.start == %d && item.end == %d && item.name_start == %d && item.name_end == %d && item.type_start == %d && item.type_end == %d);\n", count, kind, span.StartByte, span.EndByte, name.StartByte, name.EndByte, typ.StartByte, typ.EndByte)
		if expression == nil {
			program.WriteString("assert(item.expression == 0);\n")
		} else {
			fmt.Fprintf(program, "let syntax::Expression expr = syntax::expression_at(ref parsed, item.expression); assert(expr.start == %d && expr.end == %d);\n", expression.Span.StartByte, expression.Span.EndByte)
		}
		program.WriteString("}\n")
		count++
	}
	for index, fn := range reference.AST.Functions {
		start := count
		for _, p := range fn.Parameters {
			emit(1, p.Span, p.NameSpan, p.Type.Span, nil)
		}
		for _, s := range fn.Body.Statements {
			if s.Binding != nil {
				kind := 2
				if s.Binding.Mutable {
					kind = 3
				}
				var typ diagnostic.Span
				if s.Binding.Type != nil {
					typ = s.Binding.Type.Span
				}
				emit(kind, s.Span, s.Binding.NameSpan, typ, s.Binding.Expression)
			} else {
				emit(4, s.Span, diagnostic.Span{}, diagnostic.Span{}, s.Expression)
			}
		}
		fmt.Fprintf(program, "{ let syntax::Declaration decl = syntax::declaration_at(ref parsed, %d); assert(decl.entry_start == %d && decl.entry_count == %d); }\n", index+1, start, count-start)
	}
	fmt.Fprintf(program, "assert(syntax::entry_count(ref parsed) == %d); return 0; }\n", count)
	return input
}
