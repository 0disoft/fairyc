package semantic

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// borrowValueState is the proof carried by one safe-borrow value at a source program point.
// Origins answer whether the value may escape; loans answer which local place remains protected.
// Pending loans are materialized only when the borrow is retained in a local, so call-bounded
// direct borrows do not accidentally become function-long facts.
type borrowValueState struct {
	known   bool
	origins []flowOrigin
	loans   []int
	pending []pendingBorrowLoan
}

type pendingBorrowLoan struct {
	owner  *symbol
	borrow hir.Span
	kind   string
	origin flowOrigin
}

type borrowLoanRecord struct {
	owner   *symbol
	binding *symbol
	fact    BorrowFact
}

type borrowEnvironment map[*symbol]borrowValueState

type borrowLiveMap map[*symbol]hir.Span

type borrowLiveContext struct {
	breakLive    borrowLiveMap
	continueLive borrowLiveMap
}

type borrowFlowAnalyzer struct {
	analyzer  *analyzer
	module    *moduleScope
	function  *ast.Function
	loans     []borrowLoanRecord
	liveAfter map[*ast.Statement]borrowLiveMap
}

func (a *analyzer) borrowFlowFunction(module *moduleScope, function *ast.Function) {
	if function == nil || function.Body == nil {
		return
	}
	flow := &borrowFlowAnalyzer{
		analyzer: a, module: module, function: function,
		liveAfter: make(map[*ast.Statement]borrowLiveMap),
	}
	flow.liveBlock(function.Body, make(borrowLiveMap), borrowLiveContext{})

	environment := make(borrowEnvironment)
	for index := range function.Parameters {
		parameter := &function.Parameters[index]
		value := a.parameter[parameter]
		if !isSafeBorrowSymbol(value) {
			continue
		}
		environment[value] = borrowValueState{
			known: true,
			origins: []flowOrigin{{
				name:        fmt.Sprintf("Param(%d)", index),
				declaration: span(module.file.Path, parameter.NameSpan),
			}},
		}
	}
	flow.evaluateBlock(function.Body, environment, true)
	for index := range flow.loans {
		a.borrows = append(a.borrows, flow.loans[index].fact)
	}
}

func (f *borrowFlowAnalyzer) evaluateBlock(block *ast.Block, environment borrowEnvironment, tailReturns bool) (borrowValueState, bool) {
	if block == nil {
		return borrowValueState{}, true
	}
	outer := make(map[*symbol]struct{}, len(environment))
	for value := range environment {
		outer[value] = struct{}{}
	}
	reachable := true
	for index := range block.Statements {
		if !reachable {
			break
		}
		reachable = f.evaluateStatement(&block.Statements[index], environment)
	}
	result := borrowValueState{}
	if reachable && block.Tail != nil {
		result, reachable = f.evaluateExpression(block.Tail, environment)
		if tailReturns {
			f.checkReturn(block.Tail, result)
			reachable = false
		}
	}
	for value := range environment {
		if _, retained := outer[value]; !retained {
			delete(environment, value)
		}
	}
	return result, reachable
}

func (f *borrowFlowAnalyzer) evaluateStatement(statement *ast.Statement, environment borrowEnvironment) bool {
	if statement == nil {
		return true
	}
	if statement.Binding != nil {
		value, reachable := f.evaluateExpression(statement.Binding.Expression, environment)
		binding := f.analyzer.bindingSymbols[statement.Binding]
		if reachable && isSafeBorrowSymbol(binding) {
			if !value.known {
				f.analyzer.addUnsupported(f.module.file.Path, statement.Binding.Expression.Span, "safe-borrow binding with unknown origin")
				delete(environment, binding)
			} else {
				environment[binding] = f.materialize(value, binding)
			}
		}
		return reachable
	}

	switch statement.Kind {
	case syntax.KindReturn:
		value, _ := f.evaluateExpression(statement.Expression, environment)
		f.checkReturn(statement.Expression, value)
		return false
	case syntax.KindAssignment:
		value, reachable := f.evaluateExpression(statement.Expression, environment)
		if !reachable {
			return false
		}
		target := f.analyzer.resolved[statement.Target]
		if isSafeBorrowSymbol(target) {
			if !value.known {
				f.analyzer.addUnsupported(f.module.file.Path, statement.Expression.Span, "safe-borrow assignment with unknown origin")
				delete(environment, target)
			} else {
				environment[target] = f.materialize(value, target)
			}
			return true
		}
		f.checkOwnerAssignment(statement, target, environment)
		return true
	case syntax.KindBreak, syntax.KindContinue:
		return false
	default:
		_, reachable := f.evaluateExpression(statement.Expression, environment)
		return reachable
	}
}

