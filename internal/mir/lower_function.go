package mir

import (
	"context"
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"slices"
)

type bindingInfo struct {
	typ  hir.Type
	span hir.Span
}

type activeRegion struct {
	symbol    hir.SymbolID
	span      hir.Span
	factIndex int
}

type activeLoop struct {
	header         BlockID
	continueTarget BlockID
	exit           BlockID
	body           hir.Span
	regionDepth    int
	headerLive     map[hir.SymbolID]bool
}

type deferredCall struct {
	scope         hir.Span
	span          hir.Span
	callee        hir.SymbolID
	arguments     []Operand
	argumentSpans []hir.Span
}

type functionLowerer struct {
	ctx          context.Context
	module       string
	source       *hir.Function
	properties   typeProperties
	result       Function
	current      BlockID
	bindings     map[hir.SymbolID]bindingInfo
	live         map[hir.SymbolID]bool
	regions      map[hir.SymbolID]int
	regionStack  []activeRegion
	loopStack    []activeLoop
	currentScope hir.Span
	deferred     []deferredCall
	nextLocal    LocalID
}

func newFunctionLowerer(module string, function *hir.Function, properties typeProperties) *functionLowerer {
	result := Function{
		Symbol:      function.Symbol,
		Name:        module + "::" + function.Name,
		Span:        function.Span,
		ResultType:  function.ResultType,
		Effects:     slices.Clone(function.Effects),
		Entry:       0,
		DropGlueFor: function.DropGlueFor,
	}
	return &functionLowerer{
		module:     module,
		source:     function,
		properties: properties,
		result:     result,
		bindings:   make(map[hir.SymbolID]bindingInfo),
		live:       make(map[hir.SymbolID]bool),
		regions:    make(map[hir.SymbolID]int),
	}
}

func (l *functionLowerer) lower() (Function, error) {
	l.newBlock()
	maximumSymbol := hir.SymbolID(0)
	for index, parameter := range l.source.Parameters {
		maximumSymbol = max(maximumSymbol, parameter.Symbol)
		l.result.Parameters = append(l.result.Parameters, LocalID(parameter.Symbol))
		l.result.ParameterTypes = append(l.result.ParameterTypes, parameter.Type)
		l.bindings[parameter.Symbol] = bindingInfo{typ: parameter.Type, span: parameter.Span}
		isDropReceiver := l.source.DropGlueFor != hir.ParseType("") && index == 0 && parameter.Type == l.source.DropGlueFor
		if l.properties.NeedsDrop(parameter.Type) && !isDropReceiver {
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
	if !terminated && l.source.ResultType == hir.TypeVoid {
		actions := l.emitDeferredCallsForScope(l.source.Body.Span)
		actions = append(actions, l.emitFunctionExitCleanup(l.live, l.source.Body.Span, "return")...)
		l.result.CleanupPaths = append(l.result.CleanupPaths, CleanupPathFact{Exit: l.source.Body.Span, ExitKind: "return", Actions: actions})
		if cleanupContainsDrop(actions) {
			l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
		}
		l.terminate(Terminator{Kind: TerminatorReturn, Span: l.source.Body.Span, UnwindSuccessor: NoBlock})
		terminated = true
	}
	if !terminated {
		return Function{}, fmt.Errorf("function has no lowered return")
	}
	return l.result, nil
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

func (l *functionLowerer) contextError() error {
	if l.ctx == nil {
		return nil
	}
	return l.ctx.Err()
}
