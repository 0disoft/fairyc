package mir

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

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
	nextLocal  LocalID
}

func newFunctionLowerer(module string, function *hir.Function, properties TypeProperties) *functionLowerer {
	result := Function{
		Symbol:     function.Symbol,
		Name:       module + "::" + function.Name,
		Span:       function.Span,
		ResultType: function.ResultType,
		Entry:      0,
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
	maximumSymbol := hir.SymbolID(0)
	for _, parameter := range l.source.Parameters {
		maximumSymbol = max(maximumSymbol, parameter.Symbol)
		l.result.Parameters = append(l.result.Parameters, LocalID(parameter.Symbol))
		l.bindings[parameter.Symbol] = bindingInfo{typ: parameter.Type, span: parameter.Span}
		if l.properties.NeedsDrop(parameter.Type) {
			l.live[parameter.Symbol] = true
			l.result.Resources = append(l.result.Resources, ResourceLocal{
				Symbol: parameter.Symbol, Type: parameter.Type, Binding: parameter.Span, Parameter: true,
			})
		}
	}
	for _, local := range l.source.Locals {
		maximumSymbol = max(maximumSymbol, local.Symbol)
		l.bindings[local.Symbol] = bindingInfo{typ: local.Type, span: local.Span}
		if l.properties.NeedsDrop(local.Type) {
			l.result.Resources = append(l.result.Resources, ResourceLocal{
				Symbol: local.Symbol, Type: local.Type, Binding: local.Span,
			})
		}
	}
	l.nextLocal = LocalID(maximumSymbol + 1)

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
	terminated, err := l.lowerStatements(block.Statements)
	if err != nil || terminated {
		return terminated, err
	}
	if block.Tail != nil {
		if err := l.lowerReturn(block.Tail, block.Tail.Span); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (l *functionLowerer) lowerStatements(statements []hir.Statement) (bool, error) {
	for _, statement := range statements {
		if l.blockTerminated(l.current) {
			return true, nil
		}
		switch statement.Kind {
		case "binding":
			value, err := l.lowerExpression(statement.Expression)
			if err != nil {
				return false, err
			}
			if value.Kind != OperandInvalid {
				l.appendOperation(Operation{Kind: OperationAssign, Span: statement.Span, DestinationLocal: LocalID(statement.Binding), Operands: []Operand{value}})
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
			if _, err := l.lowerExpression(statement.Expression); err != nil {
				return false, err
			}
		case "assignment":
			value, err := l.lowerExpression(statement.Expression)
			if err != nil {
				return false, err
			}
			if statement.Target != nil && statement.Target.Kind == "path" && value.Kind != OperandInvalid {
				l.appendOperation(Operation{Kind: OperationAssign, Span: statement.Span, DestinationLocal: LocalID(statement.Target.Symbol), Operands: []Operand{value}})
			}
		case "while":
			if _, err := l.lowerWhile(statement.Expression); err != nil {
				return false, err
			}
		default:
			return false, fmt.Errorf("unsupported HIR statement %q", statement.Kind)
		}
	}
	return l.blockTerminated(l.current), nil
}

func (l *functionLowerer) lowerNestedBlock(block hir.Block) (Operand, bool, error) {
	terminated, err := l.lowerStatements(block.Statements)
	if err != nil || terminated {
		return Operand{}, terminated, err
	}
	if block.Tail == nil {
		return Operand{}, false, nil
	}
	value, err := l.lowerExpression(block.Tail)
	return value, l.blockTerminated(l.current), err
}

func (l *functionLowerer) lowerExpression(expression *hir.Expression) (Operand, error) {
	if expression == nil {
		return Operand{}, nil
	}
	switch expression.Kind {
	case "literal":
		if expression.Type == hir.TypeBool {
			switch expression.Value {
			case "true":
				return Operand{Kind: OperandI32, I32: 1}, nil
			case "false":
				return Operand{Kind: OperandI32, I32: 0}, nil
			default:
				return Operand{}, fmt.Errorf("invalid bool literal %q", expression.Value)
			}
		}
		value, err := parseI32(expression.Value)
		if err != nil {
			return Operand{}, err
		}
		return Operand{Kind: OperandI32, I32: value}, nil
	case "path":
		return Operand{Kind: OperandLocal, Local: LocalID(expression.Symbol)}, nil
	case "group":
		if len(expression.Children) != 1 {
			return Operand{}, fmt.Errorf("group expression has %d children", len(expression.Children))
		}
		return l.lowerExpression(&expression.Children[0])
	case "field-expression", "record":
		for index := range expression.Children {
			if _, err := l.lowerExpression(&expression.Children[index]); err != nil {
				return Operand{}, err
			}
		}
		return Operand{}, nil
	case "prefix":
		if len(expression.Children) != 1 {
			return Operand{}, fmt.Errorf("prefix expression has %d children", len(expression.Children))
		}
		operand, err := l.lowerExpression(&expression.Children[0])
		if err != nil {
			return Operand{}, err
		}
		destination := l.newTemporary()
		l.appendOperation(Operation{Kind: OperationUnary, Span: expression.Span, DestinationLocal: destination, Operands: []Operand{operand}, Operator: expression.Operator})
		return Operand{Kind: OperandLocal, Local: destination}, nil
	case "call":
		return l.lowerCall(expression)
	case "result-variant":
		return l.lowerResultVariant(expression)
	case "binary":
		if len(expression.Children) != 2 {
			return Operand{}, fmt.Errorf("binary expression has %d children", len(expression.Children))
		}
		if expression.Operator == "&&" || expression.Operator == "||" {
			return l.lowerShortCircuit(expression)
		}
		left, err := l.lowerExpression(&expression.Children[0])
		if err != nil {
			return Operand{}, err
		}
		right, err := l.lowerExpression(&expression.Children[1])
		if err != nil {
			return Operand{}, err
		}
		if expression.Operator == "/" || expression.Operator == "%" {
			if err := l.lowerCheckedDivision(expression, left, right); err != nil {
				return Operand{}, err
			}
		}
		destination := l.newTemporary()
		l.appendOperation(Operation{Kind: OperationBinary, Span: expression.Span, DestinationLocal: destination, Operands: []Operand{left, right}, Operator: expression.Operator})
		return Operand{Kind: OperandLocal, Local: destination}, nil
	case "propagate":
		return l.lowerPropagate(expression)
	case "if":
		return l.lowerIf(expression)
	case "match":
		return l.lowerMatch(expression)
	case "while":
		return l.lowerWhile(expression)
	case "block":
		if expression.Block == nil {
			return Operand{}, fmt.Errorf("block expression has no block")
		}
		value, _, err := l.lowerNestedBlock(*expression.Block)
		return value, err
	default:
		return Operand{}, fmt.Errorf("unsupported HIR expression %q", expression.Kind)
	}
}

func (l *functionLowerer) lowerResultVariant(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || (expression.Variant != "ok" && expression.Variant != "err") {
		return Operand{}, fmt.Errorf("malformed Result variant %q with %d payloads", expression.Variant, len(expression.Children))
	}
	payload, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if payload.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("Result::%s payload has no value", expression.Variant)
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationResultConstruct, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{payload}, Variant: expression.Variant,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerMatch(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || len(expression.Arms) != 2 {
		return Operand{}, fmt.Errorf("Result match has %d scrutinees and %d arms", len(expression.Children), len(expression.Arms))
	}
	scrutinee, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if scrutinee.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("Result match scrutinee has no value")
	}
	branchLive := cloneLive(l.live)
	okBlock := l.newBlock()
	errBlock := l.newBlock()
	joinBlock := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Children[0].Span,
		Successors: []BlockID{okBlock, errBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: "result", SwitchOperands: []Operand{scrutinee},
	})

	destination := NoLocal
	if expression.Type != hir.TypeVoid {
		destination = l.newTemporary()
	}
	blocks := map[string]BlockID{"ok": okBlock, "err": errBlock}
	continuing := 0
	var mergedLive map[hir.SymbolID]bool
	seen := make(map[string]struct{}, 2)
	for index := range expression.Arms {
		arm := &expression.Arms[index]
		block, found := blocks[arm.Variant]
		if !found {
			return Operand{}, fmt.Errorf("unsupported Result match variant %q", arm.Variant)
		}
		if _, duplicate := seen[arm.Variant]; duplicate {
			return Operand{}, fmt.Errorf("duplicate Result match variant %q", arm.Variant)
		}
		seen[arm.Variant] = struct{}{}
		l.current = block
		l.live = cloneLive(branchLive)
		l.appendOperation(Operation{
			Kind: OperationResultPayload, Span: arm.BindingSpan, DestinationLocal: LocalID(arm.Binding),
			Operands: []Operand{scrutinee}, Variant: arm.Variant,
		})
		armValue, lowerErr := l.lowerExpression(&arm.Expression)
		if lowerErr != nil {
			return Operand{}, lowerErr
		}
		if l.blockTerminated(l.current) {
			continue
		}
		if destination != NoLocal {
			if armValue.Kind == OperandInvalid {
				return Operand{}, fmt.Errorf("value-producing Result match arm %q has no value", arm.Variant)
			}
			l.appendOperation(Operation{Kind: OperationAssign, Span: arm.Expression.Span, DestinationLocal: destination, Operands: []Operand{armValue}})
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: arm.Expression.Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
		if continuing == 0 {
			mergedLive = cloneLive(l.live)
		} else if !sameLiveState(mergedLive, l.live) {
			return Operand{}, fmt.Errorf("Result match arms disagree on resource state after Flow validation")
		}
		continuing++
	}
	if len(seen) != 2 {
		return Operand{}, fmt.Errorf("non-exhaustive Result match")
	}
	l.current = joinBlock
	if continuing == 0 {
		l.live = cloneLive(branchLive)
		l.terminate(Terminator{Kind: TerminatorUnreachable, Span: expression.Span, UnwindSuccessor: NoBlock})
		return Operand{}, nil
	}
	l.live = mergedLive
	if destination == NoLocal {
		return Operand{}, nil
	}
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerWhile(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 2 {
		return Operand{}, fmt.Errorf("while expression has %d children", len(expression.Children))
	}
	headerLive := cloneLive(l.live)
	headerBlock := l.newBlock()
	l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Span, Successors: []BlockID{headerBlock}, UnwindSuccessor: NoBlock})

	l.current = headerBlock
	l.live = cloneLive(headerLive)
	condition, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if !sameLiveState(headerLive, l.live) {
		return Operand{}, fmt.Errorf("while condition changes resource state after Flow validation")
	}
	bodyBlock := l.newBlock()
	exitBlock := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Children[0].Span,
		Successors: []BlockID{bodyBlock, exitBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: "bool", SwitchOperands: []Operand{condition},
	})

	l.current = bodyBlock
	l.live = cloneLive(headerLive)
	if _, err := l.lowerExpression(&expression.Children[1]); err != nil {
		return Operand{}, err
	}
	if !l.blockTerminated(l.current) {
		if !sameLiveState(headerLive, l.live) {
			return Operand{}, fmt.Errorf("while back-edge changes resource state after Flow validation")
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Children[1].Span, Successors: []BlockID{headerBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = exitBlock
	l.live = cloneLive(headerLive)
	return Operand{}, nil
}

func (l *functionLowerer) lowerIf(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) < 2 || len(expression.Children) > 3 {
		return Operand{}, fmt.Errorf("if expression has %d children", len(expression.Children))
	}
	condition, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	branchLive := cloneLive(l.live)
	thenBlock := l.newBlock()
	elseBlock := l.newBlock()
	joinBlock := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Children[0].Span,
		Successors: []BlockID{thenBlock, elseBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: "bool", SwitchOperands: []Operand{condition},
	})

	destination := NoLocal
	if expression.Type != hir.TypeVoid {
		destination = l.newTemporary()
	}

	l.current = thenBlock
	l.live = cloneLive(branchLive)
	thenValue, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	thenContinues := !l.blockTerminated(l.current)
	thenLive := cloneLive(l.live)
	if thenContinues {
		if destination != NoLocal {
			if thenValue.Kind == OperandInvalid {
				return Operand{}, fmt.Errorf("value-producing if has an empty then branch")
			}
			l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[1].Span, DestinationLocal: destination, Operands: []Operand{thenValue}})
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Children[1].Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = elseBlock
	l.live = cloneLive(branchLive)
	elseValue := Operand{}
	if len(expression.Children) == 3 {
		elseValue, err = l.lowerExpression(&expression.Children[2])
		if err != nil {
			return Operand{}, err
		}
	}
	elseContinues := !l.blockTerminated(l.current)
	elseLive := cloneLive(l.live)
	if elseContinues {
		if destination != NoLocal {
			if elseValue.Kind == OperandInvalid {
				return Operand{}, fmt.Errorf("value-producing if has an empty else branch")
			}
			l.appendOperation(Operation{Kind: OperationAssign, Span: expressionSpan(&expression.Children[len(expression.Children)-1], expression.Span), DestinationLocal: destination, Operands: []Operand{elseValue}})
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = joinBlock
	switch {
	case thenContinues && elseContinues:
		if !sameLiveState(thenLive, elseLive) {
			return Operand{}, fmt.Errorf("if branches disagree on resource state after Flow validation")
		}
		l.live = thenLive
	case thenContinues:
		l.live = thenLive
	case elseContinues:
		l.live = elseLive
	default:
		l.live = cloneLive(branchLive)
		l.terminate(Terminator{Kind: TerminatorUnreachable, Span: expression.Span, UnwindSuccessor: NoBlock})
		return Operand{}, nil
	}
	if destination == NoLocal {
		return Operand{}, nil
	}
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerShortCircuit(expression *hir.Expression) (Operand, error) {
	left, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	branchLive := cloneLive(l.live)
	rightBlock := l.newBlock()
	shortBlock := l.newBlock()
	joinBlock := l.newBlock()
	successors := []BlockID{rightBlock, shortBlock}
	shortValue := int32(0)
	if expression.Operator == "||" {
		successors = []BlockID{shortBlock, rightBlock}
		shortValue = 1
	}
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Children[0].Span,
		Successors: successors, UnwindSuccessor: NoBlock,
		SwitchKind: "bool", SwitchOperands: []Operand{left},
	})

	destination := l.newTemporary()
	l.current = shortBlock
	l.live = cloneLive(branchLive)
	l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[0].Span, DestinationLocal: destination, Operands: []Operand{{Kind: OperandI32, I32: shortValue}}})
	l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
	shortLive := cloneLive(l.live)

	l.current = rightBlock
	l.live = cloneLive(branchLive)
	right, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	rightContinues := !l.blockTerminated(l.current)
	rightLive := cloneLive(l.live)
	if rightContinues {
		if right.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("short-circuit right operand has no value")
		}
		l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[1].Span, DestinationLocal: destination, Operands: []Operand{right}})
		l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = joinBlock
	if rightContinues {
		if !sameLiveState(shortLive, rightLive) {
			return Operand{}, fmt.Errorf("short-circuit branches disagree on resource state after Flow validation")
		}
		l.live = rightLive
	} else {
		l.live = shortLive
	}
	l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "source-short-circuit", RemainingCount: 0})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerCall(expression *hir.Expression) (Operand, error) {
	arguments := expression.Children
	if len(arguments) > 0 {
		arguments = arguments[1:]
	}
	argumentSpans := make([]hir.Span, 0, len(arguments))
	argumentValues := make([]Operand, 0, len(arguments))
	hasNestedSideEffect := false
	for index := range arguments {
		argument := &arguments[index]
		hasNestedSideEffect = hasNestedSideEffect || containsCall(argument)
		value, err := l.lowerExpression(argument)
		if err != nil {
			return Operand{}, err
		}
		argumentSpans = append(argumentSpans, argument.Span)
		if l.isLiveResourcePath(argument) {
			l.movePath(argument, "call-argument", true)
			value = Operand{}
		}
		argumentValues = append(argumentValues, value)
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
	destination := l.newTemporary()
	continuation := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorCall, Span: expression.Span, Successors: []BlockID{continuation}, UnwindSuccessor: NoBlock,
		Callee: expression.Symbol, Arguments: argumentValues, DestinationLocal: destination,
	})
	l.current = continuation
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerCheckedDivision(expression *hir.Expression, left, right Operand) error {
	categories := []string{"division-by-zero", "signed-division-overflow"}
	l.appendOperation(Operation{Kind: OperationCheckedDivision, Span: expression.Span, TrapCategories: slices.Clone(categories)})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: "checked-division", SwitchOperands: []Operand{left, right},
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: expression.Span, UnwindSuccessor: NoBlock, TrapCategories: slices.Clone(categories)})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: expression.Span, Categories: slices.Clone(categories), CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	return nil
}