func (f *borrowFlowAnalyzer) evaluateExpression(expression *ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	if expression == nil {
		return borrowValueState{}, true
	}
	switch expression.Kind {
	case syntax.KindLiteral:
		if isStaticBorrowLiteral(expression) && f.expressionReturnsBorrow(expression) {
			origin := flowOrigin{name: "Static", declaration: span(f.module.file.Path, expression.Span)}
			return borrowValueState{
				known: true, origins: []flowOrigin{origin},
				pending: []pendingBorrowLoan{{borrow: span(f.module.file.Path, expression.Span), kind: "shared", origin: origin}},
			}, true
		}
		return borrowValueState{}, true
	case syntax.KindPath:
		value := f.analyzer.resolved[expression]
		if !isSafeBorrowSymbol(value) {
			return borrowValueState{}, true
		}
		state, found := environment[value]
		if !found {
			return borrowValueState{known: false}, true
		}
		f.markUse(state, span(f.module.file.Path, expression.Span))
		return cloneBorrowValue(state), true
	case syntax.KindPrefix:
		if isBorrowExpression(expression) {
			owner := f.analyzer.resolved[expression.Children[0]]
			if owner == nil {
				return borrowValueState{known: false}, true
			}
			origin := f.analyzer.originForSymbol(owner)
			return borrowValueState{
				known: true, origins: []flowOrigin{origin},
				pending: []pendingBorrowLoan{{
					owner: owner, borrow: span(f.module.file.Path, expression.Span),
					kind: borrowKind(expression.Text), origin: origin,
				}},
			}, true
		}
		return f.evaluateChildren(expression.Children, environment)
	case syntax.KindGroup, syntax.KindUnsafe:
		if len(expression.Children) == 1 {
			return f.evaluateExpression(expression.Children[0], environment)
		}
		return f.evaluateChildren(expression.Children, environment)
	case syntax.KindSlice:
		if len(expression.Children) != 3 {
			return borrowValueState{known: false}, true
		}
		receiver, reachable := f.evaluateExpression(expression.Children[0], environment)
		if !reachable {
			return receiver, false
		}
		for _, bound := range expression.Children[1:] {
			if _, reachable = f.evaluateExpression(bound, environment); !reachable {
				return borrowValueState{}, false
			}
		}
		return receiver, true
	case syntax.KindBlock:
		return f.evaluateBlock(expression.Block, environment, false)
	case syntax.KindIf:
		return f.evaluateIf(expression, environment)
	case syntax.KindMatch:
		return f.evaluateMatch(expression, environment)
	case syntax.KindWhile:
		return f.evaluateWhile(expression, environment)
	case syntax.KindCall:
		if isRegionPlaceCall(expression) {
			for index := range expression.Children {
				if _, reachable := f.evaluateExpression(expression.Children[index], environment); !reachable {
					return borrowValueState{}, false
				}
			}
			return f.regionBorrowValue(expression), true
		}
		if len(expression.Children) == 0 {
			return borrowValueState{known: false}, true
		}
		if _, reachable := f.evaluateExpression(expression.Children[0], environment); !reachable {
			return borrowValueState{}, false
		}
		var borrowArguments []borrowValueState
		for _, argument := range expression.Children[1:] {
			value, reachable := f.evaluateExpression(argument, environment)
			if !reachable {
				return borrowValueState{}, false
			}
			if f.expressionReturnsBorrow(argument) {
				borrowArguments = append(borrowArguments, value)
			}
		}
		if f.expressionReturnsBorrow(expression) {
			// Stage 0 functions may return only a borrow derived from one safe-borrow
			// parameter or Static. With one borrow argument the mapping is exact.
			// Multiple borrow arguments stay as a conservative union until function
			// summaries encode the selected parameter explicitly.
			if len(borrowArguments) == 0 {
				return borrowValueState{known: false}, true
			}
			return mergeBorrowValues(borrowArguments), true
		}
		return borrowValueState{}, true
	default:
		return f.evaluateChildren(expression.Children, environment)
	}
}

