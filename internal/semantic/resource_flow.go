package semantic

import (
	"maps"
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

type resourceUse uint8

const (
	resourceRead resourceUse = iota
	resourceMove
)

type dropEffectRequirement struct {
	owner   *symbol
	effects []hir.Effect
}

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
	back        bool
}

type resourceLoopFrame struct {
	header     hir.Span
	baseline   resourceEnvironment
	edges      []resourceLoopEdge
	scopeDepth int
}

func (a *analyzer) resourceFlowFunction(module *moduleScope, function *ast.Function) {
	if function.Body == nil {
		return
	}
	environment := make(resourceEnvironment)
	a.beginResourceEffects(a.functions[function], nil)
	defer a.endResourceEffects()
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
	reachable := a.resourceBlock(module, function.Body, environment, false, tailMode)
	if reachable {
		a.checkResourceEnvironmentEffects(environment, span(module.file.Path, function.Body.Span), nil)
	}
}

func (a *analyzer) resourceFlowDropFunction(module *moduleScope, function *ast.Function) {
	if function.Body == nil || len(function.Parameters) != 1 {
		return
	}
	parameter := &function.Parameters[0]
	receiver := a.parameter[parameter]
	if receiver == nil || !a.isResource(receiver.typ) {
		return
	}
	environment := resourceEnvironment{
		receiver: {
			state:       ResourceLive,
			declaration: span(module.file.Path, parameter.NameSpan),
		},
	}
	a.beginResourceEffects(a.functions[function], receiver)
	defer a.endResourceEffects()
	reachable := a.resourceBlock(module, function.Body, environment, false, resourceRead)
	if reachable {
		a.checkResourceEnvironmentEffects(environment, span(module.file.Path, function.Body.Span), nil)
	}
	state := environment[receiver]
	if state.state == ResourceLive {
		return
	}
	primary := span(module.file.Path, function.Body.Span)
	related := []RelatedSpan{{Role: "declaration", Span: state.declaration}}
	if state.move != nil {
		primary = *state.move
		related = append(related, RelatedSpan{Role: "move", Span: *state.move})
	}
	a.addDiagnostic("E1102", primary, related...)
}

func (a *analyzer) resourceBlock(module *moduleScope, block *ast.Block, environment resourceEnvironment, scoped bool, tailMode resourceUse) bool {
	if a.canceled() {
		return false
	}
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
	a.resourceScopes = append(a.resourceScopes, make(map[*symbol]struct{}))
	scopeIndex := len(a.resourceScopes) - 1
	defer func() { a.resourceScopes = a.resourceScopes[:scopeIndex] }()
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
			reachable = a.resourceExpression(module, binding.Expression, environment, mode)
			if !reachable {
				continue
			}
			if value != nil && a.isResource(value.typ) {
				environment[value] = resourceValueState{
					state:       ResourceLive,
					declaration: span(module.file.Path, binding.NameSpan),
				}
				a.resourceScopes[scopeIndex][value] = struct{}{}
			}
		case statement.Kind == syntax.KindAssignment:
			a.resourceAssignment(module, statement, environment)
		case statement.Kind == syntax.KindWhile:
			a.resourceWhile(module, statement.Expression, environment)
		case statement.Kind == syntax.KindFor:
			a.resourceFor(module, statement.For, environment)
		case statement.Kind == syntax.KindReturn:
			if a.resourceExpression(module, statement.Expression, environment, resourceMove) {
				a.checkResourceEnvironmentEffects(environment, span(module.file.Path, statement.Span), nil)
			}
			reachable = false
		case statement.Kind == syntax.KindBurn:
			a.resourceExpression(module, statement.Expression, environment, resourceMove)
		case statement.Kind == syntax.KindBreak || statement.Kind == syntax.KindContinue:
			a.checkResourceLoopExitEffects(environment, span(module.file.Path, statement.Span))
			a.captureResourceLoopEdge(environment, statement.Kind == syntax.KindContinue)
			reachable = false
		default:
			reachable = a.resourceExpression(module, statement.Expression, environment, resourceRead)
			if reachable && (statement.Kind == syntax.KindExprStmt || statement.Kind == syntax.KindDiscard) {
				a.checkDiscardedResourceCleanup(module, statement.Expression)
			}
		}
	}
	if reachable && block.Tail != nil {
		reachable = a.resourceExpression(module, block.Tail, environment, tailMode)
	}
	if scoped {
		if reachable {
			a.checkResourceEnvironmentEffects(environment, span(module.file.Path, block.Span), a.resourceScopes[scopeIndex])
		}
		for value := range environment {
			if _, found := existing[value]; !found {
				delete(environment, value)
			}
		}
	}
	return reachable
}

