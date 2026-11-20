package hir

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

const InlineFunctionPrefix = "fyinlinefn1-sha256-"

// InlineModuleResolver admits only actual inline declarations in one module.
// Callers must keep the module immutable while using the resolver.
func InlineModuleResolver(module Module) func(SymbolID) *Function {
	functions := make(map[SymbolID]*Function)
	for i := range module.Functions {
		fn := &module.Functions[i]
		if fn.InlineBody != "" {
			functions[fn.Symbol] = fn
		}
	}
	return func(symbol SymbolID) *Function { return functions[symbol] }
}

// InlineFunctionID describes the bounded source body family. Source spans,
// local symbol numbering, parameter names, whitespace and literal spelling are inert.
func InlineFunctionID(fn Function, resolvers ...func(SymbolID) *Function) (string, error) {
	var resolve func(SymbolID) *Function
	if len(resolvers) != 0 {
		resolve = resolvers[0]
	}
	// Walk declarations, never substituted expression trees. A resolver admits
	// only actual same-module exported nongeneric @inline declarations.
	state := make(map[SymbolID]uint8)
	var root []byte
	var visit func(Function) error
	visit = func(current Function) error {
		if state[current.Symbol] == 1 {
			return fmt.Errorf("@inline recursive call graphs are closed")
		}
		if state[current.Symbol] == 2 {
			return nil
		}
		if len(state) >= 64 {
			return fmt.Errorf("@inline requires at most 64 distinct reachable functions including the root")
		}
		state[current.Symbol] = 1
		data, callees, err := inlineFunctionEncoding(current, resolve)
		if err != nil {
			return err
		}
		if current.Symbol == fn.Symbol {
			root = data
		}
		for _, callee := range callees {
			if err := visit(*callee); err != nil {
				return err
			}
		}
		state[current.Symbol] = 2
		return nil
	}
	if err := visit(fn); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%x", InlineFunctionPrefix, sha256.Sum256(root)), nil
}

func inlineFunctionEncoding(fn Function, resolve func(SymbolID) *Function) ([]byte, []*Function, error) {
	var callees []*Function
	encoder := inlineFunctionEncoder{
		parameters: make(map[SymbolID]int), locals: make(map[SymbolID]Local),
		bound: make(map[SymbolID]int), declared: make(map[SymbolID]bool),
		resolve: resolve, callees: &callees,
	}
	statements, expression, err := encoder.prepare(fn)
	if err != nil {
		return nil, nil, err
	}
	var initializers []inlineEncodingNode
	for i, statement := range statements {
		initializer, err := encoder.statement(statement, i)
		if err != nil {
			return nil, nil, err
		}
		initializers = append(initializers, initializer)
	}
	if len(encoder.declared) != len(encoder.locals) {
		return nil, nil, fmt.Errorf("@inline local declaration count mismatch")
	}
	root, err := encoder.expression(expression)
	if err != nil {
		return nil, nil, err
	}
	version := inlineEncodingVersion(len(initializers) != 0, len(callees) != 0, encoder.mutating, encoder.branches)
	if len(initializers) != 0 {
		root = inlineEncodingNode{Op: "bindings", Children: append(initializers, root)}
	}
	data, err := json.Marshal(struct {
		Version string
		Arity   int
		Root    inlineEncodingNode
	}{version, len(fn.Parameters), root})
	if err != nil {
		return nil, nil, err
	}
	return data, callees, nil
}
