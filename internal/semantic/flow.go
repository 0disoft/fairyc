package semantic

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

type flowOrigin struct {
	name        string
	declaration hir.Span
}

type flowLoan struct {
	binding *symbol
	owner   *symbol
	fact    BorrowFact
}

type flowInventory struct {
	expressions []*ast.Expression
	assignments []*ast.Statement
	returns     []*ast.Expression
}

func (a *analyzer) flowCheck() {
	a.flowStarted = true
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

	loans := a.collectLoans(module, function.Body, origins)
	for index := range loans {
		loan := &loans[index]
		loan.fact.LastUse = lastUseOf(inventory.expressions, a.resolved, loan.binding, loan.fact.Borrow.EndByte)
		a.borrows = append(a.borrows, loan.fact)
	}
	a.checkLoanConflicts(module.file.Path, loans, inventory.assignments)

	if _, _, returnsBorrow := hir.BorrowElement(a.functions[function].typ); !returnsBorrow {
		return
	}
	for _, returned := range inventory.returns {
		a.checkBorrowReturn(module, returned, origins, loans)
	}
}

func (a *analyzer) collectLoans(module *moduleScope, block *ast.Block, origins map[*symbol]flowOrigin) []flowLoan {
	var loans []flowLoan
	for index := range block.Statements {
		statement := &block.Statements[index]
		if statement.Binding != nil {
			binding := statement.Binding
			bindingSymbol := a.bindingSymbols[binding]
			candidates := a.expressionOrigins(module, binding.Expression, origins)
			if bindingSymbol != nil && len(candidates) == 1 {
				origins[bindingSymbol] = candidates[0]
			}
			if bindingSymbol != nil && isBorrowExpression(binding.Expression) && len(candidates) == 1 {
				owner := a.resolved[binding.Expression.Children[0]]
				loans = append(loans, flowLoan{
					binding: bindingSymbol,
					owner:   owner,
					fact: BorrowFact{
						Borrow: span(module.file.Path, binding.Expression.Span),
						Kind:   borrowKind(binding.Expression.Text),
						Origin: candidates[0].name,
					},
				})
			}
		}
		loans = append(loans, a.collectLoansFromExpression(module, statement.Expression, origins)...)
	}
	loans = append(loans, a.collectLoansFromExpression(module, block.Tail, origins)...)
	return loans
}

func (a *analyzer) collectLoansFromExpression(module *moduleScope, expression *ast.Expression, origins map[*symbol]flowOrigin) []flowLoan {
	if expression == nil {
		return nil
	}
	var loans []flowLoan
	if expression.Block != nil {
		loans = append(loans, a.collectLoans(module, expression.Block, origins)...)
	}
	for _, child := range expression.Children {
		loans = append(loans, a.collectLoansFromExpression(module, child, origins)...)
	}
	return loans
}

func (a *analyzer) checkLoanConflicts(sourcePath string, loans []flowLoan, assignments []*ast.Statement) {
	for _, loan := range loans {
		if loan.owner == nil || loan.fact.LastUse == nil {
			continue
		}
		for _, assignment := range assignments {
			if assignment.Target == nil || a.resolved[assignment.Target] != loan.owner {
				continue
			}
			conflict := span(sourcePath, assignment.Target.Span)
			if conflict.StartByte <= loan.fact.Borrow.EndByte || conflict.StartByte >= loan.fact.LastUse.EndByte {
				continue
			}
			a.addDiagnostic("E2101", conflict,
				RelatedSpan{Role: "loan", Span: loan.fact.Borrow},
				RelatedSpan{Role: "last-use", Span: *loan.fact.LastUse},
			)
		}
	}
}

func (a *analyzer) checkBorrowReturn(module *moduleScope, expression *ast.Expression, origins map[*symbol]flowOrigin, loans []flowLoan) {
	if expression == nil {
		return
	}
	candidates := compactOrigins(a.expressionOrigins(module, expression, origins))
	if len(candidates) == 0 {
		a.addUnsupported(module.file.Path, expression.Span, "safe-borrow return with unknown origin")
		return
	}
	if len(candidates) > 1 {
		related := make([]RelatedSpan, 0, len(candidates))
		for _, candidate := range candidates {
			related = append(related, RelatedSpan{Role: "candidate-origin", Span: candidate.declaration})
		}
		a.addDiagnostic("E2103", span(module.file.Path, expression.Span), related...)
		return
	}
	origin := candidates[0]
	if len(origin.name) >= len("Local(") && origin.name[:len("Local(")] == "Local(" {
		a.addDiagnostic("E2102", span(module.file.Path, expression.Span), RelatedSpan{
			Role: "origin",
			Span: origin.declaration,
		})
		return
	}
	if expression.Kind == syntax.KindPath {
		returnedSymbol := a.resolved[expression]
		for index := range loans {
			if loans[index].binding == returnedSymbol {
				markBorrowEscaping(a.borrows, loans[index].fact.Borrow, span(module.file.Path, expression.Span))
				return
			}
		}
		lastUse := span(module.file.Path, expression.Span)
		_, mutable, _ := hir.BorrowElement(a.functionsContainingType(expression))
		kind := "shared"
		if mutable {
			kind = "mutable"
		}
		a.borrows = append(a.borrows, BorrowFact{
			Borrow:  lastUse,
			Kind:    kind,
			Origin:  origin.name,
			LastUse: &lastUse,
			Escapes: true,
		})
	}
}

func (a *analyzer) functionsContainingType(expression *ast.Expression) hir.Type {
	if value := a.resolved[expression]; value != nil {
		return value.typ
	}
	return hir.TypeInvalid
}

func (a *analyzer) expressionOrigins(module *moduleScope, expression *ast.Expression, origins map[*symbol]flowOrigin) []flowOrigin {
	if expression == nil {
		return nil
	}
	switch expression.Kind {
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
	case syntax.KindGroup:
		if len(expression.Children) == 1 {
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
	}
	return nil
}

func (a *analyzer) originForSymbol(value *symbol) flowOrigin {
	if value == nil {
		return flowOrigin{}
	}
	switch value.kind {
	case "parameter":
		return flowOrigin{
			name:        fmt.Sprintf("Param(%d)", a.parameterIndex[value]),
			declaration: span(value.source, value.declaration),
		}
	case "local":
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
		if statement.Kind == syntax.KindAssignment {
			inventory.assignments = append(inventory.assignments, statement)
		}
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
		return left.name == right.name && compareSpan(left.declaration, right.declaration) == 0
	})
}

func isBorrowExpression(expression *ast.Expression) bool {
	return expression != nil && expression.Kind == syntax.KindPrefix &&
		(expression.Text == "&" || expression.Text == "&mut") && len(expression.Children) == 1
}

func borrowKind(operator string) string {
	if operator == "&mut" {
		return "mutable"
	}
	return "shared"
}

func markBorrowEscaping(values []BorrowFact, borrow hir.Span, lastUse hir.Span) {
	for index := range values {
		if compareSpan(values[index].Borrow, borrow) == 0 {
			values[index].Escapes = true
			values[index].LastUse = &lastUse
		}
	}
}
