package semantic

import (
	"fmt"
	"strings"
	"testing"
)

func TestBorrowLoopKeepsEntryAndHeaderEvidence(t *testing.T) {
	for _, body := range []string{
		"selected=q;",
		"selected=q;continue;",
		"selected=q;break;",
		"if(flag){selected=q;break;}selected=p;continue;",
		"while(flag){selected=q;break;}break;",
	} {
		t.Run(body, func(t *testing.T) {
			f, _ := scalarLoopFlow(t, "module main;fn i32 f(ref i32 p,ref i32 q,bool flag){var ref i32 selected=p;while(flag){"+body+"}return *selected;}")
			environment := f.parameterEnvironment()
			if !f.evaluateStatement(&f.function.Body.Statements[0], environment) {
				t.Fatal("binding did not continue")
			}
			before := make(borrowEnvironment, len(environment))
			for binding, value := range environment {
				before[binding] = cloneBorrowValue(value)
			}
			// Keep the actual shared evidence too: mutating an inherited slice
			// would otherwise be hidden by a deep copy of the expected value.
			retained := cloneBorrowEnvironment(environment)
			if _, reachable := f.evaluateWhile(f.function.Body.Statements[1].Expression, environment); !reachable {
				t.Fatal("loop lost its zero-iteration exit")
			}
			if !sameBorrowEnvironment(before, retained) {
				t.Fatal("loop changed retained entry evidence")
			}
			selected := f.analyzer.bindingSymbols[f.function.Body.Statements[0].Binding]
			value := environment[selected]
			if !value.known || len(value.origins) != 2 || value.origins[0].name != "Param(0)" || value.origins[1].name != "Param(1)" {
				t.Fatalf("zero/back/break edges lost an origin: %+v", value)
			}
			if len(f.loops) != 0 || f.probeDepth != 0 || len(f.analyzer.diagnostics) != 0 || len(f.analyzer.unsupported) != 0 {
				t.Fatalf("loop state was not restored: depth=%d probe=%d diagnostics=%v unsupported=%v", len(f.loops), f.probeDepth, f.analyzer.diagnostics, f.analyzer.unsupported)
			}
		})
	}
}

func TestBorrowLoopFailedConditionDoesNotPublishState(t *testing.T) {
	f, _ := scalarLoopFlow(t, "module main;fn i32 f(ref i32 p,ref i32 q,bool flag){var ref i32 selected=p;while({selected=q;flag}){}return *selected;}")
	environment := f.parameterEnvironment()
	f.evaluateStatement(&f.function.Body.Statements[0], environment)
	before := cloneBorrowEnvironment(environment)
	// The condition writes selected before it runs out of work at flag. Its
	// incomplete state must never replace the caller's incoming environment.
	f.analyzer.borrowWork = nil
	f.analyzer.borrowWorkLimit = len(environment) + 3
	if _, reachable := f.evaluateWhile(f.function.Body.Statements[1].Expression, environment); reachable {
		t.Fatal("incomplete condition was accepted")
	}
	if !sameBorrowEnvironment(environment, before) || len(f.loops) != 0 || f.probeDepth != 0 {
		t.Fatal("failed condition leaked speculative state")
	}
}

func BenchmarkBorrowLoopEnvironment(b *testing.B) {
	for _, bindings := range []int{64, 256} {
		parameters := make([]string, bindings)
		for index := range parameters {
			parameters[index] = fmt.Sprintf("ref i32 p%d", index)
		}
		for _, body := range []struct{ name, text string }{{"scalar", "let i32 x=1;"}, {"probe", "discard 0;"}} {
			b.Run(fmt.Sprintf("%s/bindings_%d", body.name, bindings), func(b *testing.B) {
				f, _ := scalarLoopFlow(b, "module main;fn i32 f("+strings.Join(parameters, ",")+"){while(false){"+body.text+"}return *p0;}")
				environment := f.parameterEnvironment()
				loop := f.function.Body.Statements[0].Expression
				budget := f.analyzer.borrowBudget(f.function)
				b.ReportAllocs()
				for b.Loop() {
					budget.remaining = defaultBorrowWorkLimit
					if _, reachable := f.evaluateWhile(loop, environment); !reachable {
						b.Fatal("loop failed")
					}
				}
			})
		}
	}
}
