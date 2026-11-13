package hir

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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
	if !fn.Public || fn.Unsafe || len(fn.Effects) != 0 || fn.OwnerType != (Type{}) || fn.DropGlueFor != (Type{}) || fn.ResultType != TypeI32 || len(fn.Parameters) > 4 || len(fn.Locals) > 16 {
		return nil, nil, fmt.Errorf("@inline requires a public safe i32 function with at most four i32 parameters and no effects or owner and at most 16 i32 locals")
	}
	parameters := make(map[SymbolID]int)
	for i, p := range fn.Parameters {
		if p.Type != TypeI32 || p.Mutable {
			return nil, nil, fmt.Errorf("@inline requires immutable i32 parameters")
		}
		parameters[p.Symbol] = i
	}
	expression := fn.Body.Tail
	statements := fn.Body.Statements
	if len(statements) > 0 && statements[len(statements)-1].Kind == StatementReturn && expression == nil {
		expression = statements[len(statements)-1].Expression
		statements = statements[:len(statements)-1]
	}
	if len(statements) > 64 {
		return nil, nil, fmt.Errorf("@inline requires at most 64 straight-line steps")
	}
	locals := make(map[SymbolID]Local)
	for _, local := range fn.Locals {
		if local.Type != TypeI32 {
			return nil, nil, fmt.Errorf("@inline requires i32 locals")
		}
		locals[local.Symbol] = local
	}
	bound := make(map[SymbolID]int)
	type node struct {
		Op       string
		Value    string
		Children []node
	}
	count := 0
	var callees []*Function
	hasCalls := false
	var encode func(*Expression) (node, error)
	encode = func(e *Expression) (node, error) {
		count++
		if e == nil || count > 64 || e.Type != TypeI32 || len(e.Effects) != 0 || e.Block != nil || len(e.Arms) != 0 || e.UnsafeScope != nil {
			return node{}, fmt.Errorf("@inline requires at most 64 pure i32 expression nodes")
		}
		n := node{Op: e.Kind.String()}
		switch e.Kind {
		case ExpressionGroup:
			if len(e.Children) != 1 {
				return n, fmt.Errorf("invalid inline group")
			}
			return encode(&e.Children[0])
		case ExpressionLiteral:
			text := strings.ToLower(strings.ReplaceAll(e.Value, "_", ""))
			base := 10
			if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0b") {
				base = 0
			}
			value, err := strconv.ParseInt(text, base, 64)
			if err != nil {
				return n, fmt.Errorf("invalid inline i32 literal")
			}
			n.Value = strconv.FormatInt(value, 10)
		case ExpressionPath:
			i, ok := parameters[e.Symbol]
			if !ok {
				if j, local := bound[e.Symbol]; local {
					n.Op = "local"
					n.Value = strconv.Itoa(j)
					return n, nil
				}
				return n, fmt.Errorf("inline path must name a parameter or preceding local")
			}
			n.Value = strconv.Itoa(i)
		case ExpressionPrefix:
			if len(e.Children) != 1 || (e.Operator != "-" && e.Operator != "+") {
				return n, fmt.Errorf("unsupported inline prefix")
			}
			n.Op = e.Operator
		case ExpressionBinary:
			if len(e.Children) != 2 || !strings.Contains(" + - * / % ", " "+e.Operator+" ") {
				return n, fmt.Errorf("unsupported inline binary operator")
			}
			n.Op = e.Operator
		case ExpressionCall:
			if resolve == nil || e.ForeignABI != "" || len(e.Children) == 0 || e.Children[0].Kind != ExpressionPath || e.Children[0].Symbol != e.Symbol {
				return n, fmt.Errorf("@inline requires a resolved same-module inline call")
			}
			callee := resolve(e.Symbol)
			if callee == nil || callee.Symbol != e.Symbol || callee.InlineBody == "" || !callee.Public || callee.Unsafe || len(callee.Effects) != 0 || len(e.Children)-1 != len(callee.Parameters) {
				return n, fmt.Errorf("@inline call target must be a same-module public safe nongeneric inline function with matching arity")
			}
			hasCalls = true
			callees = append(callees, callee)
			n.Op, n.Value = "call", callee.Name
			// The resolved declaration path is not an arithmetic expression node.
			for i := 1; i < len(e.Children); i++ {
				child, err := encode(&e.Children[i])
				if err != nil {
					return n, err
				}
				n.Children = append(n.Children, child)
			}
			return n, nil
		default:
			return n, fmt.Errorf("@inline calls, captures, control flow and aggregate operations are closed")
		}
		for i := range e.Children {
			child, err := encode(&e.Children[i])
			if err != nil {
				return n, err
			}
			n.Children = append(n.Children, child)
		}
		return n, nil
	}
	var initializers []node
	mutating := false
	declared := make(map[SymbolID]bool)
	for i, statement := range statements {
		symbol := statement.Binding
		if statement.Kind == StatementAssignment {
			if statement.Operator != "=" || statement.Target == nil || statement.Target.Kind != ExpressionPath || statement.Target.Type != TypeI32 {
				return nil, nil, fmt.Errorf("@inline assignment requires a mutable local path")
			}
			symbol = statement.Target.Symbol
			local, ok := locals[symbol]
			if !ok || !local.Mutable || !declared[symbol] || statement.For != nil {
				return nil, nil, fmt.Errorf("@inline assignment requires a preceding mutable i32 local")
			}
			mutating = true
		} else {
			local, ok := locals[symbol]
			_, parameter := parameters[symbol]
			if statement.Kind != StatementBinding || !ok || declared[symbol] || parameter || statement.Target != nil || statement.For != nil {
				return nil, nil, fmt.Errorf("@inline requires straight-line i32 local bindings and assignments")
			}
			mutating = mutating || local.Mutable
			declared[symbol] = true
		}
		initializer, err := encode(statement.Expression)
		if err != nil {
			return nil, nil, err
		}
		initializers = append(initializers, initializer)
		bound[symbol] = i
	}
	if len(declared) != len(locals) {
		return nil, nil, fmt.Errorf("@inline local declaration count mismatch")
	}
	root, err := encode(expression)
	if err != nil {
		return nil, nil, err
	}
	version := "fairyc.inline-function/v1"
	if len(initializers) != 0 {
		version = "fairyc.inline-function/v2"
		root = node{Op: "bindings", Children: append(initializers, root)}
	}
	if hasCalls {
		version = "fairyc.inline-function/v3"
	}
	if mutating {
		version = "fairyc.inline-function/v4"
	}
	data, err := json.Marshal(struct {
		Version string
		Arity   int
		Root    node
	}{version, len(fn.Parameters), root})
	if err != nil {
		return nil, nil, err
	}
	return data, callees, nil
}
