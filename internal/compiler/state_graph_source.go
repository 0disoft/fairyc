package compiler

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/syntax"
)

// These are the existing state-MIR graph bounds, enforced before constructing
// source graphs as well as independently at the state-MIR verification boundary.
const stateSourceMaxBlocks, stateSourceMaxOperations = 64, 1024

// CheckExperimentalStateGraph selects the explicit graph source admission.
// Neither ordinary Check nor the original single-cell admission is widened.
func CheckExperimentalStateGraph(file semantic.File) (Result, error) {
	module := file.Module
	if module == nil || len(module.Functions) != 1 || len(module.States) != 1 ||
		len(module.Imports)+len(module.Constants)+len(module.StaticAssertions)+len(module.Structs)+len(module.Resources)+len(module.Enums)+len(module.ForeignFunctions) != 0 {
		return Result{}, fmt.Errorf("experimental state graph requires one state layout and one transaction function")
	}
	function := module.Functions[0]
	if function.Name != "transaction" || function.Owner != "" || function.Public || function.Unsafe ||
		len(function.TypeParameters)+len(function.Parameters) != 0 || function.ResultType.Kind != ast.TypeKindNamed ||
		function.ResultType.Path != "void" || len(function.ResultType.Arguments) != 0 || function.Body == nil {
		return Result{}, fmt.Errorf("experimental state graph requires a private safe parameterless void transaction")
	}
	for _, attribute := range function.Attributes {
		if attribute.Name != "effects" {
			return Result{}, fmt.Errorf("experimental state graph only admits transaction effects attributes")
		}
	}
	// Normal semantics owns the schema, package, signature and declared effects.
	// Every executable source statement is admitted below into the state side
	// table; no body, binding or caller-owned AST slice is mutated or skipped.
	copyModule, copyFunction := *module, function
	copyFunction.Body = &ast.Block{Span: function.Body.Span}
	copyModule.Functions = []ast.Function{copyFunction}
	copyFile := file
	copyFile.Module = &copyModule
	analysis, err := AnalyzePackages(semantic.PackageInput{Files: []semantic.File{copyFile}})
	if err != nil {
		return Result{}, err
	}
	checked := analysis.Semantic
	if checked.CompletedThrough != semantic.PhaseFlow || checked.FailedPhase != semantic.PhaseNone ||
		len(checked.Diagnostics)+len(checked.Unsupported) != 0 {
		return Result{Semantic: checked}, nil
	}
	if len(checked.HIR.Modules) != 1 || len(checked.HIR.Modules[0].Functions) != 1 || len(checked.HIR.StateLayouts) != 1 {
		return Result{}, fmt.Errorf("experimental state graph requires one semantic owner and layout")
	}
	owner := checked.HIR.Modules[0]
	lower := stateGraphSource{file: file, fields: make(map[string]hir.StateField)}
	for _, field := range checked.HIR.StateLayouts[0].Fields {
		lower.fields[field.Name] = field
	}
	_, _, err = lower.newBlock(nil)
	if err != nil {
		return Result{}, err
	}
	if end, _, err := lower.block(function.Body, 0, nil); err != nil {
		return Result{}, err
	} else if end >= 0 {
		return Result{}, lower.fail(function.Body.Span, "every transaction path must commit, reject or trap")
	}
	analysis.Semantic.HIR.StateTransactions = []hir.StateTransaction{{
		Package: owner.Package, Function: owner.Functions[0].Symbol, Entry: 0,
		Blocks: lower.blocks, Span: lower.span(function.Body.Span),
	}}
	// Re-enter the existing atomic ordinary/state publication and independent
	// state-MIR checks, including effects, types, edges, cost and codec validity.
	return LowerAnalyzed(analysis)
}

type stateGraphBinding struct {
	name, typeID string
	value, found hir.StateValueID
}

type stateGraphSource struct {
	file       semantic.File
	fields     map[string]hir.StateField
	blocks     []hir.StateBlock
	values     hir.StateValueID
	operations int
}

func (lower *stateGraphSource) span(span diagnostic.Span) hir.Span {
	packageID := lower.file.Package
	if packageID == 0 {
		packageID = hir.DefaultPackage
	}
	return hir.Span{Package: packageID, Source: lower.file.Path, StartByte: span.StartByte, EndByte: span.EndByte}
}

func (lower *stateGraphSource) fail(span diagnostic.Span, detail string) error {
	return fmt.Errorf("experimental state graph at %s:%d: %s", lower.file.Path, span.StartByte, detail)
}

func (lower *stateGraphSource) value() hir.StateValueID {
	lower.values++
	return lower.values
}