func (f *borrowFlowAnalyzer) evaluateChildren(children []*ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	result := borrowValueState{}
	for _, child := range children {
		value, reachable := f.evaluateExpression(child, environment)
		if !reachable {
			return value, false
		}
		result = value
	}
	return result, true
}

func (f *borrowFlowAnalyzer) evaluateIf(expression *ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	if len(expression.Children) < 2 {
		return borrowValueState{known: false}, true
	}
	if _, reachable := f.evaluateExpression(expression.Children[0], environment); !reachable {
		return borrowValueState{}, false
	}
	base := cloneBorrowEnvironment(environment)
	var branchValues []borrowValueState
	var branchEnvironments []borrowEnvironment
	for _, branch := range expression.Children[1:] {
		branchEnvironment := cloneBorrowEnvironment(base)
		value, reachable := f.evaluateExpression(branch, branchEnvironment)
		if reachable {
			branchValues = append(branchValues, value)
			branchEnvironments = append(branchEnvironments, branchEnvironment)
		}
	}
	if len(expression.Children) == 2 {
		branchEnvironments = append(branchEnvironments, base)
	}
	if len(branchEnvironments) == 0 {
		return borrowValueState{}, false
	}
	replaceBorrowEnvironment(environment, mergeBorrowEnvironments(base, branchEnvironments))
	if !f.expressionReturnsBorrow(expression) {
		return borrowValueState{}, true
	}
	return mergeBorrowValues(branchValues), true
}

func (f *borrowFlowAnalyzer) evaluateMatch(expression *ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	if len(expression.Children) < 2 {
		return borrowValueState{known: false}, true
	}
	if _, reachable := f.evaluateExpression(expression.Children[0], environment); !reachable {
		return borrowValueState{}, false
	}
	base := cloneBorrowEnvironment(environment)
	var branchValues []borrowValueState
	var branchEnvironments []borrowEnvironment
	for _, arm := range expression.Children[1:] {
		if arm == nil || len(arm.Children) < 2 {
			continue
		}
		branchEnvironment := cloneBorrowEnvironment(base)
		value, reachable := f.evaluateExpression(arm.Children[len(arm.Children)-1], branchEnvironment)
		if reachable {
			branchValues = append(branchValues, value)
			branchEnvironments = append(branchEnvironments, branchEnvironment)
		}
	}
	if len(branchEnvironments) == 0 {
		return borrowValueState{}, false
	}
	replaceBorrowEnvironment(environment, mergeBorrowEnvironments(base, branchEnvironments))
	if !f.expressionReturnsBorrow(expression) {
		return borrowValueState{}, true
	}
	return mergeBorrowValues(branchValues), true
}

func (f *borrowFlowAnalyzer) evaluateWhile(expression *ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	if len(expression.Children) != 2 {
		return f.evaluateChildren(expression.Children, environment)
	}
	if _, reachable := f.evaluateExpression(expression.Children[0], environment); !reachable {
		return borrowValueState{}, false
	}
	base := cloneBorrowEnvironment(environment)
	body := cloneBorrowEnvironment(base)
	_, bodyReachable := f.evaluateExpression(expression.Children[1], body)
	branches := []borrowEnvironment{base}
	if bodyReachable {
		branches = append(branches, body)
	}
	replaceBorrowEnvironment(environment, mergeBorrowEnvironments(base, branches))
	return borrowValueState{}, true
}

func (f *borrowFlowAnalyzer) regionBorrowValue(expression *ast.Expression) borrowValueState {
	if !isRegionPlaceCall(expression) {
		return borrowValueState{known: false}
	}
	receiver := expression.Children[0].Children[0]
	value := f.analyzer.resolved[receiver]
	allocation := span(f.module.file.Path, expression.Span)
	switch {
	case value != nil && value.kind == "parameter" && value.typ == hir.TypeRegion:
		return borrowValueState{known: true, origins: []flowOrigin{{
			name:        fmt.Sprintf("RegionParam(%d)", f.analyzer.parameterIndex[value]),
			declaration: span(value.source, value.declaration), allocation: &allocation,
		}}}
	case value != nil && value.kind == "region":
		return borrowValueState{known: true, origins: []flowOrigin{{
			name: "LocalRegion(" + value.name + ")", declaration: span(value.source, value.declaration), allocation: &allocation,
		}}}
	default:
		return borrowValueState{known: false}
	}
}

