package semantic

import (
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func scalarLoopFlow(t *testing.T, input string) (*borrowFlowAnalyzer, Result) {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || parsed.AST == nil {
		t.Fatalf("parse: %v", err)
	}
	a := newAnalyzer([]File{{Path: "main.fy", Module: parsed.AST}}, nil)
	a.resolve()
	result := a.typeCheck()
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
		t.Fatalf("invalid fixture: %+v %+v", a.diagnostics, a.unsupported)
	}
	m := a.moduleOrder[0]
	fn := &m.file.Module.Functions[len(m.file.Module.Functions)-1]
	f := &borrowFlowAnalyzer{analyzer: a, module: m, function: fn,
		liveAfter: make(map[*ast.Statement]borrowLiveMap), liveAfterExpression: make(map[*ast.Expression]borrowLiveMap), loanByKey: make(map[borrowLoanKey]int)}
	f.liveBlock(fn.Body, make(borrowLiveMap), borrowLiveContext{})
	return f, result
}

func TestScalarLoopBorrowProbeMatchesBaseline(t *testing.T) {
	for _, input := range []string{
		"module main;fn i32 f(){while(false){while(false){let i32 x=1;}}return 0;}",
		"module main;fn i32 f(i32 n){var i32 x=0;while(x<n){let i32 y=x+1;x=y;}return x;}",
		"module main;fn bool f(bool a){var bool x=a;while(x){x=false;}return x;}",
		"module main;fn i32 f(){var i32 x=0;while(true){if(x>2){break;}x=x+1;continue;}return x;}",
		"module main;fn void observe(ref i32 p){}fn i32 f(){var i32 x=1;let ref i32 p=ref x;while(false){x=2;}observe(p);return x;}",
		"module main;fn void observe(ref i32 p){}fn i32 f(){var i32 x=1;while(false){let ref i32 p=ref x;observe(p);x=2;}return x;}",
		"module main;fn i32 f(ref i32 p){while(false){let i32 x=1;}return *p;}",
		"module main;fn i32 f(){while(true){return 1;}return 0;}",
	} {
		var results [2]Result
		for mode := range results {
			f, result := scalarLoopFlow(t, input)
			if mode == 1 {
				// Force the old convergence path for every typed expression without
				// a production switch or a second implementation of the algorithm.
				f.neutralScalars = make(map[*ast.Expression]bool)
				loops := 0
				for expression := range f.analyzer.expressionTypes {
					f.neutralScalars[expression] = false
					if expression.Kind == syntax.KindWhile {
						loops++
						if f.canSkipScalarLoopProbe(expression, nil) {
							t.Fatal("baseline probe was not restored")
						}
					}
				}
				if loops == 0 {
					t.Fatal("baseline has no typed loop witness")
				}
			}
			f.evaluateBlock(f.function.Body, f.parameterEnvironment(), true)
			for _, loan := range f.loans {
				fact := loan.fact
				fact.Function = f.analyzer.functions[f.function].id
				f.analyzer.borrows = append(f.analyzer.borrows, fact)
			}
			f.analyzer.finish()
			result.Diagnostics, result.Unsupported, result.Borrows = f.analyzer.diagnostics, f.analyzer.unsupported, f.analyzer.borrows
			results[mode] = result
		}
		if !reflect.DeepEqual(results[0], results[1]) {
			t.Fatalf("borrow result changed for %s", input)
		}
	}
}

func TestScalarLoopBorrowProbeGuards(t *testing.T) {
	f, _ := scalarLoopFlow(t, "module main;fn i32 f(){while(false){let i32 x=1;}return 0;}")
	loop := f.function.Body.Statements[0].Expression
	if !f.canSkipScalarLoopProbe(loop, make(borrowEnvironment)) {
		t.Fatal("closed scalar witness did not select the fast path")
	}
	f.summarizing = true
	if f.canSkipScalarLoopProbe(loop, nil) {
		t.Fatal("summary guard")
	}
	f.summarizing = false
	f.callBorrows = []borrowValueState{{}}
	if f.canSkipScalarLoopProbe(loop, nil) {
		t.Fatal("temporary borrow guard")
	}
	f.callBorrows = nil
	f.loans = []borrowLoanRecord{{}}
	if f.canSkipScalarLoopProbe(loop, nil) {
		t.Fatal("loan guard")
	}
	f.loans = nil
	if f.canSkipScalarLoopProbe(loop, borrowEnvironment{nil: {}}) {
		t.Fatal("environment guard")
	}
	for _, body := range []string{"break;", "continue;", "return 1;", "if(true){}", "let ref i32 p=ref x;", "x+=1;"} {
		g, _ := scalarLoopFlow(t, "module main;fn i32 f(){var i32 x=0;while(true){"+body+"}return x;}")
		if g.canSkipScalarLoopProbe(g.function.Body.Statements[1].Expression, nil) {
			t.Fatalf("unsafe fast-path admission: %s", body)
		}
	}
}
