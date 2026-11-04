package genericsource

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/syntax"
)

func materializationBudgetFunction(name string, generic bool) ast.Function {
	function := ast.Function{
		Name:       name,
		Span:       diagnostic.Span{StartByte: 1, EndByte: 4},
		Attributes: []ast.Attribute{{Name: "test", Arguments: []ast.AttributeArgument{{Raw: "1", Span: diagnostic.Span{StartByte: 1, EndByte: 2}}}}},
		Parameters: []ast.Parameter{{Name: "value", Span: diagnostic.Span{StartByte: 2, EndByte: 3}, Type: ast.TypeReference{
			Kind: ast.TypeKindNamed, Path: "Box", Arguments: []ast.TypeReference{{Kind: ast.TypeKindNamed, Path: "T"}},
		}}},
		ResultType: ast.TypeReference{Kind: ast.TypeKindReference, Element: &ast.TypeReference{
			Kind: ast.TypeKindArray, Element: &ast.TypeReference{Kind: ast.TypeKindNamed, Path: "T"},
			Length: &ast.TypeLength{Text: "4", Span: diagnostic.Span{StartByte: 3, EndByte: 4}},
		}},
		Body: &ast.Block{Statements: []ast.Statement{{
			Binding: &ast.Binding{
				Name: "copy", Span: diagnostic.Span{StartByte: 3, EndByte: 4},
				Type: &ast.TypeReference{Kind: ast.TypeKindSlice, Element: &ast.TypeReference{Kind: ast.TypeKindNamed, Path: "T"}},
				Expression: &ast.Expression{Kind: syntax.KindCall, Children: []*ast.Expression{
					{Kind: syntax.KindPath, Text: "value"},
					{Kind: syntax.KindPath, Text: "value"},
				}},
			},
		}}},
	}
	if generic {
		function.TypeParameters = []string{"T"}
	}
	function.Body.Tail = &ast.Expression{Kind: syntax.KindClosure, Closure: &ast.Closure{
		Captures: []ast.Parameter{{Name: "copy", Type: ast.TypeReference{Kind: ast.TypeKindNamed, Path: "T"}}},
		Function: ast.Function{Parameters: []ast.Parameter{{Name: "inner", Type: ast.TypeReference{Kind: ast.TypeKindNamed, Path: "T"}}},
			ResultType: ast.TypeReference{Kind: ast.TypeKindNamed, Path: "T"}},
	}}
	return function
}

func TestMaterializationBudgetBelowAtAndOver(t *testing.T) {
	plain := materializationBudgetFunction("main", false)
	template := materializationBudgetFunction("id", true)
	module := &ast.Module{
		Path: "main", Span: diagnostic.Span{StartByte: 0, EndByte: 16}, SourceSize: 16,
		Functions: []ast.Function{plain, template},
	}
	x := &expander{module: module, owner: "owner"}
	declaration := x.id("id")
	plan := genericplan.Plan{Specializations: []genericplan.Specialization{
		{Declaration: declaration}, {Declaration: declaration}, {Declaration: declaration},
	}}
	before, err := json.Marshal(module)
	if err != nil {
		t.Fatal(err)
	}
	work, err := materializedASTWork([]*expander{x}, plan)
	if err != nil || work == 0 {
		t.Fatalf("materialized work = %d, error = %v", work, err)
	}
	plainNodes, err := materializedFunctionNodes(plain)
	if err != nil {
		t.Fatal(err)
	}
	templateNodes, err := materializedFunctionNodes(template)
	if err != nil {
		t.Fatal(err)
	}
	if work != plainNodes+3*templateNodes {
		t.Fatalf("materialized work = %d, want %d", work, plainNodes+3*templateNodes)
	}
	if work >= defaultMaterializedASTNodeBudget {
		t.Fatalf("fixture work %d exceeds the production budget", work)
	}
	if err := checkMaterializationBudget([]*expander{x}, plan, work+1); err != nil {
		t.Fatalf("below budget rejected: %v", err)
	}
	if err := checkMaterializationBudget([]*expander{x}, plan, work); err != nil {
		t.Fatalf("exact budget rejected: %v", err)
	}
	err = checkMaterializationBudget([]*expander{x}, plan, work-1)
	var source *SourceError
	if !errors.As(err, &source) || genericplan.ErrorCode(err) != genericplan.CodePlanningBudgetExceeded {
		t.Fatalf("over budget error = %v", err)
	}
	after, err := json.Marshal(module)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("budget preflight mutated the source AST")
	}
}

func TestMaterializedNodeWorkZeroAndOverflow(t *testing.T) {
	if _, err := addMaterializedNodeWork(0, 1, 1, 0); genericplan.ErrorCode(err) != genericplan.CodePlanningBudgetExceeded {
		t.Fatalf("zero budget error = %v", err)
	}
	if total, err := addMaterializedNodeWork(7, 0, ^uint64(0), 8); err != nil || total != 7 {
		t.Fatalf("zero nodes = %d, error = %v", total, err)
	}
	if total, err := addMaterializedNodeWork(7, 1, 0, 8); err != nil || total != 7 {
		t.Fatalf("zero copies = %d, error = %v", total, err)
	}
	if _, err := addMaterializedNodeWork(0, 1<<63, 2, ^uint64(0)); genericplan.ErrorCode(err) != genericplan.CodePlanningBudgetExceeded {
		t.Fatalf("multiplication overflow error = %v", err)
	}
	if _, err := addMaterializedNodeWork(^uint64(0)-1, 2, 1, ^uint64(0)); genericplan.ErrorCode(err) != genericplan.CodePlanningBudgetExceeded {
		t.Fatalf("addition overflow error = %v", err)
	}
}