func (f *borrowFlowAnalyzer) materialize(value borrowValueState, binding *symbol) borrowValueState {
	value = cloneBorrowValue(value)
	for _, pending := range value.pending {
		loan := borrowLoanRecord{
			owner: pending.owner, binding: binding,
			fact: BorrowFact{Borrow: pending.borrow, Kind: pending.kind, Origin: pending.origin.name},
		}
		value.loans = append(value.loans, len(f.loans))
		f.loans = append(f.loans, loan)
	}
	value.pending = nil
	value.loans = uniqueInts(value.loans)
	value.origins = compactOrigins(value.origins)
	return value
}

func (f *borrowFlowAnalyzer) markUse(value borrowValueState, use hir.Span) {
	for _, loanID := range value.loans {
		if loanID < 0 || loanID >= len(f.loans) {
			continue
		}
		last := f.loans[loanID].fact.LastUse
		if last == nil || use.EndByte > last.EndByte {
			copy := use
			f.loans[loanID].fact.LastUse = &copy
		}
	}
}

func (f *borrowFlowAnalyzer) checkOwnerAssignment(statement *ast.Statement, owner *symbol, environment borrowEnvironment) {
	if statement == nil || owner == nil || statement.Target == nil {
		return
	}
	live := f.liveAfter[statement]
	seen := make(map[int]struct{})
	for binding, lastUse := range live {
		value, found := environment[binding]
		if !found {
			continue
		}
		for _, loanID := range value.loans {
			if loanID < 0 || loanID >= len(f.loans) || f.loans[loanID].owner != owner {
				continue
			}
			if _, duplicate := seen[loanID]; duplicate {
				continue
			}
			seen[loanID] = struct{}{}
			f.analyzer.addDiagnostic("E2101", span(f.module.file.Path, statement.Target.Span),
				RelatedSpan{Role: "loan", Span: f.loans[loanID].fact.Borrow},
				RelatedSpan{Role: "last-use", Span: lastUse},
			)
		}
	}
}

func (f *borrowFlowAnalyzer) checkReturn(expression *ast.Expression, value borrowValueState) {
	if expression == nil {
		return
	}
	if _, _, returnsBorrow := hir.BorrowElement(f.analyzer.functions[f.function].typ); !returnsBorrow {
		return
	}
	returnSpan := span(f.module.file.Path, expression.Span)
	candidates := compactOrigins(value.origins)
	if !value.known || len(candidates) == 0 {
		f.analyzer.addUnsupported(f.module.file.Path, expression.Span, "safe-borrow return with unknown origin")
		return
	}
	if len(candidates) > 1 {
		related := make([]RelatedSpan, 0, len(candidates))
		for _, candidate := range candidates {
			related = append(related, RelatedSpan{Role: "candidate-origin", Span: candidate.declaration})
		}
		f.analyzer.addDiagnostic("E2103", returnSpan, related...)
		return
	}
	origin := candidates[0]
	if strings.HasPrefix(origin.name, "Local(") {
		f.analyzer.addDiagnostic("E2102", returnSpan, RelatedSpan{Role: "origin", Span: origin.declaration})
		return
	}
	if strings.HasPrefix(origin.name, "LocalRegion(") {
		var related []RelatedSpan
		if origin.allocation != nil {
			related = append(related, RelatedSpan{Role: "allocation", Span: *origin.allocation})
		}
		f.analyzer.addDiagnostic("E3101", returnSpan, related...)
		return
	}

	marked := false
	for _, loanID := range value.loans {
		if loanID < 0 || loanID >= len(f.loans) {
			continue
		}
		copy := returnSpan
		f.loans[loanID].fact.LastUse = &copy
		f.loans[loanID].fact.Escapes = true
		marked = true
	}
	for _, pending := range value.pending {
		copy := returnSpan
		f.loans = append(f.loans, borrowLoanRecord{owner: pending.owner, fact: BorrowFact{
			Borrow: pending.borrow, Kind: pending.kind, Origin: pending.origin.name, LastUse: &copy, Escapes: true,
		}})
		marked = true
	}
	if !marked {
		kind := "shared"
		if _, mutable, _ := hir.BorrowElement(f.analyzer.functions[f.function].typ); mutable {
			kind = "mutable"
		}
		copy := returnSpan
		f.loans = append(f.loans, borrowLoanRecord{fact: BorrowFact{
			Borrow: returnSpan, Kind: kind, Origin: origin.name, LastUse: &copy, Escapes: true,
		}})
	}
}

