package mir

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

// Lower normalizes typed HIR into executable mutable-local MIR.
func Lower(input hir.Package, properties TypeProperties) (Program, error) {
	program := Program{}
	for _, module := range input.Modules {
		for index := range module.Functions {
			function := &module.Functions[index]
			lowerer := newFunctionLowerer(module.Path, function, properties)
			lowered, err := lowerer.lower()
			if err != nil {
				return Program{}, fmt.Errorf("lower %s::%s: %w", module.Path, function.Name, err)
			}
			program.Functions = append(program.Functions, lowered)
		}
	}
	slices.SortFunc(program.Functions, func(left, right Function) int {
		if left.Name < right.Name {
			return -1
		}
		if left.Name > right.Name {
			return 1
		}
		return 0
	})
	return program, nil
}

type bindingInfo struct {
	typ  hir.Type
	span hir.Span
}

type functionLowerer struct {
	module     string
	source     *hir.Function
	properties TypeProperties
	result     Function
	current    BlockID
	bindings   map[hir.SymbolID]bindingInfo
	live       map[hir.SymbolID]bool
}

func newFunctionLowerer(module string, function *hir.Function, properties TypeProperties) *functionLowerer {
	result := Function{
		Name:  module + "::" + function.Name,
		Span:  function.Span,
		Entry: 0,
	}
	return &functionLowerer{
		module:     module,
		source:     function,
		properties: properties,
		result:     result,
		bindings:   make(map[hir.SymbolID]bindingInfo),
		live:       make(map[hir.SymbolID]bool),
	}
}

func (l *functionLowerer) lower() (Function, error) {
	l.newBlock()
	for _, parameter := range l.source.Parameters {
		l.bindings[parameter.Symbol] = bindingInfo{typ: parameter.Type, span: parameter.Span}
		if l.properties.NeedsDrop(parameter.Type) {
			l.live[parameter.Symbol] = true
			l.result.Resources = append(l.result.Resources, ResourceLocal{
				Symbol: parameter.Symbol, Type: parameter.Type, Binding: parameter.Span, Parameter: true,
			})
		}
	}
	for _, local := range l.source.Locals {
		l.bindings[local.Symbol] = bindingInfo{typ: local.Type, span: local.Span}
		if l.properties.NeedsDrop(local.Type) {
			l.result.Resources = append(l.result.Resources, ResourceLocal{
				Symbol: local.Symbol, Type: local.Type, Binding: local.Span,
			})
		}
	}

	terminated, err := l.lowerBlock(l.source.Body)
	if err != nil {
		return Function{}, err
	}
	if !terminated {
		return Function{}, fmt.Errorf("function has no lowered return")
	}
	return l.result, nil
}

