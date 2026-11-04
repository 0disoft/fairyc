package semantic

import (
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestResourceObservationsDoNotChangeAnalysis(t *testing.T) {
	for _, body := range []string{
		"discard 1;",
		"take(x); take(x);",
		"if (flag) { take(x); }",
	} {
		t.Run(body, func(t *testing.T) {
			files := borrowLiveFiles(t, "module main;resource R{cleanup{}}fn void take(R x){}fn void f(R x,bool flag){"+body+"}")
			ordinary := Analyze(files)
			observed := AnalyzeWithResourceStates(files)
			if len(ordinary.ResourceStates) != 0 || len(observed.ResourceStates) == 0 {
				t.Fatalf("observations: ordinary=%d observed=%d", len(ordinary.ResourceStates), len(observed.ResourceStates))
			}
			if body != "discard 1;" && len(ordinary.Diagnostics) == 0 {
				t.Fatal("ownership violation was not checked")
			}
			observed.ResourceStates = nil
			if !reflect.DeepEqual(ordinary, observed) {
				t.Fatal("observation mode changed semantic results")
			}
		})
	}
}

func TestResourceObservationsSkippedForLargeEnvironment(t *testing.T) {
	environment := make(resourceEnvironment, 256)
	for range 256 {
		environment[&symbol{}] = resourceValueState{state: ResourceLive}
	}
	a := &analyzer{}
	allocs := testing.AllocsPerRun(1, func() {
		for range 4096 {
			a.recordResourceStates(environment, hir.Span{}, "before")
			a.recordResourceStates(environment, hir.Span{}, "after")
		}
	})
	if allocs != 0 || len(a.resourceStates) != 0 {
		t.Fatalf("disabled observations allocated: %g allocations, %d facts", allocs, len(a.resourceStates))
	}
}
