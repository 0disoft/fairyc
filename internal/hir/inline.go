package hir

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const InlineFunctionPrefix = "fyinlinefn1-sha256-"

// InlineFunctionID describes the closed first source body family. Source spans,
// local symbol numbering, parameter names, whitespace and literal spelling are inert.
func InlineFunctionID(fn Function) (string, error) {
	if !fn.Public || fn.Unsafe || len(fn.Effects) != 0 || fn.OwnerType != (Type{}) || fn.DropGlueFor != (Type{}) || fn.ResultType != TypeI32 || len(fn.Parameters) > 4 || len(fn.Locals) != 0 {
		return "", fmt.Errorf("@inline requires a public safe i32 function with at most four i32 parameters and no effects, owner or locals")
	}
	parameters := make(map[SymbolID]int)
	for i, p := range fn.Parameters {
		if p.Type != TypeI32 || p.Mutable {
			return "", fmt.Errorf("@inline requires immutable i32 parameters")
		}
		parameters[p.Symbol] = i
	}
	expression := fn.Body.Tail
	if len(fn.Body.Statements) == 1 && fn.Body.Statements[0].Kind == StatementReturn && fn.Body.Tail == nil {
		expression = fn.Body.Statements[0].Expression
	} else if len(fn.Body.Statements) != 0 {
		return "", fmt.Errorf("@inline requires one returned expression")
	}
	type node struct {
		Op       string
		Value    string
		Children []node
	}
	count := 0
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
				return n, fmt.Errorf("inline path must name a parameter")
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
	root, err := encode(expression)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(struct {
		Version string
		Arity   int
		Root    node
	}{"fairyc.inline-function/v1", len(fn.Parameters), root})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%x", InlineFunctionPrefix, sha256.Sum256(data)), nil
}
