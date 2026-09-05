package compiler

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/syntax"
)

// CheckExperimentalState admits one explicit source transaction. The ordinary Check entry point
// deliberately does not recognize these calls. All extracted effects re-enter state MIR verification.
func CheckExperimentalState(file semantic.File) (Result, error) {
	fail := func() (Result, error) {
		return Result{}, fmt.Errorf("experimental state source requires one void transaction with literal single-u64-cell operations and one final exit")
	}
	module := file.Module
	if module == nil || len(module.Functions) != 1 || len(module.States) != 1 || len(module.States[0].Fields) != 1 ||
		len(module.Imports)+len(module.Constants)+len(module.StaticAssertions)+len(module.Structs)+len(module.Resources)+len(module.Enums)+len(module.ForeignFunctions) != 0 {
		return fail()
	}
	field := module.States[0].Fields[0]
	if field.Kind != ast.StateFieldCell || field.ValueType.Path != "u64" {
		return fail()
	}
	function := module.Functions[0]
	if function.Name != "transaction" || function.Owner != "" || function.Public || function.Unsafe ||
		len(function.Parameters) != 0 || function.ResultType.Path != "void" || function.Body == nil || function.Body.Tail != nil {
		return fail()
	}
	transaction := hir.StateTransaction{}
	for index, statement := range function.Body.Statements {
		expression := statement.Expression
		if statement.Kind != syntax.KindExprStmt || expression == nil || expression.Kind != syntax.KindCall || len(expression.Children) == 0 ||
			expression.Children[0].Kind != syntax.KindPath || transaction.Exit != "" {
			return fail()
		}
		name := expression.Children[0].Text
		args := expression.Children[1:]
		access := hir.StateAccess{Field: field.Name, Span: hir.Span{Package: file.Package, Source: file.Path, StartByte: statement.Span.StartByte, EndByte: statement.Span.EndByte}}
		switch name {
		case "storage::read", "storage::write", "storage::delete":
			count := 1
			if name == "storage::write" {
				count = 2
			}
			if len(args) != count || args[0].Kind != syntax.KindLiteral || !strings.HasPrefix(args[0].Text, "\"") || sourceStateString(args[0]) != field.Name {
				return fail()
			}
			switch name {
			case "storage::read":
				access.Kind = hir.StateAccessCellRead
			case "storage::delete":
				access.Kind = hir.StateAccessCellDelete
			case "storage::write":
				if args[1].Kind != syntax.KindLiteral {
					return fail()
				}
				value, err := strconv.ParseUint(args[1].Text, 10, 64)
				if err != nil {
					return fail()
				}
				encoded, err := contractstate.U64.Encode(value)
				if err != nil {
					return fail()
				}
				access.Kind = hir.StateAccessCellWrite
				access.Value = hir.StateValue{TypeID: contractstate.U64.TypeID(), Encoded: encoded}
			}
			transaction.Accesses = append(transaction.Accesses, access)
		case "storage::commit", "storage::reject", "storage::trap":
			if index != len(function.Body.Statements)-1 {
				return fail()
			}
			if name == "storage::trap" {
				if len(args) != 1 || args[0].Kind != syntax.KindLiteral || !strings.HasPrefix(args[0].Text, "\"") {
					return fail()
				}
				transaction.Exit, transaction.TrapCategory = hir.StateTransactionTrap, sourceStateString(args[0])
			} else {
				if len(args) != 0 {
					return fail()
				}
				transaction.Exit = hir.StateTransactionExit(strings.TrimPrefix(name, "storage::"))
			}
		default:
			return fail()
		}
	}
	if transaction.Exit == "" {
		return fail()
	}
	// The ordinary owner has no unrepresented executable statements. Keep declarations and
	// attributes intact so normal semantic analysis remains the layout and effect authority.
	copyModule := *module
	copyFunction := function
	copyFunction.Body = &ast.Block{Span: function.Body.Span}
	copyModule.Functions = []ast.Function{copyFunction}
	file.Module = &copyModule
	analysis, err := AnalyzePackages(semantic.PackageInput{Files: []semantic.File{file}})
	if err != nil {
		return Result{}, err
	}
	if analysis.Semantic.CompletedThrough != semantic.PhaseFlow || len(analysis.Semantic.HIR.Modules) != 1 || len(analysis.Semantic.HIR.Modules[0].Functions) != 1 {
		return Result{Semantic: analysis.Semantic}, nil
	}
	owner := analysis.Semantic.HIR.Modules[0]
	transaction.Package, transaction.Function = owner.Package, owner.Functions[0].Symbol
	analysis.Semantic.HIR.StateTransactions = []hir.StateTransaction{transaction}
	return LowerAnalyzed(analysis)
}

func sourceStateString(expression *ast.Expression) string {
	decoded, err := hex.DecodeString(expression.Cooked)
	if err != nil {
		return ""
	}
	return string(decoded)
}