func (l *functionLowerer) lowerBlock(block hir.Block) (bool, error) {
	for _, statement := range block.Statements {
		if l.blockTerminated(l.current) {
			break
		}
		switch statement.Kind {
		case "binding":
			if err := l.lowerExpression(statement.Expression); err != nil {
				return false, err
			}
			if binding, found := l.bindings[statement.Binding]; found && l.properties.NeedsDrop(binding.typ) {
				l.appendOperation(Operation{Kind: OperationInitialize, Span: binding.span, Symbol: statement.Binding})
				l.live[statement.Binding] = true
			}
		case "return":
			if err := l.lowerReturn(statement.Expression, expressionSpan(statement.Expression, statement.Span)); err != nil {
				return false, err
			}
			return true, nil
		case "discard", "expression-statement":
			if err := l.lowerExpression(statement.Expression); err != nil {
				return false, err
			}
		case "assignment":
			if err := l.lowerExpression(statement.Expression); err != nil {
				return false, err
			}
		default:
			return false, fmt.Errorf("unsupported HIR statement %q", statement.Kind)
		}
	}
	if l.blockTerminated(l.current) {
		return true, nil
	}
	if block.Tail != nil {
		if err := l.lowerReturn(block.Tail, block.Tail.Span); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (l *functionLowerer) lowerExpression(expression *hir.Expression) error {
	if expression == nil {
		return nil
	}
	switch expression.Kind {
	case "literal", "path":
		return nil
	case "group", "field-expression", "record", "prefix":
		for index := range expression.Children {
			if err := l.lowerExpression(&expression.Children[index]); err != nil {
				return err
			}
		}
		return nil
	case "call":
		return l.lowerCall(expression)
	case "binary":
		for index := range expression.Children {
			if err := l.lowerExpression(&expression.Children[index]); err != nil {
				return err
			}
		}
		if expression.Operator == "/" || expression.Operator == "%" {
			return l.lowerCheckedDivision(expression)
		}
		return nil
	case "propagate":
		return l.lowerPropagate(expression)
	case "block":
		if expression.Block == nil {
			return fmt.Errorf("block expression has no block")
		}
		_, err := l.lowerBlock(*expression.Block)
		return err
	default:
		return fmt.Errorf("unsupported HIR expression %q", expression.Kind)
	}
}

func (l *functionLowerer) lowerCall(expression *hir.Expression) error {
	arguments := expression.Children
	if len(arguments) > 0 {
		arguments = arguments[1:]
	}
	argumentSpans := make([]hir.Span, 0, len(arguments))
	hasNestedSideEffect := false
	for index := range arguments {
		argument := &arguments[index]
		hasNestedSideEffect = hasNestedSideEffect || containsCall(argument)
		if err := l.lowerExpression(argument); err != nil {
			return err
		}
		argumentSpans = append(argumentSpans, argument.Span)
		if l.isLiveResourcePath(argument) {
			l.movePath(argument, "call-argument", true)
		}
	}
	if len(argumentSpans) >= 2 {
		l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
			Expression: expression.Span, Sequence: slices.Clone(argumentSpans), MaterializedBeforeParent: true,
		})
	}
	if hasNestedSideEffect {
		l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "nested-side-effect", RemainingCount: 0})
	}
	l.result.Calls = append(l.result.Calls, CallFact{
		Call: expression.Span, ArgumentOrder: slices.Clone(argumentSpans), NormalSuccessor: true,
	})
	continuation := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorCall, Span: expression.Span, Successors: []BlockID{continuation}, UnwindSuccessor: NoBlock,
	})
	l.current = continuation
	return nil
}

func (l *functionLowerer) lowerCheckedDivision(expression *hir.Expression) error {
	categories := []string{"division-by-zero", "signed-division-overflow"}
	l.appendOperation(Operation{Kind: OperationCheckedDivision, Span: expression.Span, TrapCategories: slices.Clone(categories)})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: expression.Span, UnwindSuccessor: NoBlock, TrapCategories: slices.Clone(categories)})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: expression.Span, Categories: slices.Clone(categories), CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	return nil
}

func (l *functionLowerer) lowerPropagate(expression *hir.Expression) error {
	if len(expression.Children) != 1 {
		return fmt.Errorf("propagate expression has %d children", len(expression.Children))
	}
	operand := &expression.Children[0]
	if err := l.lowerExpression(operand); err != nil {
		return err
	}
	if l.isLiveResourcePath(operand) {
		l.appendOperation(Operation{Kind: OperationMove, Span: operand.Span, Symbol: operand.Symbol, Destination: "result-branch"})
		l.live[operand.Symbol] = false
	}
	l.appendOperation(Operation{Kind: OperationResultSwitch, Span: expression.Span})
	success := l.newBlock()
	errorBlock := l.newBlock()
	l.terminate(Terminator{Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{success, errorBlock}, UnwindSuccessor: NoBlock})

	errorLive := cloneLive(l.live)
	l.current = errorBlock
	errorActions := l.emitCleanup(errorLive)
	l.terminate(Terminator{Kind: TerminatorReturn, Span: expression.Span, UnwindSuccessor: NoBlock})
	question := hir.Span{Source: expression.Span.Source, StartByte: max(expression.Span.StartByte, expression.Span.EndByte-1), EndByte: expression.Span.EndByte}
	l.result.ResultBranches = append(l.result.ResultBranches, ResultBranchFact{
		Question: question, ResultExpression: operand.Span, EvaluationCount: 1,
		SuccessPayload: "Move", ErrorPayload: "Move", ResultTemporaryDeadOn: []string{"success", "error"},
		ErrorCleanupActions: errorActions,
	})
	l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "question", RemainingCount: 0})
	l.current = success
	return nil
}