func (l *functionLowerer) lowerPropagate(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 {
		return Operand{}, fmt.Errorf("propagate expression has %d children", len(expression.Children))
	}
	operand := &expression.Children[0]
	resultValue, err := l.lowerExpression(operand)
	if err != nil {
		return Operand{}, err
	}
	if resultValue.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("propagate operand has no value")
	}
	if l.isLiveResourcePath(operand) {
		movedResult := l.newTemporary()
		l.appendOperation(Operation{Kind: OperationAssign, Span: operand.Span, DestinationLocal: movedResult, Operands: []Operand{resultValue}})
		l.appendOperation(Operation{Kind: OperationMove, Span: operand.Span, Symbol: operand.Symbol, Destination: "result-branch"})
		l.live[operand.Symbol] = false
		resultValue = Operand{Kind: OperandLocal, Local: movedResult}
	}
	l.appendOperation(Operation{Kind: OperationResultSwitch, Span: expression.Span, Operands: []Operand{resultValue}})
	success := l.newBlock()
	errorBlock := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{success, errorBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: "result", SwitchOperands: []Operand{resultValue},
	})

	errorLive := cloneLive(l.live)
	l.current = errorBlock
	errorActions := l.emitCleanup(errorLive)
	l.terminate(Terminator{Kind: TerminatorReturn, Span: expression.Span, UnwindSuccessor: NoBlock, ReturnValue: resultValue})
	question := hir.Span{Source: expression.Span.Source, StartByte: max(expression.Span.StartByte, expression.Span.EndByte-1), EndByte: expression.Span.EndByte}
	l.result.ResultBranches = append(l.result.ResultBranches, ResultBranchFact{
		Question: question, ResultExpression: operand.Span, EvaluationCount: 1,
		SuccessPayload: "Move", ErrorPayload: "Move", ResultTemporaryDeadOn: []string{"success", "error"},
		ErrorCleanupActions: errorActions,
	})
	l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "question", RemainingCount: 0})
	l.current = success
	payload := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationResultPayload, Span: expression.Span, DestinationLocal: payload,
		Operands: []Operand{resultValue}, Variant: "ok",
	})
	return Operand{Kind: OperandLocal, Local: payload}, nil
}

