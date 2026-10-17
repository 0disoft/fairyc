package main

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

// These projections retain actual Go HIR field names and scalar tree structure,
// while omitting package process IDs and provenance. Nominal corpora additionally
// retain actual struct declaration/field facts, independently of candidate IDs.
// Expression.Children preserve order. A call's first child is its callee path
// (Type == invalid, empty Value); later children are arguments. If has condition,
// then-block and optional else-block/if children. While has condition/body.
// For is a Statement.For with Initializer, Condition, Post and Body, including
// explicit null omitted clauses. No candidate IDs or lowered output are read.
type hirOraclePackage struct{ Modules []hirOracleModule }
type hirOracleModule struct {
	Path       string
	SourcePath string
	Functions  []hirOracleFunction
	Structs    []hir.Struct `json:",omitempty"`
}
type hirOracleFunction struct {
	Symbol     hir.SymbolID
	Name       string
	Public     bool
	Span       hir.Span
	Parameters []hir.Parameter
	Locals     []hir.Local
	ResultType hir.Type
	Body       hir.Block
}

// Symbol is only a lookup key, never the comparison identity. ModuleIndex is the
// index in hir.Modules (Go sorts source paths, independently of Files order).
// Declaration is the exact function/parameter/local NAME span, not body span.
// Owner and OwnerDeclaration distinguish same-named bindings between functions;
// Declaration distinguishes shadowed bindings within a function.
type hirOracleSymbol struct {
	Kind             string    `json:"kind"`
	ModuleIndex      int       `json:"moduleIndex"`
	Module           string    `json:"module"`
	Name             string    `json:"name"`
	QualifiedName    string    `json:"qualifiedName"`
	Owner            string    `json:"owner,omitempty"`
	OwnerDeclaration *hir.Span `json:"ownerDeclaration,omitempty"`
	Declaration      hir.Span  `json:"declaration"`
	Type             hir.Type  `json:"type"`
	Mutable          bool      `json:"mutable"`
}

func observeOracleHIR(pkg hir.Package, files []semantic.File) (*hirOraclePackage, map[string]hirOracleSymbol, error) {
	result := &hirOraclePackage{Modules: []hirOracleModule{}}
	symbols := map[string]hirOracleSymbol{}
	declarations := map[string]hir.Span{}
	for _, file := range files {
		for _, fn := range file.Module.Functions {
			declarations[file.Path+"\x00"+fn.Name] = hir.Span{Source: file.Path, StartByte: fn.NameSpan.StartByte, EndByte: fn.NameSpan.EndByte}
		}
	}
	register := func(id hir.SymbolID, identity hirOracleSymbol) error {
		key := strconv.Itoa(int(id))
		if id == hir.NoSymbol {
			return fmt.Errorf("declaration has no symbol: %s", identity.Name)
		}
		if _, found := symbols[key]; found {
			return fmt.Errorf("duplicate symbol %d", id)
		}
		symbols[key] = identity
		return nil
	}
	for mi, mod := range pkg.Modules {
		_, sourcePath := hir.SourcePathParts(mod.SourcePath)
		module := hirOracleModule{Path: mod.Path, SourcePath: sourcePath, Functions: []hirOracleFunction{}, Structs: mod.Structs}
		for _, fn := range mod.Functions {
			declaration, found := declarations[sourcePath+"\x00"+fn.Name]
			if !found {
				return nil, nil, fmt.Errorf("missing function declaration %s::%s", mod.Path, fn.Name)
			}
			qualified := mod.Path + "::" + fn.Name
			if err := register(fn.Symbol, hirOracleSymbol{Kind: "function", ModuleIndex: mi, Module: mod.Path, Name: fn.Name, QualifiedName: qualified, Declaration: declaration, Type: fn.ResultType}); err != nil {
				return nil, nil, err
			}
			function := hirOracleFunction{Symbol: fn.Symbol, Name: fn.Name, Public: fn.Public, Span: fn.Span, ResultType: fn.ResultType,
				Parameters: append([]hir.Parameter{}, fn.Parameters...), Locals: append([]hir.Local{}, fn.Locals...), Body: fn.Body}
			for _, parameter := range fn.Parameters {
				identity := hirOracleSymbol{Kind: "parameter", ModuleIndex: mi, Module: mod.Path, Name: parameter.Name, QualifiedName: qualified + "::" + parameter.Name, Owner: qualified, OwnerDeclaration: &declaration, Declaration: parameter.Span, Type: parameter.Type, Mutable: parameter.Mutable}
				if err := register(parameter.Symbol, identity); err != nil {
					return nil, nil, err
				}
			}
			for _, local := range fn.Locals {
				identity := hirOracleSymbol{Kind: "local", ModuleIndex: mi, Module: mod.Path, Name: local.Name, QualifiedName: qualified + "::" + local.Name, Owner: qualified, OwnerDeclaration: &declaration, Declaration: local.Span, Type: local.Type, Mutable: local.Mutable}
				if err := register(local.Symbol, identity); err != nil {
					return nil, nil, err
				}
			}
			module.Functions = append(module.Functions, function)
		}
		result.Modules = append(result.Modules, module)
	}
	// Register every declaration before traversing: forward/imported/recursive
	// references must resolve against the complete independent symbol map.
	for mi := range result.Modules {
		for fi := range result.Modules[mi].Functions {
			fn := &result.Modules[mi].Functions[fi]
			body, err := observeOracleBlock(fn.Body, symbols)
			if err != nil {
				return nil, nil, err
			}
			fn.Body = body
		}
	}
	return result, symbols, nil
}

