package hir

import (
	"fmt"
	"strconv"
	"strings"
)

// ADR 0557 bounds each inline body independently of its reachable call graph.
const (
	inlineMaxParameters      = 4
	inlineMaxLocals          = 16
	inlineMaxSteps           = 64
	inlineMaxExpressionNodes = 64
)

type inlineEncodingNode struct {
	Op       string
	Value    string
	Children []inlineEncodingNode
}

// All mutable state belongs to one encoding; the source HIR is read-only.
type inlineFunctionEncoder struct {
	parameters map[SymbolID]int
	locals     map[SymbolID]Local
	bound      map[SymbolID]int
	declared   map[SymbolID]bool
	resolve    func(SymbolID) *Function
	// A separate result slice keeps the lookup maps from escaping with targets.
	callees   *[]*Function
	count     int
	mutating  bool
	branches  bool
	armLocals []SymbolID
	hasLoop   bool
	loopDepth int
}

func (encoder *inlineFunctionEncoder) prepare(fn Function) ([]Statement, *Expression, error) {
	if !fn.Public || fn.Unsafe || len(fn.Effects) != 0 || fn.OwnerType != (Type{}) || fn.DropGlueFor != (Type{}) || fn.ResultType != TypeI32 || len(fn.Parameters) > inlineMaxParameters || len(fn.Locals) > inlineMaxLocals {
		return nil, nil, fmt.Errorf("@inline requires a public safe i32 function with at most four i32 parameters and no effects or owner and at most 16 i32 locals")
	}
	for i, p := range fn.Parameters {
		if p.Type != TypeI32 || p.Mutable {
			return nil, nil, fmt.Errorf("@inline requires immutable i32 parameters")
		}
		encoder.parameters[p.Symbol] = i
	}
	expression, statements := fn.Body.Tail, fn.Body.Statements
	if len(statements) > 0 && statements[len(statements)-1].Kind == StatementReturn && expression == nil {
		expression = statements[len(statements)-1].Expression
		statements = statements[:len(statements)-1]
	}
	if len(statements) > inlineMaxSteps {
		return nil, nil, fmt.Errorf("@inline requires at most 64 straight-line steps")
	}
	for _, local := range fn.Locals {
		if local.Type != TypeI32 {
			return nil, nil, fmt.Errorf("@inline requires i32 locals")
		}
		encoder.locals[local.Symbol] = local
	}
	return statements, expression, nil
}

func (encoder *inlineFunctionEncoder) statement(statement Statement, index int) (inlineEncodingNode, error) {
	symbol, mutable, err := inlineStatementBinding(statement, encoder.locals, encoder.parameters, encoder.declared)
	if err != nil {
		return inlineEncodingNode{}, err
	}
	encoder.mutating = encoder.mutating || mutable
	if statement.Kind == StatementBinding {
		encoder.declared[symbol] = true
	}
	initializer, err := encoder.expression(statement.Expression)
	if err != nil {
		return inlineEncodingNode{}, err
	}
	// Bind only after encoding the initializer: a local cannot read itself.
	encoder.bound[symbol] = index
	return initializer, nil
}

func inlineStatementBinding(statement Statement, locals map[SymbolID]Local, parameters map[SymbolID]int, declared map[SymbolID]bool) (SymbolID, bool, error) {
	if statement.Kind == StatementAssignment {
		if statement.Operator != "=" || statement.Target == nil || statement.Target.Kind != ExpressionPath || statement.Target.Type != TypeI32 {
			return 0, false, fmt.Errorf("@inline assignment requires a mutable local path")
		}
		symbol := statement.Target.Symbol
		local, ok := locals[symbol]
		if !ok || !local.Mutable || !declared[symbol] || statement.For != nil {
			return 0, false, fmt.Errorf("@inline assignment requires a preceding mutable i32 local")
		}
		return symbol, true, nil
	}
	local, ok := locals[statement.Binding]
	_, parameter := parameters[statement.Binding]
	if statement.Kind != StatementBinding || !ok || declared[statement.Binding] || parameter || statement.Target != nil || statement.For != nil {
		return 0, false, fmt.Errorf("@inline requires straight-line i32 local bindings and assignments")
	}
	return statement.Binding, local.Mutable, nil
}