func (l *functionLowerer) lowerReturn(expression *hir.Expression, exit hir.Span) error {
	value, err := l.lowerExpression(expression)
	if err != nil {
		return err
	}
	if l.isLiveResourcePath(expression) {
		l.movePath(expression, "return-value", true)
		value = Operand{}
	}
	actions := l.emitCleanup(l.live)
	l.result.CleanupPaths = append(l.result.CleanupPaths, CleanupPathFact{Exit: exit, ExitKind: "return", Actions: actions})
	if len(actions) > 0 {
		l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
	}
	l.terminate(Terminator{Kind: TerminatorReturn, Span: exit, UnwindSuccessor: NoBlock, ReturnValue: value})
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

func (l *functionLowerer) newTemporary() LocalID {
	result := l.nextLocal
	l.nextLocal++
	return result
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
	for index := range expression.Arms {
		if containsCall(&expression.Arms[index].Expression) {
			return true
		}
	}
	return false
}

func parseI32(text string) (int32, error) {
	clean := strings.ReplaceAll(text, "_", "")
	base := 10
	digits := clean
	if strings.HasPrefix(clean, "0x") || strings.HasPrefix(clean, "0X") {
		base, digits = 16, clean[2:]
	} else if strings.HasPrefix(clean, "0b") || strings.HasPrefix(clean, "0B") {
		base, digits = 2, clean[2:]
	}
	value, err := strconv.ParseInt(digits, base, 32)
	if err != nil {
		return 0, fmt.Errorf("lower i32 literal %q: %w", text, err)
	}
	return int32(value), nil
}
