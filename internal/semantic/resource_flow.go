package semantic

import (
	"maps"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

type resourceUse uint8

const (
	resourceRead resourceUse = iota
	resourceMove
)

type resourceValueState struct {
	state       ResourceState
	declaration hir.Span
	move        *hir.Span
}

type resourceEnvironment map[*symbol]resourceValueState

type resourceBranch struct {
	environment resourceEnvironment
	reachable   bool
}

type resourceLoopEdge struct {
	environment resourceEnvironment
}

type resourceLoopFrame struct {
	header   hir.Span
	baseline resourceEnvironment
	edges    []resourceLoopEdge
}

func (a *analyzer) resourceFlowFunction(module *moduleScope, function *ast.Function) {
	if function.Body == nil {
		return
	}
	environment := make(resourceEnvironment)
	for index := range function.Parameters {
		parameter := &function.Parameters[index]
		value := a.parameter[parameter]
		if value != nil && a.isResource(value.typ) {
			environment[value] = resourceValueState{
				state:       ResourceLive,
				declaration: span(module.file.Path, parameter.NameSpan),
			}
		}
	}
	tailMode := resourceRead
	if value := a.functions[function]; value != nil && a.isResource(value.typ) {
		tailMode = resourceMove
	}
	a.resourceBlock(module, function.Body, environment, false, tailMode)
}

func (a *analyzer) resourceBlock(module *moduleScope, block *ast.Block, environment resourceEnvironment, scoped bool, tailMode resourceUse) bool {
	if block == nil {
		return true
	}
	var existing map[*symbol]struct{}
	if scoped {
		existing = make(map[*symbol]struct{}, len(environment))
		for value := range environment {
			existing[value] = struct{}{}
		}
	}
	reachable := true
	for index := range block.Statements {
		if !reachable {
			break
		}
		statement := &block.Statements[index]
		switch {
		case statement.Binding != nil:
			binding := statement.Binding
			value := a.bindingSymbols[binding]
			mode := resourceRead
			if value != nil && a.isResource(value.typ) {
				mode = resourceMove
			}
			a.resourceExpression(module, binding.Expression, environment, mode)
			if value != nil && a.isResource(value.typ) {
				environment[value] = resourceValueState{
					state:       ResourceLive,
					declaration: span(module.file.Path, binding.NameSpan),
				}
			}
		case statement.Kind == syntax.KindAssignment:
			a.resourceAssignment(module, statement, environment)
		case statement.Kind == syntax.KindWhile:
			a.resourceWhile(module, statement.Expression, environment)
		case statement.Kind == syntax.KindReturn:
			a.resourceExpression(module, statement.Expression, environment, resourceMove)
			reachable = false
		case statement.Kind == syntax.KindBreak || statement.Kind == syntax.KindContinue:
			a.captureResourceLoopEdge(environment)
			reachable = false
		default:
			a.resourceExpression(module, statement.Expression, environment, resourceRead)
		}
	}
	if reachable && block.Tail != nil {
		a.resourceExpression(module, block.Tail, environment, tailMode)
	}
	if scoped {
		for value := range environment {
			if _, found := existing[value]; !found {
				delete(environment, value)
			}
		}
	}
	return reachable
}

func (a *analyzer) resourceAssignment(module *moduleScope, statement *ast.Statement, environment resourceEnvironment) {
	a.resourceExpression(module, statement.Expression, environment, resourceMove)
	if statement.Target == nil {
		return
	}
	target := a.resolved[statement.Target]
	if target == nil || !a.isResource(target.typ) {
		a.resourceExpression(module, statement.Target, environment, resourceRead)
		return
	}
	state, tracked := environment[target]
	if !tracked {
		return
	}
	at := span(module.file.Path, statement.Target.Span)
	a.recordResourceStates(environment, at, "before")
	if state.state != ResourceMaybeLive {
		state.state = ResourceLive
		state.move = nil
		environment[target] = state
	}
	a.recordResourceStates(environment, at, "after")
}

func (a *analyzer) resourceExpression(module *moduleScope, expression *ast.Expression, environment resourceEnvironment, mode resourceUse) bool {
	if expression == nil {
		return true
	}
	at := span(module.file.Path, expression.Span)
	a.recordResourceStates(environment, at, "before")
	reachable := true
	switch expression.Kind {
	case syntax.KindPath:
		a.resourcePath(expression, environment, mode)
	case syntax.KindGroup:
		if len(expression.Children) == 1 {
			reachable = a.resourceExpression(module, expression.Children[0], environment, mode)
		}
	case syntax.KindPrefix:
		childMode := resourceRead
		if expression.Text != "&" && expression.Text != "&mut" && expression.Text != "*" {
			childMode = mode
		}
		if len(expression.Children) == 1 {
			a.resourceExpression(module, expression.Children[0], environment, childMode)
		}
	case syntax.KindCall:
		a.resourceCall(module, expression, environment)
	case syntax.KindRecord:
		a.resourceRecord(module, expression, environment)
	case syntax.KindFieldExpr:
		a.resourceField(module, expression, environment, mode)
	case syntax.KindIf:
		reachable = a.resourceIf(module, expression, environment, mode)
	case syntax.KindWhile:
		a.resourceWhile(module, expression, environment)
	case syntax.KindMatch:
		reachable = a.resourceMatch(module, expression, environment, mode)
	case syntax.KindBlock:
		reachable = a.resourceBlock(module, expression.Block, environment, true, mode)
	case syntax.KindRegion:
		if len(expression.Children) == 2 {
			a.resourceExpression(module, expression.Children[0], environment, resourceRead)
			reachable = a.resourceExpression(module, expression.Children[1], environment, mode)
		}
	default:
		for _, child := range expression.Children {
			a.resourceExpression(module, child, environment, resourceRead)
		}
	}
	a.recordResourceStates(environment, at, "after")
	return reachable
}

func (a *analyzer) resourcePath(expression *ast.Expression, environment resourceEnvironment, mode resourceUse) {
	value := a.resolved[expression]
	state, tracked := environment[value]
	if !tracked {
		return
	}
	use := span(value.source, expression.Span)
	if state.state == ResourceDead {
		related := []RelatedSpan{{Role: "declaration", Span: state.declaration}}
		if state.move != nil {
			related = append(related, RelatedSpan{Role: "move", Span: *state.move})
		}
		a.addDiagnostic("E1101", use, related...)
		return
	}
	if state.state != ResourceLive || mode != resourceMove {
		return
	}
	move := use
	state.state = ResourceDead
	state.move = &move
	environment[value] = state
}

func (a *analyzer) resourceCall(module *moduleScope, expression *ast.Expression, environment resourceEnvironment) {
	if len(expression.Children) == 0 {
		return
	}
	callee := a.resolved[expression.Children[0]]
	for index, argument := range expression.Children[1:] {
		mode := resourceRead
		if callee != nil && callee.function != nil && index < len(callee.function.Parameters) {
			parameter := &callee.function.Parameters[index]
			if value := a.parameter[parameter]; value != nil && a.isResource(value.typ) {
				mode = resourceMove
			}
		} else if isBuiltinVariantPath(expression.Children[0].Text) && a.isResource(a.expressionTypes[argument]) {
			mode = resourceMove
		}
		a.resourceExpression(module, argument, environment, mode)
	}
}

func (a *analyzer) resourceMatch(module *moduleScope, expression *ast.Expression, environment resourceEnvironment, mode resourceUse) bool {
	if len(expression.Children) != 3 {
		return true
	}
	scrutinee := expression.Children[0]
	scrutineeMode := resourceRead
	if a.isResource(a.expressionTypes[scrutinee]) {
		scrutineeMode = resourceMove
	}
	a.resourceExpression(module, scrutinee, environment, scrutineeMode)
	baseline := cloneResourceEnvironment(environment)
	branches := make([]resourceBranch, 0, 2)
	for _, arm := range expression.Children[1:] {
		branch := cloneResourceEnvironment(baseline)
		var binding *symbol
		if arm != nil && arm.Kind == syntax.KindPattern && len(arm.Children) == 2 {
			pattern := arm.Children[0]
			if pattern != nil && pattern.Kind == syntax.KindPattern && len(pattern.Children) == 2 {
				binding = a.resolved[pattern.Children[1]]
				if binding != nil && a.isResource(binding.typ) {
					branch[binding] = resourceValueState{
						state:       ResourceLive,
						declaration: span(binding.source, binding.declaration),
					}
				}
			}
			reachable := a.resourceExpression(module, arm.Children[1], branch, mode)
			if binding != nil {
				delete(branch, binding)
			}
			branches = append(branches, resourceBranch{environment: branch, reachable: reachable})
			continue
		}
		if binding != nil {
			delete(branch, binding)
		}
		branches = append(branches, resourceBranch{environment: branch, reachable: true})
	}
	return a.mergeResourceBranches(span(module.file.Path, expression.Span), environment, baseline, branches)
}

func (a *analyzer) resourceRecord(module *moduleScope, expression *ast.Expression, environment resourceEnvironment) {
	if len(expression.Children) == 0 {
		return
	}
	constructor := a.resolved[expression.Children[0]]
	if constructor == nil {
		return
	}
	info := a.resourceTypes[constructor.typ]
	if info == nil {
		return
	}
	for _, fieldExpression := range expression.Children[1:] {
		if len(fieldExpression.Children) != 1 {
			continue
		}
		mode := resourceRead
		if field, found := info.fields[fieldExpression.Text]; found && a.isResource(field.typ) {
			mode = resourceMove
		}
		a.resourceExpression(module, fieldExpression.Children[0], environment, mode)
	}
}

func (a *analyzer) resourceField(module *moduleScope, expression *ast.Expression, environment resourceEnvironment, mode resourceUse) {
	if len(expression.Children) != 1 {
		return
	}
	receiver := expression.Children[0]
	owner := a.resolved[receiver]
	if owner == nil {
		a.resourceExpression(module, receiver, environment, resourceRead)
		return
	}
	ownerType := owner.typ
	if element, _, borrow := hir.BorrowElement(ownerType); borrow {
		ownerType = element
	}
	info := a.resourceTypes[ownerType]
	field, found := resourceField{}, false
	if info != nil {
		field, found = info.fields[expression.Text]
	}
	if found && mode == resourceMove && a.isResource(field.typ) {
		a.addDiagnostic("E1103", span(module.file.Path, expression.Span),
			RelatedSpan{Role: "field", Span: field.declaration},
			RelatedSpan{Role: "owner", Span: span(owner.source, owner.declaration)},
		)
	}
	a.resourceExpression(module, receiver, environment, resourceRead)
}

func (a *analyzer) resourceIf(module *moduleScope, expression *ast.Expression, environment resourceEnvironment, mode resourceUse) bool {
	if len(expression.Children) < 2 {
		return true
	}
	a.resourceExpression(module, expression.Children[0], environment, resourceRead)
	baseline := cloneResourceEnvironment(environment)
	thenEnvironment := cloneResourceEnvironment(baseline)
	thenReachable := a.resourceExpression(module, expression.Children[1], thenEnvironment, mode)
	elseEnvironment := cloneResourceEnvironment(baseline)
	elseReachable := true
	if len(expression.Children) > 2 {
		elseReachable = a.resourceExpression(module, expression.Children[2], elseEnvironment, mode)
	}
	return a.mergeResourceBranches(span(module.file.Path, expression.Span), environment, baseline, []resourceBranch{
		{environment: thenEnvironment, reachable: thenReachable},
		{environment: elseEnvironment, reachable: elseReachable},
	})
}

func (a *analyzer) resourceWhile(module *moduleScope, expression *ast.Expression, environment resourceEnvironment) {
	if expression == nil || len(expression.Children) != 2 {
		return
	}
	condition := expression.Children[0]
	header := hir.Span{Source: module.file.Path, StartByte: expression.Span.StartByte, EndByte: condition.Span.EndByte}
	a.recordResourceStates(environment, header, "before")
	a.resourceExpression(module, condition, environment, resourceRead)
	baseline := cloneResourceEnvironment(environment)
	bodyEnvironment := cloneResourceEnvironment(baseline)
	a.resourceLoops = append(a.resourceLoops, resourceLoopFrame{header: header, baseline: cloneResourceEnvironment(baseline)})
	bodyReachable := a.resourceExpression(module, expression.Children[1], bodyEnvironment, resourceRead)
	frame := a.resourceLoops[len(a.resourceLoops)-1]
	a.resourceLoops = a.resourceLoops[:len(a.resourceLoops)-1]
	if bodyReachable {
		a.validateResourceLoopEdge(span(module.file.Path, expression.Span), frame, bodyEnvironment)
	}
	for _, edge := range frame.edges {
		a.validateResourceLoopEdge(span(module.file.Path, expression.Span), frame, edge.environment)
	}
}

func (a *analyzer) captureResourceLoopEdge(environment resourceEnvironment) {
	if len(a.resourceLoops) == 0 {
		return
	}
	index := len(a.resourceLoops) - 1
	a.resourceLoops[index].edges = append(a.resourceLoops[index].edges, resourceLoopEdge{environment: cloneResourceEnvironment(environment)})
}

func (a *analyzer) validateResourceLoopEdge(loopSpan hir.Span, frame resourceLoopFrame, actual resourceEnvironment) {
	for value, before := range frame.baseline {
		after, found := actual[value]
		if !found {
			after = before
		}
		if before.state == after.state {
			continue
		}
		related := []RelatedSpan{
			{Role: "declaration", Span: before.declaration},
			{Role: "loop-header", Span: frame.header},
		}
		if after.move != nil {
			related = append(related, RelatedSpan{Role: "move", Span: *after.move})
		}
		a.addDiagnostic("E1105", loopSpan, related...)
	}
}

func (a *analyzer) mergeResourceBranches(at hir.Span, environment, baseline resourceEnvironment, branches []resourceBranch) bool {
	reachable := make([]resourceEnvironment, 0, len(branches))
	for _, branch := range branches {
		if branch.reachable {
			reachable = append(reachable, branch.environment)
		}
	}
	if len(reachable) == 0 {
		return false
	}
	for value, before := range baseline {
		joined, found := reachable[0][value]
		if !found {
			joined = before
		}
		for _, branch := range reachable[1:] {
			actual, found := branch[value]
			if !found {
				actual = before
			}
			joined = joinResourceState(joined, actual)
		}
		environment[value] = joined
		if joined.state != ResourceMaybeLive || before.state == ResourceMaybeLive {
			continue
		}
		related := []RelatedSpan{{Role: "declaration", Span: before.declaration}}
		if joined.move != nil {
			related = append(related, RelatedSpan{Role: "move", Span: *joined.move})
		}
		a.addDiagnostic("E1104", at, related...)
	}
	return true
}

func (a *analyzer) recordResourceStates(environment resourceEnvironment, at hir.Span, observation string) {
	for _, value := range environment {
		fact := ResourceStateFact{
			Binding:     value.declaration,
			At:          at,
			Observation: observation,
			State:       value.state,
		}
		switch value.state {
		case ResourceLive:
			count := 1
			fact.CleanupObligations = &count
		case ResourceDead:
			count := 0
			fact.CleanupObligations = &count
		}
		a.resourceStates = append(a.resourceStates, fact)
	}
}

func (a *analyzer) isResource(value hir.Type) bool {
	if _, found := a.resourceTypes[value]; found {
		return true
	}
	if element, option := hir.OptionElement(value); option {
		return a.isResource(element)
	}
	success, failure, result := hir.ResultElements(value)
	return result && (a.isResource(success) || a.isResource(failure))
}

func cloneResourceEnvironment(source resourceEnvironment) resourceEnvironment {
	return maps.Clone(source)
}

func joinResourceState(left, right resourceValueState) resourceValueState {
	if left.state == right.state {
		if left.move == nil {
			left.move = right.move
		}
		return left
	}
	result := left
	result.state = ResourceMaybeLive
	if result.move == nil {
		result.move = right.move
	}
	return result
}