func (encoder *inlineFunctionEncoder) expression(e *Expression) (inlineEncodingNode, error) {
	return encoder.typedExpression(e, TypeI32, false)
}

func (encoder *inlineFunctionEncoder) typedExpression(e *Expression, typ Type, branchArm bool) (inlineEncodingNode, error) {
	encoder.count++
	if e == nil || encoder.count > inlineMaxExpressionNodes || e.Type != typ || len(e.Effects) != 0 || len(e.Arms) != 0 || e.UnsafeScope != nil {
		return inlineEncodingNode{}, fmt.Errorf("@inline requires at most 64 pure i32 expression nodes")
	}
	if branchArm && e.Kind == ExpressionBlock && e.Block != nil && len(e.Children) == 0 {
		return encoder.valueBlock(e.Block)
	}
	if e.Block != nil {
		return inlineEncodingNode{}, fmt.Errorf("@inline blocks are only supported as if value arms")
	}
	if e.Kind == ExpressionGroup {
		if len(e.Children) != 1 {
			return inlineEncodingNode{}, fmt.Errorf("invalid inline group")
		}
		return encoder.typedExpression(&e.Children[0], typ, branchArm)
	}
	if typ == TypeBool {
		return encoder.condition(e)
	}
	if e.Kind == ExpressionIf {
		if len(e.Children) != 3 {
			return inlineEncodingNode{}, fmt.Errorf("@inline if requires two value arms")
		}
		condition, err := encoder.typedExpression(&e.Children[0], TypeBool, false)
		if err != nil {
			return inlineEncodingNode{}, err
		}
		thenValue, err := encoder.typedExpression(&e.Children[1], TypeI32, true)
		if err != nil {
			return inlineEncodingNode{}, err
		}
		elseValue, err := encoder.typedExpression(&e.Children[2], TypeI32, true)
		if err != nil {
			return inlineEncodingNode{}, err
		}
		encoder.branches = true
		return inlineEncodingNode{Op: "if", Children: []inlineEncodingNode{condition, thenValue, elseValue}}, nil
	}
	n, children, err := encoder.expressionNode(e)
	if err != nil {
		return n, err
	}
	// Preserve canonical null leaves and cap allocation even for malformed HIR.
	if len(children) != 0 {
		n.Children = make([]inlineEncodingNode, 0, min(len(children), inlineMaxExpressionNodes-encoder.count))
	}
	for i := range children {
		child, err := encoder.expression(&children[i])
		if err != nil {
			return n, err
		}
		n.Children = append(n.Children, child)
	}
	return n, nil
}

func (encoder *inlineFunctionEncoder) valueBlock(block *Block) (inlineEncodingNode, error) {
	if len(block.Statements) == 0 {
		return encoder.expression(block.Tail)
	}
	if len(block.Statements) > len(encoder.locals)-len(encoder.declared) {
		return inlineEncodingNode{}, fmt.Errorf("@inline arm declarations exceed the local budget")
	}
	defer encoder.closeScope(len(encoder.armLocals))
	node := inlineEncodingNode{Op: "arm-bindings", Children: make([]inlineEncodingNode, 0, len(block.Statements)+1)}
	for _, statement := range block.Statements {
		if statement.Kind != StatementBinding || encoder.locals[statement.Binding].Mutable {
			return inlineEncodingNode{}, fmt.Errorf("@inline arms require immutable i32 bindings")
		}
		initializer, err := encoder.scopedBinding(statement)
		if err != nil {
			return inlineEncodingNode{}, err
		}
		node.Children = append(node.Children, initializer)
	}
	tail, err := encoder.expression(block.Tail)
	if err != nil {
		return inlineEncodingNode{}, err
	}
	node.Children = append(node.Children, tail)
	return node, nil
}