func (a *analyzer) resourceFor(module *moduleScope, loop *ast.ForLoop, environment resourceEnvironment) {
	if loop == nil {
		return
	}
	existing := make(map[*symbol]struct{}, len(environment))
	for value := range environment {
		existing[value] = struct{}{}
	}
	a.resourceScopes = append(a.resourceScopes, make(map[*symbol]struct{}))
	scopeIndex := len(a.resourceScopes) - 1
	defer func() { a.resourceScopes = a.resourceScopes[:scopeIndex] }()
	if initializer := loop.Initializer; initializer != nil {
		if initializer.Binding != nil {
			binding := initializer.Binding
			value := a.bindingSymbols[binding]
			mode := resourceRead
			if value != nil && a.isResource(value.typ) {
				mode = resourceMove
			}
			a.resourceExpression(module, binding.Expression, environment, mode)
			if value != nil && a.isResource(value.typ) {
				environment[value] = resourceValueState{state: ResourceLive, declaration: span(module.file.Path, binding.NameSpan)}
				a.resourceScopes[scopeIndex][value] = struct{}{}
			}
		} else if initializer.Kind == syntax.KindAssignment {
			a.resourceAssignment(module, initializer, environment)
		} else {
			a.resourceExpression(module, initializer.Expression, environment, resourceRead)
		}
	}
	a.resourceWhile(module, forAsWhile(loop), environment)
	a.checkResourceEnvironmentEffects(environment, span(module.file.Path, loop.Span), a.resourceScopes[scopeIndex])
	for value := range environment {
		if _, found := existing[value]; !found {
			delete(environment, value)
		}
	}
}

func (a *analyzer) resourceAssignment(module *moduleScope, statement *ast.Statement, environment resourceEnvironment) {
	if indexedAssignmentTarget(statement.Target) {
		a.resourceExpression(module, statement.Target, environment, resourceRead)
		a.resourceExpression(module, statement.Expression, environment, resourceRead)
		return
	}
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
	if state.state == ResourceLive {
		a.checkResourceValueEffects(target, at)
	}
	if state.state != ResourceMaybeLive {
		state.state = ResourceLive
		state.move = nil
		environment[target] = state
	}
	a.recordResourceStates(environment, at, "after")
}

