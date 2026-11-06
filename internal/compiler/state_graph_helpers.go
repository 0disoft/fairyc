package compiler

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/syntax"
)

const stateSourceMaxHelpers = 32

func stateGraphSkeleton(file semantic.File) (semantic.File, error) {
	module := file.Module
	if module == nil || len(module.Functions) == 0 || len(module.Functions) > stateSourceMaxHelpers+1 || len(module.States) != 1 ||
		len(module.Imports)+len(module.Constants)+len(module.StaticAssertions)+len(module.Structs)+len(module.Resources)+len(module.Enums)+len(module.ForeignFunctions) != 0 {
		return semantic.File{}, fmt.Errorf("experimental state graph requires one layout, one transaction and at most 32 helpers")
	}
	copyModule := *module
	copyModule.Functions = append([]ast.Function(nil), module.Functions...)
	roots := 0
	for index, function := range module.Functions {
		if function.Name == "transaction" {
			roots++
		}
		if function.Owner != "" || function.Public || function.Unsafe || len(function.TypeParameters)+len(function.Parameters) != 0 ||
			function.ResultType.Kind != ast.TypeKindNamed || function.ResultType.Path != "void" || len(function.ResultType.Arguments) != 0 || function.Body == nil {
			return semantic.File{}, fmt.Errorf("experimental state owners must be private safe nongeneric parameterless void functions")
		}
		for _, attribute := range function.Attributes {
			if attribute.Name != "effects" {
				return semantic.File{}, fmt.Errorf("experimental state owners only admit effects attributes")
			}
		}
		copyModule.Functions[index].Body = &ast.Block{Span: function.Body.Span}
	}
	if roots != 1 {
		return semantic.File{}, fmt.Errorf("experimental state source requires exactly one transaction function")
	}
	file.Module = &copyModule
	return file, nil
}

func stateGraphHelperReachability(transactions []hir.StateTransaction, root hir.SymbolID) error {
	bySymbol := make(map[hir.SymbolID]hir.StateTransaction, len(transactions))
	for _, transaction := range transactions {
		bySymbol[transaction.Function] = transaction
	}
	seen, active := make(map[hir.SymbolID]bool), make(map[hir.SymbolID]bool)
	var visit func(hir.SymbolID) error
	visit = func(symbol hir.SymbolID) error {
		if active[symbol] {
			return fmt.Errorf("experimental state helper recursion is not admitted")
		}
		if seen[symbol] {
			return nil
		}
		seen[symbol], active[symbol] = true, true
		for _, block := range bySymbol[symbol].Blocks {
			for _, access := range block.Accesses {
				if access.Kind == hir.StateAccessCall {
					if err := visit(access.Callee); err != nil {
						return err
					}
				}
			}
		}
		active[symbol] = false
		return nil
	}
	if err := visit(root); err != nil {
		return err
	}
	if len(seen) != len(transactions) {
		return fmt.Errorf("experimental state helpers must be reachable from transaction")
	}
	return nil
}

func stateGraphCallProjection(skeleton semantic.File, transactions []hir.StateTransaction, owners map[string]hir.SymbolID) semantic.File {
	names := make(map[hir.SymbolID]string, len(owners))
	for name, symbol := range owners {
		names[symbol] = name
	}
	module := *skeleton.Module
	module.Functions = append([]ast.Function(nil), module.Functions...)
	for index, transaction := range transactions {
		body := &ast.Block{Span: module.Functions[index].Body.Span}
		for _, block := range transaction.Blocks {
			for _, access := range block.Accesses {
				if access.Kind != hir.StateAccessCall {
					continue
				}
				span := diagnostic.Span{StartByte: access.Span.StartByte, EndByte: access.Span.EndByte}
				body.Statements = append(body.Statements, ast.Statement{Kind: syntax.KindExprStmt, Span: span, Expression: &ast.Expression{
					Kind: syntax.KindCall, Span: span, Children: []*ast.Expression{{Kind: syntax.KindPath, Text: names[access.Callee], Span: span}},
				}})
			}
		}
		module.Functions[index].Body = body
	}
	skeleton.Module = &module
	return skeleton
}

func rebindStateGraphOwners(transactions []hir.StateTransaction, source []ast.Function, previous map[string]hir.SymbolID, current hir.Package) error {
	if len(current.Modules) != 1 || len(current.Modules[0].Functions) != len(transactions) {
		return fmt.Errorf("experimental state call projection lost semantic owners")
	}
	names := make(map[string]hir.SymbolID, len(transactions))
	for _, function := range current.Modules[0].Functions {
		names[function.Name] = function.Symbol
	}
	remap := make(map[hir.SymbolID]hir.SymbolID, len(transactions))
	for name, symbol := range previous {
		fresh, found := names[name]
		if !found {
			return fmt.Errorf("experimental state call projection lost owner %s", name)
		}
		remap[symbol] = fresh
	}
	for index := range transactions {
		transaction := &transactions[index]
		transaction.Package, transaction.Function = current.Modules[0].Package, names[source[index].Name]
		for b := range transaction.Blocks {
			for a := range transaction.Blocks[b].Accesses {
				access := &transaction.Blocks[b].Accesses[a]
				if access.Kind == hir.StateAccessCall {
					access.Callee = remap[access.Callee]
				}
			}
		}
	}
	return nil
}
