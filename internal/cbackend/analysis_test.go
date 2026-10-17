package cbackend

import (
	"maps"
	"reflect"
	"sync"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestCompletedAnalysisSupportsIndependentWriters(t *testing.T) {
	file := source.NewFile("main.fy", []byte(`module main;
 @abilities(store, key) resource Coin { i32 amount; cleanup { assert(false); } }
 fn i32 main() {let Coin coin=Coin{.amount=42}; burn coin; return 42;}`))
	parsed, err := frontend.AnalyzeSyntax(file, source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("syntax: %v", err)
	}
	checked := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if len(checked.Diagnostics) != 0 || len(checked.Unsupported) != 0 {
		t.Fatalf("semantic: %+v", checked)
	}
	program, err := mir.Lower(checked.HIR)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []Profile{Stage0Profile(), Stage0WindowsMSVCProfile(), Stage0DropTraceProfile(), Stage0WindowsMSVCDropTraceProfile()} {
		t.Run(string(profile.Target)+"/"+profile.Instrumentation, func(t *testing.T) {
			plan, err := analyzeEmission(program, "main::main", profile, false)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := plan.features["resource-burn"]; !ok {
				t.Fatal("burn requirement must be known before output")
			}
			before := maps.Clone(plan.features)
			want, err := (&emitter{analysis: plan}).emitUnit()
			if err != nil {
				t.Fatal(err)
			}
			var wg sync.WaitGroup
			for range 4 {
				wg.Go(func() {
					got, err := (&emitter{analysis: plan}).emitUnit()
					if err != nil || !reflect.DeepEqual(got, want) {
						t.Errorf("independent writer changed unit: %v", err)
					}
				})
			}
			wg.Wait()
			if !maps.Equal(plan.features, before) {
				t.Fatal("output mutated analysis requirements")
			}
		})
	}
}