func (a *analyzer) resourceExpression(module *moduleScope, expression *ast.Expression, environment resourceEnvironment, mode resourceUse) bool {
	if a.canceled() {
		return false
	}
	if expression == nil {
		return true
	}
	at := span(module.file.Path, expression.Span)
	a.recordResourceStates(environment, at, "before")
	reachable := true
	switch expression.Kind {
	case syntax.KindPath:
		a.resourcePath(expression, environment, mode)
	case syntax.KindGroup, syntax.KindUnsafe:
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
		reachable = a.resourceCall(module, expression, environment)
	case syntax.KindPropagate:
		if len(expression.Children) == 1 {
			reachable = a.resourceExpression(module, expression.Children[0], environment, resourceMove)
			if reachable {
				// The error alternative returns before any subsequent success-path
				// transfer. The operand is consumed on both alternatives.
				a.checkResourceEnvironmentEffects(environment, at, nil)
			}
		}
	case ast.PrivateStoragePut:
		if len(expression.Children) == 2 {
			a.resourceExpression(module, expression.Children[0], environment, resourceRead)
			a.resourceExpression(module, expression.Children[1], environment, resourceMove)
		}
	case syntax.KindRecord:
		reachable = a.resourceRecord(module, expression, environment)
	case syntax.KindFieldExpr:
		if expression.Text == "is_ok" && len(expression.Children) == 1 && isResultType(a.expressionTypes[expression.Children[0]]) {
			scrutinee := expression.Children[0]
			scrutineeMode := resourceRead
			if a.isResource(a.expressionTypes[scrutinee]) {
				scrutineeMode = resourceMove
			}
			a.resourceExpression(module, scrutinee, environment, scrutineeMode)
			if scrutineeMode == resourceMove {
				a.checkResourceValueEffects(&symbol{typ: a.expressionTypes[scrutinee]}, at)
			}
		} else {
			a.resourceField(module, expression, environment, mode)
		}
	case syntax.KindIf:
		reachable = a.resourceIf(module, expression, environment, mode)
	case syntax.KindBinary:
		if len(expression.Children) == 2 && (expression.Text == "&&" || expression.Text == "||") {
			reachable = a.resourceExpression(module, expression.Children[0], environment, resourceRead)
			if reachable {
				baseline := cloneResourceEnvironment(environment)
				right := cloneResourceEnvironment(baseline)
				rightReachable := a.resourceExpression(module, expression.Children[1], right, resourceRead)
				reachable = a.mergeResourceBranches(at, environment, baseline, []resourceBranch{
					{environment: baseline, reachable: true},
					{environment: right, reachable: rightReachable},
				})
			}
		} else {
			for _, child := range expression.Children {
				a.resourceExpression(module, child, environment, resourceRead)
			}
		}
	case syntax.KindWhile:
		a.resourceWhile(module, expression, environment)
	case syntax.KindSwitch:
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

func (a *analyzer) resourceCall(module *moduleScope, expression *ast.Expression, environment resourceEnvironment) bool {
	if len(expression.Children) == 0 {
		return true
	}
	base := len(a.resourcePending)
	defer func() { a.resourcePending = a.resourcePending[:base] }()
	callee := a.resolved[expression.Children[0]]
	parameterOffset := 0
	if method := a.methodCalls[expression]; method != nil {
		callee = method
		parameterOffset = 1
		if field := expression.Children[0]; field != nil && len(field.Children) == 1 {
			mode := resourceRead
			if len(callee.function.Parameters) > 0 {
				parameter := &callee.function.Parameters[0]
				if value := a.parameter[parameter]; value != nil && a.isResource(value.typ) {
					mode = resourceMove
				}
			}
			if !a.resourceExpression(module, field.Children[0], environment, mode) {
				return false
			}
			a.retainResourcePending(a.expressionTypes[field.Children[0]], mode)
		}
	}
	for index, argument := range expression.Children[1:] {
		mode := resourceRead
		parameterIndex := index + parameterOffset
		if callee != nil && callee.function != nil && parameterIndex < len(callee.function.Parameters) {
			parameter := &callee.function.Parameters[parameterIndex]
			if value := a.parameter[parameter]; value != nil && a.isResource(value.typ) {
				mode = resourceMove
			}
		} else if isBuiltinVariantPath(expression.Children[0].Text) && a.isResource(a.expressionTypes[argument]) {
			mode = resourceMove
		}
		if !a.resourceExpression(module, argument, environment, mode) {
			return false
		}
		a.retainResourcePending(a.expressionTypes[argument], mode)
	}
	return true
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
			// A match payload lives until its arm exits. Loop exits inside the
			// arm must include it, while exits from a nested loop retain it.
			patternScope := len(a.resourceScopes) - 1
			if binding != nil && a.isResource(binding.typ) && patternScope >= 0 {
				a.resourceScopes[patternScope][binding] = struct{}{}
			}
			reachable := a.resourceExpression(module, arm.Children[1], branch, mode)
			if binding != nil {
				if patternScope >= 0 {
					delete(a.resourceScopes[patternScope], binding)
				}
				if reachable {
					a.checkResourceEnvironmentEffects(branch, span(module.file.Path, arm.Span), map[*symbol]struct{}{binding: {}})
				}
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

func (a *analyzer) resourceRecord(module *moduleScope, expression *ast.Expression, environment resourceEnvironment) bool {
	if len(expression.Children) == 0 {
		return true
	}
	constructor := a.resolved[expression.Children[0]]
	if constructor == nil {
		return true
	}
	info := a.resourceTypes[constructor.typ]
	if info == nil {
		if structure := a.structTypes[constructor.typ]; structure != nil {
			for _, fieldExpression := range expression.Children[1:] {
				if len(fieldExpression.Children) == 1 {
					if !a.resourceExpression(module, fieldExpression.Children[0], environment, resourceRead) {
						return false
					}
				}
			}
		}
		return true
	}
	base := len(a.resourcePending)
	defer func() { a.resourcePending = a.resourcePending[:base] }()
	for _, fieldExpression := range expression.Children[1:] {
		if len(fieldExpression.Children) != 1 {
			continue
		}
		mode := resourceRead
		if field, found := info.fields[fieldExpression.Text]; found && a.isResource(field.typ) {
			mode = resourceMove
		}
		if !a.resourceExpression(module, fieldExpression.Children[0], environment, mode) {
			return false
		}
		a.retainResourcePending(a.expressionTypes[fieldExpression.Children[0]], mode)
	}
	return true
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
	} else if structure := a.structTypes[ownerType]; structure != nil {
		field, found = structure.fields[expression.Text]
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
	header := sourceSpan(module.file.Path, expression.Span.StartByte, condition.Span.EndByte)
	a.recordResourceStates(environment, header, "before")
	a.resourceExpression(module, condition, environment, resourceRead)
	baseline := cloneResourceEnvironment(environment)
	bodyEnvironment := cloneResourceEnvironment(baseline)
	a.resourceLoops = append(a.resourceLoops, resourceLoopFrame{
		header: header, baseline: cloneResourceEnvironment(baseline), scopeDepth: len(a.resourceScopes),
	})
	bodyReachable := a.resourceExpression(module, expression.Children[1], bodyEnvironment, resourceRead)
	frame := a.resourceLoops[len(a.resourceLoops)-1]
	a.resourceLoops = a.resourceLoops[:len(a.resourceLoops)-1]
	if bodyReachable {
		a.validateResourceLoopEdge(span(module.file.Path, expression.Span), frame, bodyEnvironment)
		a.checkResourceLoopCondition(module, condition, frame, bodyEnvironment)
	}
	for _, edge := range frame.edges {
		a.validateResourceLoopEdge(span(module.file.Path, expression.Span), frame, edge.environment)
		if edge.back {
			a.checkResourceLoopCondition(module, condition, frame, edge.environment)
		}
	}
}

func (a *analyzer) captureResourceLoopEdge(environment resourceEnvironment, back bool) {
	if len(a.resourceLoops) == 0 {
		return
	}
	index := len(a.resourceLoops) - 1
	a.resourceLoops[index].edges = append(a.resourceLoops[index].edges, resourceLoopEdge{environment: cloneResourceEnvironment(environment), back: back})
}

// An unchanged post-condition state can still contain a consumed condition
// operand. Check its next evaluation only on re-entry edges, with exited locals
// removed. A mismatched edge already has E1105 and is not a valid re-entry.
func (a *analyzer) checkResourceLoopCondition(module *moduleScope, condition *ast.Expression, frame resourceLoopFrame, actual resourceEnvironment) {
	environment := cloneResourceEnvironment(frame.baseline)
	for value, before := range frame.baseline {
		after, found := actual[value]
		if !found {
			after = before
		}
		if after.state != before.state {
			return
		}
		environment[value] = after
	}
	a.resourceExpression(module, condition, environment, resourceRead)
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

func (a *analyzer) beginResourceEffects(caller, exempt *symbol) {
	a.resourceCaller = caller
	a.resourceCleanupExempt = exempt
	a.resourceScopes = nil
	a.resourcePending = nil
}

func (a *analyzer) endResourceEffects() {
	a.resourceCaller = nil
	a.resourceCleanupExempt = nil
	a.resourceScopes = nil
	a.resourcePending = nil
}

func (a *analyzer) checkResourceLoopExitEffects(environment resourceEnvironment, at hir.Span) {
	if len(a.resourceLoops) == 0 {
		return
	}
	frame := a.resourceLoops[len(a.resourceLoops)-1]
	values := make(map[*symbol]struct{})
	for index := frame.scopeDepth; index < len(a.resourceScopes); index++ {
		for value := range a.resourceScopes[index] {
			values[value] = struct{}{}
		}
	}
	a.checkResourceCleanupEffects(environment, at, values, frame.scopeDepth+1)
}

func (a *analyzer) checkResourceEnvironmentEffects(environment resourceEnvironment, at hir.Span, only map[*symbol]struct{}) {
	pendingScope := -1
	if only == nil {
		pendingScope = 0
	}
	a.checkResourceCleanupEffects(environment, at, only, pendingScope)
}

func (a *analyzer) checkResourceCleanupEffects(environment resourceEnvironment, at hir.Span, only map[*symbol]struct{}, pendingScope int) {
	values := make([]*symbol, 0, len(environment))
	for value, state := range environment {
		if state.state != ResourceLive || value == a.resourceCleanupExempt {
			continue
		}
		if only != nil {
			if _, selected := only[value]; !selected {
				continue
			}
		}
		values = append(values, value)
	}
	slices.SortFunc(values, func(left, right *symbol) int {
		return compareSpan(span(left.source, left.declaration), span(right.source, right.declaration))
	})
	seen := make(map[*symbol]struct{})
	for _, value := range values {
		a.checkResourceValueEffectsOnce(value, at, seen)
	}
	a.checkResourcePendingEffects(at, pendingScope, seen)
}

func (a *analyzer) checkResourceValueEffects(value *symbol, at hir.Span) {
	a.checkResourceValueEffectsOnce(value, at, make(map[*symbol]struct{}))
}

func (a *analyzer) checkResourceValueEffectsOnce(value *symbol, at hir.Span, seen map[*symbol]struct{}) {
	if value == nil {
		return
	}
	blockers := a.nonDroppableResourceOwners(value.typ, make(map[hir.Type]struct{}))
	if len(blockers) > 0 {
		for _, owner := range blockers {
			if owner == nil {
				continue
			}
			if _, duplicate := seen[owner]; duplicate {
				continue
			}
			seen[owner] = struct{}{}
			_, declaration := a.resourceAllowsImplicitDrop(owner)
			a.addDiagnostic("E1108", at, RelatedSpan{
				Role: "ability-declaration", Span: span(owner.source, declaration),
			})
		}
		return
	}

	caller := a.resourceCaller
	if caller == nil || !caller.effectsValid {
		return
	}
	for _, requirement := range a.requiredDropEffects(value.typ) {
		owner := requirement.owner
		if owner == nil || !owner.effectsValid || len(hir.MissingEffects(caller.effects, requirement.effects)) == 0 {
			continue
		}
		if _, duplicate := seen[owner]; duplicate {
			continue
		}
		seen[owner] = struct{}{}
		related := owner.declaration
		if owner.effectSpan.EndByte > owner.effectSpan.StartByte {
			related = owner.effectSpan
		}
		a.addDiagnostic("E7102", at, RelatedSpan{
			Role: "effect-declaration", Span: span(owner.source, related),
		})
	}
}

func (a *analyzer) requiredDropEffects(value hir.Type) []dropEffectRequirement {
	var result []dropEffectRequirement
	seen := make(map[hir.Type]struct{})
	var visit func(hir.Type)
	visit = func(current hir.Type) {
		if _, duplicate := seen[current]; duplicate {
			return
		}
		seen[current] = struct{}{}
		if info := a.resourceTypes[current]; info != nil && info.symbol != nil {
			result = append(result, dropEffectRequirement{owner: info.symbol, effects: slices.Clone(info.symbol.effects)})
			// Implicit field destruction follows the owner's user cleanup.
			// Moving the complete root transfers both obligations together.
			for _, field := range info.orderedFields {
				visit(field.typ)
			}
			return
		}
		if element, option := hir.OptionElement(current); option {
			visit(element)
			return
		}
		if success, failure, tagged := hir.ResultElements(current); tagged {
			visit(success)
			visit(failure)
		}
	}
	visit(value)
	return result
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

// Unbound expression values still execute cleanup. The must-use companion
// owns linear temporary diagnostics in ordinary functions; cleanup bodies do
// not run that pass, so their linear temporaries are checked here as well.
func (a *analyzer) checkDiscardedResourceCleanup(module *moduleScope, expression *ast.Expression) {
	if expression == nil || resourcePlaceExpression(expression) {
		return
	}
	value := a.expressionTypes[expression]
	if !a.isResource(value) {
		return
	}
	if len(a.nonDroppableResourceOwners(value, make(map[hir.Type]struct{}))) != 0 {
		if a.resourceCleanupExempt != nil {
			a.rejectDiscardedLinearResource(module, expression)
		}
		return
	}
	a.checkResourceValueEffects(&symbol{typ: value}, span(module.file.Path, expression.Span))
}
