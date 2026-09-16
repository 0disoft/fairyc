package semantic

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestScalarResultBackEdgeMatchesBaseline(t *testing.T) {
	inputs := []string{
		"module main;fn i32 f(){while(false){while(false){let i32 x=1;}}return 0;}",
		"module main;fn i32 f(i32 n){var i32 x=0;while(x<n){let i32 y=x+1;x=y;}return x;}",
		"module main;fn bool f(bool a){var bool x=a;while(x){x=false;}return x;}",
		"module main;fn i32 f(){var i32 x=0;while(true){if(x>2){break;}x=x+1;continue;}return x;}",
		"module main;fn i32 f(){while(true){return 1;}return 0;}",
		"module main;fn i32 f(Result<i32,i32> p){while(false){let i32 x=1;}return 0;}",
	}
	for _, name := range []string{"fail/break-pending", "fail/continue-pending", "fail/continue-overwrite", "fail/loop-overwrite", "fail/early-return-unhandled", "pass/handled-reassignment", "pass/propagate-result"} {
		data, err := os.ReadFile(filepath.Join("../../spec/fixtures/result/cases", name, "src/main.fy"))
		if err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, string(data))
	}
	for index, input := range inputs {
		var results [2]Result
		for mode := range results {
			f, result := scalarLoopFlow(t, input)
			a := f.analyzer
			if mode == 1 {
				a.resultNeutralScalars = make(map[*ast.Expression]bool)
				for expression := range a.expressionTypes {
					a.resultNeutralScalars[expression] = false
					if expression.Kind == syntax.KindWhile && a.canSkipScalarResultBackEdge(expression, nil) {
						t.Fatal("baseline not restored")
					}
				}
			}
			a.flowCheck()
			a.finish()
			result.Diagnostics, result.Unsupported = a.diagnostics, a.unsupported
			result.ResultHandling, result.Borrows = a.resultHandling, a.borrows
			results[mode] = result
		}
		if !reflect.DeepEqual(results[0], results[1]) {
			t.Fatalf("flow result changed for input %d", index)
		}
		if index == 5 && len(results[0].Diagnostics) == 0 {
			t.Fatal("pending Result diagnostic disappeared")
		}
	}
}

func TestScalarResultBackEdgeGuards(t *testing.T) {
	f, _ := scalarLoopFlow(t, "module main;fn i32 f(){while(false){let i32 x=1;}return 0;}")
	a, loop := f.analyzer, f.function.Body.Statements[0].Expression
	if !a.canSkipScalarResultBackEdge(loop, nil) {
		t.Fatal("scalar witness rejected")
	}
	if a.canSkipScalarResultBackEdge(loop, resultEnvironment{nil: {pending: true}}) {
		t.Fatal("pending environment admitted")
	}
	if a.canSkipScalarResultBackEdge(loop, resultEnvironment{nil: {}}) {
		t.Fatal("handled environment admitted")
	}
	copy := *loop
	if a.canSkipScalarResultBackEdge(&copy, nil) {
		t.Fatal("untyped synthesized loop admitted")
	}
	for _, body := range []string{"break;", "continue;", "return 1;", "if(true){}", "x+=1;", "let Result<i32,i32> p=.ok(1);", "let ref i32 p=ref x;"} {
		g, _ := scalarLoopFlow(t, "module main;fn i32 f(){var i32 x=0;while(true){"+body+"}return x;}")
		if g.analyzer.canSkipScalarResultBackEdge(g.function.Body.Statements[1].Expression, nil) {
			t.Fatalf("non-scalar body admitted: %s", body)
		}
	}
}