func observeOracleBlock(block hir.Block, symbols map[string]hirOracleSymbol) (hir.Block, error) {
	result := hir.Block{Span: block.Span, Statements: []hir.Statement{}}
	for _, statement := range block.Statements {
		observed, err := observeOracleStatement(statement, symbols)
		if err != nil {
			return hir.Block{}, err
		}
		result.Statements = append(result.Statements, observed)
	}
	tail, err := observeOracleExpression(block.Tail, symbols)
	result.Tail = tail
	return result, err
}

func observeOracleStatement(statement hir.Statement, symbols map[string]hirOracleSymbol) (hir.Statement, error) {
	switch statement.Kind {
	case hir.StatementBinding, hir.StatementAssignment, hir.StatementExpressionStatement, hir.StatementWhile, hir.StatementFor, hir.StatementReturn, hir.StatementDefer, hir.StatementAssert, hir.StatementDiscard, hir.StatementBreak, hir.StatementContinue:
	default:
		return hir.Statement{}, fmt.Errorf("unobserved statement kind %q", statement.Kind)
	}
	result := statement
	if statement.Binding != hir.NoSymbol {
		if _, found := symbols[strconv.Itoa(int(statement.Binding))]; !found {
			return result, fmt.Errorf("unknown binding %d", statement.Binding)
		}
	}
	var err error
	if result.Target, err = observeOracleExpression(statement.Target, symbols); err != nil {
		return result, err
	}
	if result.Expression, err = observeOracleExpression(statement.Expression, symbols); err != nil {
		return result, err
	}
	if statement.For != nil {
		loop := *statement.For
		if loop.Initializer != nil {
			init, err := observeOracleStatement(*loop.Initializer, symbols)
			if err != nil {
				return result, err
			}
			loop.Initializer = &init
		}
		if loop.Condition, err = observeOracleExpression(loop.Condition, symbols); err != nil {
			return result, err
		}
		if loop.Post != nil {
			post, err := observeOracleStatement(*loop.Post, symbols)
			if err != nil {
				return result, err
			}
			loop.Post = &post
		}
		if loop.Body, err = observeOracleBlock(loop.Body, symbols); err != nil {
			return result, err
		}
		result.For = &loop
	}
	return result, nil
}

func observeOracleExpression(expression *hir.Expression, symbols map[string]hirOracleSymbol) (*hir.Expression, error) {
	if expression == nil {
		return nil, nil
	}
	switch expression.Kind {
	case hir.ExpressionLiteral, hir.ExpressionPath, hir.ExpressionGroup, hir.ExpressionPrefix, hir.ExpressionBinary, hir.ExpressionCall, hir.ExpressionBlock, hir.ExpressionIf, hir.ExpressionWhile, hir.ExpressionRecord, hir.ExpressionFieldExpression:
	default:
		return nil, fmt.Errorf("unobserved expression kind %q", expression.Kind)
	}
	result := *expression
	if expression.Symbol != hir.NoSymbol && expression.Kind != hir.ExpressionRecord {
		if _, found := symbols[strconv.Itoa(int(expression.Symbol))]; !found {
			return nil, fmt.Errorf("unknown expression target %d", expression.Symbol)
		}
	}
	if result.Kind == hir.ExpressionLiteral && result.Type != hir.TypeBool {
		value, err := oracleCanonicalInteger(result.Value)
		if err != nil {
			return nil, err
		}
		result.Value = value
	}
	result.Children = []hir.Expression{}
	for index := range expression.Children {
		child, err := observeOracleExpression(&expression.Children[index], symbols)
		if err != nil {
			return nil, err
		}
		result.Children = append(result.Children, *child)
	}
	if expression.Block != nil {
		block, err := observeOracleBlock(*expression.Block, symbols)
		if err != nil {
			return nil, err
		}
		result.Block = &block
	}
	return &result, nil
}

func oracleCanonicalInteger(raw string) (string, error) {
	text := strings.ReplaceAll(raw, "_", "")
	base := 10
	if strings.HasPrefix(text, "0x") {
		base, text = 16, text[2:]
	} else if strings.HasPrefix(text, "0b") {
		base, text = 2, text[2:]
	}
	value, ok := new(big.Int).SetString(text, base)
	if !ok {
		return "", fmt.Errorf("non-integer scalar literal %q", raw)
	}
	return value.String(), nil
}