func (encoder *inlineFunctionEncoder) condition(e *Expression) (inlineEncodingNode, error) {
	if e.Kind == ExpressionLiteral && len(e.Children) == 0 && (e.Value == "true" || e.Value == "false") {
		return inlineEncodingNode{Op: "literal", Value: e.Value}, nil
	}
	operandType := TypeI32
	arity := 2
	kind := ExpressionBinary
	switch e.Operator {
	case "!":
		operandType, arity, kind = TypeBool, 1, ExpressionPrefix
	case "&&", "||":
		operandType = TypeBool
	case "==", "!=", "<", "<=", ">", ">=":
	default:
		return inlineEncodingNode{}, fmt.Errorf("unsupported @inline condition")
	}
	if e.Kind != kind || len(e.Children) != arity {
		return inlineEncodingNode{}, fmt.Errorf("invalid @inline condition operands")
	}
	children := make([]inlineEncodingNode, arity)
	for i := range e.Children {
		child, err := encoder.typedExpression(&e.Children[i], operandType, false)
		if err != nil {
			return inlineEncodingNode{}, err
		}
		children[i] = child
	}
	return inlineEncodingNode{Op: e.Operator, Children: children}, nil
}

func (encoder *inlineFunctionEncoder) expressionNode(e *Expression) (inlineEncodingNode, []Expression, error) {
	n := inlineEncodingNode{Op: e.Kind.String()}
	switch e.Kind {
	case ExpressionLiteral:
		value, err := inlineLiteralValue(e.Value)
		n.Value = value
		return n, e.Children, err
	case ExpressionPath:
		if i, ok := encoder.parameters[e.Symbol]; ok {
			n.Value = strconv.Itoa(i)
			return n, e.Children, nil
		}
		if i, ok := encoder.bound[e.Symbol]; ok {
			return inlineEncodingNode{Op: "local", Value: strconv.Itoa(i)}, nil, nil
		}
		return n, nil, fmt.Errorf("inline path must name a parameter or preceding local")
	case ExpressionPrefix:
		if len(e.Children) != 1 || (e.Operator != "-" && e.Operator != "+") {
			return n, nil, fmt.Errorf("unsupported inline prefix")
		}
		n.Op = e.Operator
	case ExpressionBinary:
		if len(e.Children) != 2 || !strings.Contains(" + - * / % ", " "+e.Operator+" ") {
			return n, nil, fmt.Errorf("unsupported inline binary operator")
		}
		n.Op = e.Operator
	case ExpressionCall:
		return encoder.callNode(e)
	default:
		return n, nil, fmt.Errorf("@inline calls, captures, control flow and aggregate operations are closed")
	}
	return n, e.Children, nil
}

func (encoder *inlineFunctionEncoder) callNode(e *Expression) (inlineEncodingNode, []Expression, error) {
	n := inlineEncodingNode{Op: e.Kind.String()}
	if encoder.resolve == nil || e.ForeignABI != "" || len(e.Children) == 0 || e.Children[0].Kind != ExpressionPath || e.Children[0].Symbol != e.Symbol {
		return n, nil, fmt.Errorf("@inline requires a resolved same-module inline call")
	}
	callee := encoder.resolve(e.Symbol)
	if callee == nil || callee.Symbol != e.Symbol || callee.InlineBody == "" || !callee.Public || callee.Unsafe || len(callee.Effects) != 0 || len(e.Children)-1 != len(callee.Parameters) {
		return n, nil, fmt.Errorf("@inline call target must be a same-module public safe nongeneric inline function with matching arity")
	}
	*encoder.callees = append(*encoder.callees, callee)
	n.Op, n.Value = "call", callee.Name
	// The resolved declaration path is not an arithmetic expression node.
	return n, e.Children[1:], nil
}

func inlineLiteralValue(text string) (string, error) {
	text = strings.ToLower(strings.ReplaceAll(text, "_", ""))
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0b") {
		base = 0
	}
	value, err := strconv.ParseInt(text, base, 64)
	if err != nil {
		return "", fmt.Errorf("invalid inline i32 literal")
	}
	return strconv.FormatInt(value, 10), nil
}

func inlineEncodingVersion(hasBindings, hasCalls, mutating, branches bool) string {
	if branches {
		return "fairyc.inline-function/v5"
	}
	if mutating {
		return "fairyc.inline-function/v4"
	}
	if hasCalls {
		return "fairyc.inline-function/v3"
	}
	if hasBindings {
		return "fairyc.inline-function/v2"
	}
	return "fairyc.inline-function/v1"
}