func (f *borrowFlowAnalyzer) expressionReturnsBorrow(expression *ast.Expression) bool {
	if expression == nil {
		return false
	}
	_, _, borrow := hir.BorrowElement(f.analyzer.expressionTypes[expression])
	return borrow
}

func isSafeBorrowSymbol(value *symbol) bool {
	if value == nil {
		return false
	}
	_, _, borrow := hir.BorrowElement(value.typ)
	return borrow
}

func cloneBorrowValue(value borrowValueState) borrowValueState {
	value.origins = slices.Clone(value.origins)
	value.loans = slices.Clone(value.loans)
	value.pending = slices.Clone(value.pending)
	return value
}

func cloneBorrowEnvironment(value borrowEnvironment) borrowEnvironment {
	result := make(borrowEnvironment, len(value))
	for symbol, state := range value {
		result[symbol] = cloneBorrowValue(state)
	}
	return result
}

func replaceBorrowEnvironment(target, source borrowEnvironment) {
	clear(target)
	for symbol, state := range source {
		target[symbol] = cloneBorrowValue(state)
	}
}

func mergeBorrowEnvironments(base borrowEnvironment, branches []borrowEnvironment) borrowEnvironment {
	result := make(borrowEnvironment)
	for symbol := range base {
		values := make([]borrowValueState, 0, len(branches))
		for _, branch := range branches {
			if value, found := branch[symbol]; found {
				values = append(values, value)
			} else {
				values = append(values, borrowValueState{known: false})
			}
		}
		result[symbol] = mergeBorrowValues(values)
	}
	return result
}

func mergeBorrowValues(values []borrowValueState) borrowValueState {
	if len(values) == 0 {
		return borrowValueState{known: false}
	}
	result := borrowValueState{known: true}
	for _, value := range values {
		result.known = result.known && value.known
		result.origins = append(result.origins, value.origins...)
		result.loans = append(result.loans, value.loans...)
		result.pending = append(result.pending, value.pending...)
	}
	result.origins = compactOrigins(result.origins)
	result.loans = uniqueInts(result.loans)
	slices.SortFunc(result.pending, func(left, right pendingBorrowLoan) int {
		if value := compareSpan(left.borrow, right.borrow); value != 0 {
			return value
		}
		return strings.Compare(left.origin.name, right.origin.name)
	})
	result.pending = slices.CompactFunc(result.pending, func(left, right pendingBorrowLoan) bool {
		return left.owner == right.owner && compareSpan(left.borrow, right.borrow) == 0 && left.kind == right.kind && sameFlowOrigin(left.origin, right.origin)
	})
	return result
}

func uniqueInts(values []int) []int {
	slices.Sort(values)
	return slices.Compact(values)
}

func cloneBorrowLive(value borrowLiveMap) borrowLiveMap {
	result := make(borrowLiveMap, len(value))
	for symbol, use := range value {
		result[symbol] = use
	}
	return result
}

func mergeBorrowLive(values ...borrowLiveMap) borrowLiveMap {
	result := make(borrowLiveMap)
	for _, value := range values {
		for symbol, use := range value {
			if previous, found := result[symbol]; !found || use.EndByte > previous.EndByte {
				result[symbol] = use
			}
		}
	}
	return result
}

func sameBorrowLive(left, right borrowLiveMap) bool {
	if len(left) != len(right) {
		return false
	}
	for symbol, use := range left {
		other, found := right[symbol]
		if !found || compareSpan(use, other) != 0 {
			return false
		}
	}
	return true
}

