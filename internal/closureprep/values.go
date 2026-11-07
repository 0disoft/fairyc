package closureprep

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/syntax"
)

// ClosureValueType consumes only structural source types, never diagnostic Text.
func ClosureValueType(typ ast.TypeReference) bool {
	direct := func(t ast.TypeReference) bool {
		return t.Kind == ast.TypeKindNamed && t.Path == "i32" && len(t.Arguments) == 0
	}
	return typ.Kind == ast.TypeKindClosureFunction && typ.ABI == "" && typ.Result != nil &&
		direct(*typ.Result) && len(typ.Parameters) == 1 && direct(typ.Parameters[0])
}

func (x *lifter) valueBinding(statement ast.Statement, closure *ast.Closure, env map[string]local, owner string) (ast.Statement, error) {
	x.next++
	name := fmt.Sprintf("%s%s_%d", prefix, owner, x.next)
	fn := closure.Function
	fn.Name, fn.NameSpan = name, statement.Binding.NameSpan
	fn.Parameters = append(slices.Clone(closure.Captures), closure.Function.Parameters...)
	fn.LiftedClosure = &ast.LiftedClosure{CaptureCount: len(closure.Captures)}
	constructor := *statement.Binding.Expression
	constructor.ClosureValue = &ast.ClosureValue{Target: name}
	constructor.Closure = nil
	constructor.Children = nil
	for _, capture := range closure.Captures {
		if _, ok := env[capture.Name]; ok {
			return statement, fail(capture.Span, "closure values cannot be captured")
		}
		constructor.Children = append(constructor.Children, &ast.Expression{Kind: syntax.KindPath, Text: capture.Name, Span: capture.NameSpan})
	}
	binding := *statement.Binding
	binding.Expression = &constructor
	statement.Binding = &binding
	x.functions = append(x.functions, fn)
	return statement, nil
}

// ValidateValueTarget rechecks the real safe pure function behind private
// lifting metadata. Runtime/artifact verification must independently check HIR.
func ValidateValueTarget(fn ast.Function) error {
	if fn.LiftedClosure == nil || !strings.HasPrefix(fn.Name, prefix) {
		return fail(fn.Span, "closure value has no private lifted target")
	}
	count := fn.LiftedClosure.CaptureCount
	if count < 0 || count > 4 || len(fn.Parameters) != count+1 {
		return fail(fn.Span, "closure value target capture/argument count")
	}
	captures := fn.Parameters[:count]
	fn.Parameters = fn.Parameters[count:]
	return validate(&ast.Closure{Captures: captures, Function: fn}, fn.Span)
}
