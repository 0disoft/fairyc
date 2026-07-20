package semantic

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

type flowOrigin struct {
	name        string
	declaration hir.Span
	allocation  *hir.Span
}

type flowInventory struct {
	expressions []*ast.Expression
	returns     []*ast.Expression
}

func (a *analyzer) flowCheck() {
	a.flowStarted = true
	a.inferBorrowReturnSummaries()
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Functions {
			function := &module.file.Module.Functions[index]
			if a.functions[function] != nil {
				a.flowFunction(module, function)
			}
		}
	}
}

func (a *analyzer) flowFunction(module *moduleScope, function *ast.Function) {
	if function.Body == nil {
		return
	}
	inventory := flowInventory{}
	inventory.collectFunction(function.Body)
	origins := make(map[*symbol]flowOrigin)
	for index := range function.Parameters {
		parameter := &function.Parameters[index]
		value := a.parameter[parameter]
		if value == nil {
			continue
		}
		if _, _, ok := hir.BorrowElement(value.typ); ok {
			origins[value] = flowOrigin{
				name:        fmt.Sprintf("Param(%d)", index),
				declaration: span(module.file.Path, parameter.NameSpan),
			}
		}
	}

	a.collectFlowOrigins(module, function.Body, origins)
	a.borrowFlowFunction(module, function)
	a.collectRegionOriginFacts(module, &inventory, origins)
	a.resourceFlowFunction(module, function)
	a.resultFlowFunction(module, function)
}

func (a *analyzer) collectFlowOrigins(module *moduleScope, block *ast.Block, origins map[*symbol]flowOrigin) {
	if block == nil {
		return
	}
	for index := range block.Statements {
		statement := &block.Statements[index]
		if statement.Binding != nil {
			bindingSymbol := a.bindingSymbols[statement.Binding]
			candidates := a.expressionOrigins(module, statement.Binding.Expression, origins)
			if bindingSymbol != nil && len(candidates) == 1 {
				origins[bindingSymbol] = candidates[0]
			}
		}
		a.collectFlowOriginsFromExpression(module, statement.Expression, origins)
	}
	a.collectFlowOriginsFromExpression(module, block.Tail, origins)
}

func (a *analyzer) collectFlowOriginsFromExpression(module *moduleScope, expression *ast.Expression, origins map[*symbol]flowOrigin) {
	if expression == nil {
		return
	}
	if expression.Block != nil {
		a.collectFlowOrigins(module, expression.Block, origins)
	}
	for _, child := range expression.Children {
		a.collectFlowOriginsFromExpression(module, child, origins)
	}
}

func (a *analyzer) expressionOrigins(module *moduleScope, expression *ast.Expression, origins map[*symbol]flowOrigin) []flowOrigin {
	if expression == nil {
		return nil
	}
	switch expression.Kind {
	case syntax.KindLiteral:
		if isStaticBorrowLiteral(expression) {
			return []flowOrigin{{
				name:        "Static",
				declaration: span(module.file.Path, expression.Span),
			}}
		}
	case syntax.KindPath:
		value := a.resolved[expression]
		if origin, found := origins[value]; found {
			return []flowOrigin{origin}
		}
	case syntax.KindPrefix:
		if isBorrowExpression(expression) {
			if value := a.resolved[expression.Children[0]]; value != nil {
				return []flowOrigin{a.originForSymbol(value)}
			}
		}
	case syntax.KindGroup, syntax.KindUnsafe:
		if len(expression.Children) == 1 {
			return a.expressionOrigins(module, expression.Children[0], origins)
		}
	case syntax.KindSlice:
		if len(expression.Children) == 3 {
			return a.expressionOrigins(module, expression.Children[0], origins)
		}
	case syntax.KindBlock:
		if expression.Block != nil {
			return a.expressionOrigins(module, expression.Block.Tail, origins)
		}
	case syntax.KindIf:
		var result []flowOrigin
		for _, branch := range expression.Children[1:] {
			result = append(result, a.expressionOrigins(module, branch, origins)...)
		}
		return compactOrigins(result)
	case syntax.KindCall:
		if isRegionPlaceCall(expression) {
			receiver := expression.Children[0].Children[0]
			value := a.resolved[receiver]
			allocation := span(module.file.Path, expression.Span)
			switch {
			case value != nil && value.kind == "parameter" && value.typ == hir.TypeRegion:
				return []flowOrigin{{name: fmt.Sprintf("RegionParam(%d)", a.parameterIndex[value]), declaration: span(value.source, value.declaration), allocation: &allocation}}
			case value != nil && value.kind == "region":
				return []flowOrigin{{name: "LocalRegion(" + value.name + ")", declaration: span(value.source, value.declaration), allocation: &allocation}}
			}
		}
	}
	return nil
}