func (lower *stateGraphSource) newBlock(environment []stateGraphBinding) (int, []stateGraphBinding, error) {
	if len(lower.blocks) >= stateSourceMaxBlocks {
		return -1, nil, fmt.Errorf("experimental state graph exceeds 64 blocks")
	}
	id := len(lower.blocks)
	block := hir.StateBlock{ID: hir.StateBlockID(id)}
	copy := append([]stateGraphBinding(nil), environment...)
	for index := range copy {
		copy[index].value = lower.value()
		block.Parameters = append(block.Parameters, hir.StateBlockParameter{ID: copy[index].value, TypeID: copy[index].typeID})
		if copy[index].found != 0 {
			copy[index].found = lower.value()
			block.Parameters = append(block.Parameters, hir.StateBlockParameter{ID: copy[index].found, TypeID: stateSourceBoolType})
		}
	}
	lower.blocks = append(lower.blocks, block)
	return id, copy, nil
}

func stateGraphArguments(environment []stateGraphBinding) []hir.StateValueID {
	arguments := make([]hir.StateValueID, 0, len(environment)*2)
	for _, binding := range environment {
		arguments = append(arguments, binding.value)
		if binding.found != 0 {
			arguments = append(arguments, binding.found)
		}
	}
	return arguments
}

func (lower *stateGraphSource) block(body *ast.Block, current int, environment []stateGraphBinding) (int, []stateGraphBinding, error) {
	if body == nil {
		return -1, nil, fmt.Errorf("experimental state graph requires a block")
	}
	for _, statement := range body.Statements {
		if current < 0 {
			return -1, nil, lower.fail(statement.Span, "statement follows a closed transaction path")
		}
		var err error
		switch statement.Kind {
		case syntax.KindBinding:
			environment, err = lower.binding(statement, current, environment)
		case syntax.KindIf:
			current, environment, err = lower.branch(statement.Expression, current, environment)
		case syntax.KindExprStmt:
			if statement.Expression != nil && statement.Expression.Kind == syntax.KindIf {
				current, environment, err = lower.branch(statement.Expression, current, environment)
			} else {
				current, err = lower.statement(statement, current, environment)
			}
		default:
			err = lower.fail(statement.Span, "only immutable storage reads, storage operations and if statements are admitted")
		}
		if err != nil {
			return -1, nil, err
		}
	}
	// The ordinary parser represents a final if as a tail expression. Admit
	// that same statement-form control flow, never an arbitrary value tail.
	if body.Tail != nil {
		if current < 0 || body.Tail.Kind != syntax.KindIf {
			return -1, nil, lower.fail(body.Tail.Span, "only a reachable structured if may be a block tail")
		}
		return lower.branch(body.Tail, current, environment)
	}
	return current, environment, nil
}

func (lower *stateGraphSource) branch(expression *ast.Expression, current int, environment []stateGraphBinding) (int, []stateGraphBinding, error) {
	if expression == nil || expression.Kind != syntax.KindIf || len(expression.Children) < 2 || len(expression.Children) > 3 || expression.Children[1] == nil || expression.Children[1].Block == nil {
		return -1, nil, fmt.Errorf("experimental state graph requires a structured if with block arms")
	}
	condition, typeID, err := lower.reference(expression.Children[0], environment)
	if err != nil || typeID != stateSourceBoolType {
		return -1, nil, lower.fail(expression.Span, "if condition requires a read bool or storage::found(read)")
	}
	trueID, trueEnv, err := lower.newBlock(environment)
	if err != nil {
		return -1, nil, err
	}
	falseID, falseEnv, err := lower.newBlock(environment)
	if err != nil {
		return -1, nil, err
	}
	arguments := stateGraphArguments(environment)
	lower.blocks[current].Terminator = hir.StateTerminator{Kind: hir.StateTerminatorBranch, Condition: condition,
		TrueTarget: hir.StateBlockID(trueID), FalseTarget: hir.StateBlockID(falseID), TrueArguments: arguments, FalseArguments: append([]hir.StateValueID(nil), arguments...), Cost: 1}
	trueEnd, trueOut, err := lower.block(expression.Children[1].Block, trueID, trueEnv)
	if err != nil {
		return -1, nil, err
	}
	falseBody := &ast.Block{}
	if len(expression.Children) == 3 {
		if expression.Children[2] == nil || expression.Children[2].Block == nil {
			return -1, nil, lower.fail(expression.Span, "else requires a block")
		}
		falseBody = expression.Children[2].Block
	}
	falseEnd, falseOut, err := lower.block(falseBody, falseID, falseEnv)
	if err != nil || trueEnd < 0 && falseEnd < 0 {
		return -1, nil, err
	}
	join, joined, err := lower.newBlock(environment)
	if err != nil {
		return -1, nil, err
	}
	for _, end := range []struct {
		id  int
		env []stateGraphBinding
	}{{trueEnd, trueOut}, {falseEnd, falseOut}} {
		if end.id >= 0 {
			lower.blocks[end.id].Terminator = hir.StateTerminator{Kind: hir.StateTerminatorJump, Target: hir.StateBlockID(join), Arguments: stateGraphArguments(end.env[:len(environment)]), Cost: 1}
		}
	}
	return join, joined, nil
}