func (l *functionLowerer) lowerReturn(expression *hir.Expression, exit hir.Span) error {
	if err := l.lowerExpression(expression); err != nil {
		return err
	}
	if l.isLiveResourcePath(expression) {
		l.movePath(expression, "return-value", true)
	}
	actions := l.emitCleanup(l.live)
	l.result.CleanupPaths = append(l.result.CleanupPaths, CleanupPathFact{Exit: exit, ExitKind: "return", Actions: actions})
	if len(actions) > 0 {
		l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
	}
	l.terminate(Terminator{Kind: TerminatorReturn, Span: exit, UnwindSuccessor: NoBlock})
	return nil
}

func (l *functionLowerer) movePath(expression *hir.Expression, destination string, record bool) {
	binding := l.bindings[expression.Symbol]
	l.appendOperation(Operation{Kind: OperationMove, Span: expression.Span, Symbol: expression.Symbol, Destination: destination})
	l.live[expression.Symbol] = false
	if record {
		l.result.Moves = append(l.result.Moves, MoveFact{
			Binding: binding.span, Operand: expression.Span, Destination: destination, CleanupObligationTransferred: true,
		})
	}
}

func (l *functionLowerer) emitCleanup(live map[hir.SymbolID]bool) []CleanupAction {
	actions := make([]CleanupAction, 0)
	for index := len(l.result.Resources) - 1; index >= 0; index-- {
		resource := l.result.Resources[index]
		if !live[resource.Symbol] {
			continue
		}
		l.appendOperation(Operation{Kind: OperationDrop, Span: resource.Binding, Symbol: resource.Symbol, FlagClearedFirst: true})
		live[resource.Symbol] = false
		actions = append(actions, CleanupAction{Kind: "drop", Binding: resource.Binding})
	}
	return actions
}

func (l *functionLowerer) isLiveResourcePath(expression *hir.Expression) bool {
	if expression == nil || expression.Kind != "path" || expression.Symbol == hir.NoSymbol || !l.live[expression.Symbol] {
		return false
	}
	binding, found := l.bindings[expression.Symbol]
	return found && l.properties.NeedsDrop(binding.typ)
}

func (l *functionLowerer) newBlock() BlockID {
	id := BlockID(len(l.result.Blocks))
	l.result.Blocks = append(l.result.Blocks, Block{ID: id, Terminator: Terminator{UnwindSuccessor: NoBlock}})
	return id
}

func (l *functionLowerer) appendOperation(operation Operation) {
	l.result.Blocks[l.current].Operations = append(l.result.Blocks[l.current].Operations, operation)
}

func (l *functionLowerer) terminate(terminator Terminator) {
	l.result.Blocks[l.current].Terminator = terminator
}

func (l *functionLowerer) blockTerminated(id BlockID) bool {
	return l.result.Blocks[id].Terminator.Kind != ""
}

func expressionSpan(expression *hir.Expression, fallback hir.Span) hir.Span {
	if expression == nil {
		return fallback
	}
	return expression.Span
}

func cloneLive(input map[hir.SymbolID]bool) map[hir.SymbolID]bool {
	result := make(map[hir.SymbolID]bool, len(input))
	for symbol, live := range input {
		result[symbol] = live
	}
	return result
}

func containsCall(expression *hir.Expression) bool {
	if expression == nil {
		return false
	}
	if expression.Kind == "call" {
		return true
	}
	for index := range expression.Children {
		if containsCall(&expression.Children[index]) {
			return true
		}
	}
	return false
}