func (a *analyzer) collectRegionOriginFacts(module *moduleScope, inventory *flowInventory, origins map[*symbol]flowOrigin) {
	for _, expression := range inventory.expressions {
		if !isRegionPlaceCall(expression) {
			continue
		}
		candidates := compactOrigins(a.expressionOrigins(module, expression, origins))
		if len(candidates) != 1 {
			continue
		}
		origin := candidates[0]
		fact := RegionOriginFact{Operation: "place", Allocation: span(module.file.Path, expression.Span), Origin: origin.name}
		if binding := a.bindingForExpression(expression); binding != nil {
			fact.LastUse = lastUseOf(inventory.expressions, a.resolved, binding, expression.Span.EndByte)
		}
		for _, returned := range inventory.returns {
			for _, returnedOrigin := range a.expressionOrigins(module, returned, origins) {
				if sameFlowOrigin(origin, returnedOrigin) {
					fact.Escapes = true
				}
			}
		}
		a.regionOrigins = append(a.regionOrigins, fact)
		argument := expression.Children[1]
		argumentType := a.expressionTypes[argument]
		if info := a.resourceTypes[argumentType]; info != nil && info.symbol != nil {
			a.addDiagnostic("E3102", fact.Allocation, RelatedSpan{Role: "type", Span: span(info.symbol.source, info.symbol.declaration)})
		}
	}
}

func (a *analyzer) bindingForExpression(expression *ast.Expression) *symbol {
	for binding, value := range a.bindingSymbols {
		if binding.Expression == expression {
			return value
		}
	}
	return nil
}

func sameFlowOrigin(left, right flowOrigin) bool {
	return left.name == right.name && compareSpan(left.declaration, right.declaration) == 0
}

func (a *analyzer) originForSymbol(value *symbol) flowOrigin {
	if value == nil {
		return flowOrigin{}
	}
	switch value.kind {
	case "parameter", "local":
		// A direct `&place` borrows the storage occupied by that place in the
		// current call frame. Even when the place is a by-value parameter, that
		// storage is local to the callee and cannot be returned. Param(i) is
		// reserved for an already-borrowed parameter propagated by value.
		return flowOrigin{
			name:        "Local(" + value.name + ")",
			declaration: span(value.source, value.declaration),
		}
	default:
		return flowOrigin{name: "RawUnknown", declaration: span(value.source, value.declaration)}
	}
}

func (inventory *flowInventory) collectFunction(block *ast.Block) {
	inventory.collectBlock(block)
	if block != nil && block.Tail != nil {
		inventory.returns = append(inventory.returns, block.Tail)
	}
}

func (inventory *flowInventory) collectBlock(block *ast.Block) {
	if block == nil {
		return
	}
	for index := range block.Statements {
		statement := &block.Statements[index]
		if statement.Kind == syntax.KindReturn && statement.Expression != nil {
			inventory.returns = append(inventory.returns, statement.Expression)
		}
		if statement.Binding != nil {
			inventory.collectExpression(statement.Binding.Expression)
		}
		inventory.collectExpression(statement.Target)
		inventory.collectExpression(statement.Expression)
	}
	if block.Tail != nil {
		inventory.collectExpression(block.Tail)
	}
}

func (inventory *flowInventory) collectExpression(expression *ast.Expression) {
	if expression == nil {
		return
	}
	inventory.expressions = append(inventory.expressions, expression)
	if expression.Block != nil {
		inventory.collectBlock(expression.Block)
	}
	for _, child := range expression.Children {
		inventory.collectExpression(child)
	}
}

func lastUseOf(expressions []*ast.Expression, resolved map[*ast.Expression]*symbol, target *symbol, after int) *hir.Span {
	var last *hir.Span
	for _, expression := range expressions {
		if resolved[expression] != target || expression.Span.StartByte < after {
			continue
		}
		candidate := span(target.source, expression.Span)
		if last == nil || candidate.EndByte > last.EndByte {
			copy := candidate
			last = &copy
		}
	}
	return last
}

func compactOrigins(values []flowOrigin) []flowOrigin {
	slices.SortFunc(values, func(left, right flowOrigin) int {
		return compareSpan(left.declaration, right.declaration)
	})
	return slices.CompactFunc(values, func(left, right flowOrigin) bool {
		return sameFlowOrigin(left, right)
	})
}

func isBorrowExpression(expression *ast.Expression) bool {
	return expression != nil && expression.Kind == syntax.KindPrefix &&
		(expression.Text == "&" || expression.Text == "&mut") && len(expression.Children) == 1
}

func isStaticBorrowLiteral(expression *ast.Expression) bool {
	return expression != nil && expression.Kind == syntax.KindLiteral &&
		(strings.HasPrefix(expression.Text, "\"") || strings.HasPrefix(expression.Text, "b\"") || strings.HasPrefix(expression.Text, "c\""))
}

func borrowKind(operator string) string {
	if operator == "&mut" {
		return "mutable"
	}
	return "shared"
}