func (f *borrowFlowAnalyzer) liveBlock(block *ast.Block, liveOut borrowLiveMap, context borrowLiveContext) borrowLiveMap {
	live := cloneBorrowLive(liveOut)
	if block == nil {
		return live
	}
	if block.Tail != nil {
		live = f.liveExpression(block.Tail, live, context)
	}
	for index := len(block.Statements) - 1; index >= 0; index-- {
		statement := &block.Statements[index]
		f.liveAfter[statement] = cloneBorrowLive(live)
		switch {
		case statement.Binding != nil:
			if binding := f.analyzer.bindingSymbols[statement.Binding]; isSafeBorrowSymbol(binding) {
				delete(live, binding)
			}
			live = f.liveExpression(statement.Binding.Expression, live, context)
		case statement.Kind == syntax.KindReturn:
			live = f.liveExpression(statement.Expression, make(borrowLiveMap), context)
		case statement.Kind == syntax.KindBreak:
			live = cloneBorrowLive(context.breakLive)
		case statement.Kind == syntax.KindContinue:
			live = cloneBorrowLive(context.continueLive)
		case statement.Kind == syntax.KindAssignment:
			if target := f.analyzer.resolved[statement.Target]; isSafeBorrowSymbol(target) {
				delete(live, target)
			}
			live = f.liveExpression(statement.Expression, live, context)
		default:
			live = f.liveExpression(statement.Expression, live, context)
		}
	}
	return live
}

func (f *borrowFlowAnalyzer) liveExpression(expression *ast.Expression, liveOut borrowLiveMap, context borrowLiveContext) borrowLiveMap {
	if expression == nil {
		return cloneBorrowLive(liveOut)
	}
	switch expression.Kind {
	case syntax.KindPath:
		live := cloneBorrowLive(liveOut)
		value := f.analyzer.resolved[expression]
		if isSafeBorrowSymbol(value) {
			use := span(f.module.file.Path, expression.Span)
			if previous, found := live[value]; !found || use.EndByte > previous.EndByte {
				live[value] = use
			}
		}
		return live
	case syntax.KindBlock:
		return f.liveBlock(expression.Block, liveOut, context)
	case syntax.KindIf:
		if len(expression.Children) < 2 {
			return f.liveChildren(expression.Children, liveOut, context)
		}
		var branchLive []borrowLiveMap
		for _, branch := range expression.Children[1:] {
			branchLive = append(branchLive, f.liveExpression(branch, liveOut, context))
		}
		if len(expression.Children) == 2 {
			branchLive = append(branchLive, cloneBorrowLive(liveOut))
		}
		return f.liveExpression(expression.Children[0], mergeBorrowLive(branchLive...), context)
	case syntax.KindMatch:
		if len(expression.Children) < 2 {
			return f.liveChildren(expression.Children, liveOut, context)
		}
		var branchLive []borrowLiveMap
		for _, arm := range expression.Children[1:] {
			if arm != nil && len(arm.Children) > 1 {
				branchLive = append(branchLive, f.liveExpression(arm.Children[len(arm.Children)-1], liveOut, context))
			}
		}
		return f.liveExpression(expression.Children[0], mergeBorrowLive(branchLive...), context)
	case syntax.KindWhile:
		return f.liveWhile(expression, liveOut, context)
	default:
		live := f.liveChildren(expression.Children, liveOut, context)
		if expression.Block != nil {
			live = f.liveBlock(expression.Block, live, context)
		}
		return live
	}
}

func (f *borrowFlowAnalyzer) liveChildren(children []*ast.Expression, liveOut borrowLiveMap, context borrowLiveContext) borrowLiveMap {
	live := cloneBorrowLive(liveOut)
	for index := len(children) - 1; index >= 0; index-- {
		live = f.liveExpression(children[index], live, context)
	}
	return live
}

func (f *borrowFlowAnalyzer) liveWhile(expression *ast.Expression, liveOut borrowLiveMap, outer borrowLiveContext) borrowLiveMap {
	if len(expression.Children) != 2 {
		return f.liveChildren(expression.Children, liveOut, outer)
	}
	header := cloneBorrowLive(liveOut)
	// Each iteration can only add or advance a fact to one of the function's
	// finitely many typed source expressions. This bound follows that lattice
	// instead of using a magic retry count.
	limit := len(f.analyzer.expressionTypes) + 1
	for iteration := 0; iteration < limit; iteration++ {
		loopContext := borrowLiveContext{breakLive: cloneBorrowLive(liveOut), continueLive: cloneBorrowLive(header)}
		body := f.liveExpression(expression.Children[1], header, loopContext)
		conditionOut := mergeBorrowLive(liveOut, body)
		next := f.liveExpression(expression.Children[0], conditionOut, outer)
		if sameBorrowLive(header, next) {
			return next
		}
		header = mergeBorrowLive(header, next)
	}
	return header
}
