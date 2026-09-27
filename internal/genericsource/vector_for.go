package genericsource

import (
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
)

// activateSDKVectorLoops exposes prepared calls to the bounded generic planner
// only when the exact std::vec import resolves to an authenticated SDK owner.
// The semantic source gate remains closed until it independently validates the
// written loop, type identity and owner-bound borrow.
func (x *expander) activateSDKVectorLoops() error {
	if x == nil || x.module == nil {
		return nil
	}
	for _, imported := range x.module.Imports {
		if imported.Path != "std::vec" {
			continue
		}
		provider := x.imports[imported.Alias]
		if provider == nil || provider.module == nil || provider.module.Path != "std::vec" || !provider.allowStorage {
			return nil
		}
		var detached *ast.Module
		for i, function := range x.module.Functions {
			if !hasVectorForCandidate(function.Body) {
				continue
			}
			if detached == nil {
				copy := *x.module
				copy.Functions = slices.Clone(x.module.Functions)
				detached = &copy
			}
			clone, err := (transform{}).function(function)
			if err != nil {
				return err
			}
			ast.PrepareSDKVectorLoopBody(clone.Body, imported.Alias)
			activateVectorForBlock(clone.Body)
			detached.Functions[i] = clone
		}
		if detached != nil {
			x.module = detached
		}
		return nil
	}
	return nil
}

func hasVectorForCandidate(block *ast.Block) bool {
	if block == nil {
		return false
	}
	for i := range block.Statements {
		if hasVectorForStatement(&block.Statements[i]) {
			return true
		}
	}
	return hasVectorForExpression(block.Tail)
}

func hasVectorForStatement(statement *ast.Statement) bool {
	if statement == nil {
		return false
	}
	if statement.ByteFor != nil && (statement.ByteFor.Type.Kind == ast.TypeKindReference &&
		!statement.ByteFor.Type.Mutable && statement.ByteFor.Type.Element != nil &&
		statement.ByteFor.Body != nil && ast.VectorForEligibleView(statement.ByteFor.View) ||
		hasVectorForCandidate(statement.ByteFor.Body)) {
		return true
	}
	if statement.For != nil && (hasVectorForStatement(statement.For.Initializer) ||
		hasVectorForExpression(statement.For.Condition) ||
		hasVectorForStatement(statement.For.Post) ||
		hasVectorForCandidate(statement.For.Body)) {
		return true
	}
	return hasVectorForExpression(statement.Target) || hasVectorForExpression(statement.Expression) ||
		statement.Binding != nil && hasVectorForExpression(statement.Binding.Expression)
}

func hasVectorForExpression(expression *ast.Expression) bool {
	if expression == nil {
		return false
	}
	if hasVectorForCandidate(expression.Block) {
		return true
	}
	for _, child := range expression.Children {
		if hasVectorForExpression(child) {
			return true
		}
	}
	return false
}

func activateVectorForBlock(block *ast.Block) {
	if block == nil {
		return
	}
	for i := range block.Statements {
		activateVectorForStatement(&block.Statements[i])
	}
	activateVectorForExpression(block.Tail)
}

func activateVectorForStatement(statement *ast.Statement) {
	if statement == nil {
		return
	}
	if statement.ByteFor != nil && statement.ByteFor.PreparedVector != nil && statement.Expression == nil {
		statement.Expression = statement.ByteFor.PreparedVector
	}
	if statement.ByteFor != nil && statement.Expression == nil {
		activateVectorForBlock(statement.ByteFor.Body)
	}
	if statement.For != nil {
		activateVectorForStatement(statement.For.Initializer)
		activateVectorForExpression(statement.For.Condition)
		activateVectorForStatement(statement.For.Post)
		activateVectorForBlock(statement.For.Body)
	}
	if statement.Binding != nil {
		activateVectorForExpression(statement.Binding.Expression)
	}
	activateVectorForExpression(statement.Target)
	activateVectorForExpression(statement.Expression)
}

func activateVectorForExpression(expression *ast.Expression) {
	if expression == nil {
		return
	}
	activateVectorForBlock(expression.Block)
	for _, child := range expression.Children {
		activateVectorForExpression(child)
	}
}
